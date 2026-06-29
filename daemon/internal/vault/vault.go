// Package vault — Cryptomator-compatible vault (format 8, SIV_GCM).
// Reads real Cryptomator vaults created by the official app.
package vault

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/scrypt"
)

// ErrWrongPassword is returned when the password is incorrect (AES-KW integrity check failed).
var ErrWrongPassword = errors.New("wrong vault password")

// Vault is an unlocked Cryptomator vault (format 8, SIV_GCM).
type Vault struct {
	encKey []byte // 32B — AES-GCM content encryption + header key
	macKey []byte // 32B — SIV MAC half + JWT signature
}

// masterkeyFile is the JSON structure of masterkey.cryptomator.
type masterkeyFile struct {
	Version          int    `json:"version"`
	ScryptSalt       string `json:"scryptSalt"`
	ScryptCostParam  int    `json:"scryptCostParam"`
	ScryptBlockSize  int    `json:"scryptBlockSize"`
	PrimaryMasterKey string `json:"primaryMasterKey"`
	HmacMasterKey    string `json:"hmacMasterKey"`
	VersionMac       string `json:"versionMac"`
}

// Open opens a Cryptomator vault at vaultDir using the given password.
// Reads masterkey.cryptomator and vault.cryptomator from the local filesystem.
// Wrong password → ErrWrongPassword.
func Open(vaultDir, password string) (*Vault, error) {
	mkData, err := os.ReadFile(filepath.Join(vaultDir, "masterkey.cryptomator"))
	if err != nil {
		return nil, fmt.Errorf("vault: reading masterkey.cryptomator: %w", err)
	}
	jwtData, err := os.ReadFile(filepath.Join(vaultDir, "vault.cryptomator"))
	if err != nil {
		return nil, fmt.Errorf("vault: reading vault.cryptomator: %w", err)
	}
	return deriveVault(mkData, jwtData, password)
}

// OpenWithKeys opens a vault directly from master keys recovered from a recovery phrase
// (see RecoveryToKeys), bypassing the password. It verifies the keys against
// vault.cryptomator, so a phrase from a different vault is rejected.
func OpenWithKeys(vaultDir string, encKey, macKey []byte) (*Vault, error) {
	jwtData, err := os.ReadFile(filepath.Join(vaultDir, "vault.cryptomator"))
	if err != nil {
		return nil, fmt.Errorf("vault: reading vault.cryptomator: %w", err)
	}
	if err := verifyVaultJWT(strings.TrimSpace(string(jwtData)), encKey, macKey); err != nil {
		return nil, fmt.Errorf("vault: recovery phrase does not match this vault: %w", err)
	}
	return &Vault{encKey: encKey, macKey: macKey}, nil
}

// deriveVault unwraps the master keys from masterkey.cryptomator bytes and verifies
// vault.cryptomator bytes. Wrong password → ErrWrongPassword.
func deriveVault(mkData, jwtData []byte, password string) (*Vault, error) {
	// 1. Parse masterkey.cryptomator
	var mk masterkeyFile
	if err := json.Unmarshal(mkData, &mk); err != nil {
		return nil, fmt.Errorf("vault: parsing masterkey.cryptomator: %w", err)
	}

	// 2. Decode salt and wrapped keys
	salt, err := base64.StdEncoding.DecodeString(mk.ScryptSalt)
	if err != nil {
		return nil, fmt.Errorf("vault: decoding scryptSalt: %w", err)
	}
	wrappedEnc, err := base64.StdEncoding.DecodeString(mk.PrimaryMasterKey)
	if err != nil {
		return nil, fmt.Errorf("vault: decoding primaryMasterKey: %w", err)
	}
	wrappedMac, err := base64.StdEncoding.DecodeString(mk.HmacMasterKey)
	if err != nil {
		return nil, fmt.Errorf("vault: decoding hmacMasterKey: %w", err)
	}

	// 3. Derive KEK via scrypt
	kek, err := scrypt.Key([]byte(password), salt, mk.ScryptCostParam, mk.ScryptBlockSize, 1, 32)
	if err != nil {
		return nil, fmt.Errorf("vault: scrypt: %w", err)
	}

	// 4. Unwrap keys (AES-KW). Wrong password → IV mismatch
	encKey, err := aesKWUnwrap(kek, wrappedEnc)
	if err != nil {
		return nil, ErrWrongPassword
	}
	macKey, err := aesKWUnwrap(kek, wrappedMac)
	if err != nil {
		return nil, ErrWrongPassword
	}

	// 5. Verify vault.cryptomator (compact JWT, alg=HS256)
	if err := verifyVaultJWT(strings.TrimSpace(string(jwtData)), encKey, macKey); err != nil {
		return nil, fmt.Errorf("vault: verifying vault.cryptomator: %w", err)
	}

	return &Vault{encKey: encKey, macKey: macKey}, nil
}

// Create generates a new Cryptomator vault at vaultDir: encKey/macKey (CSPRNG), writes
// masterkey.cryptomator (scrypt KEK + AES-KW wrap) and vault.cryptomator (JWT HS256).
// Returns the opened Vault.
func Create(vaultDir, password string) (*Vault, error) {
	// 1. Generate keys
	encKey := make([]byte, 32)
	macKey := make([]byte, 32)
	scryptSalt := make([]byte, 8)
	if _, err := rand.Read(encKey); err != nil {
		return nil, fmt.Errorf("vault: generating encKey: %w", err)
	}
	if _, err := rand.Read(macKey); err != nil {
		return nil, fmt.Errorf("vault: generating macKey: %w", err)
	}
	if _, err := rand.Read(scryptSalt); err != nil {
		return nil, fmt.Errorf("vault: generating scryptSalt: %w", err)
	}

	// 2. KEK = scrypt(password, salt, N=32768, r=8, p=1, dkLen=32)
	kek, err := scrypt.Key([]byte(password), scryptSalt, 32768, 8, 1, 32)
	if err != nil {
		return nil, fmt.Errorf("vault: scrypt: %w", err)
	}

	// 3. Wrap keys (AES-KW)
	wrappedEnc, err := aesKWWrap(kek, encKey)
	if err != nil {
		return nil, fmt.Errorf("vault: wrap encKey: %w", err)
	}
	wrappedMac, err := aesKWWrap(kek, macKey)
	if err != nil {
		return nil, fmt.Errorf("vault: wrap macKey: %w", err)
	}

	// 4. versionMac = HMAC-SHA256(macKey, BE32(999))
	vm := vaultVersionMac(macKey, 999)

	// 5. Write masterkey.cryptomator (field order matches the reference)
	mk := masterkeyFile{
		Version:          999,
		ScryptSalt:       base64.StdEncoding.EncodeToString(scryptSalt),
		ScryptCostParam:  32768,
		ScryptBlockSize:  8,
		PrimaryMasterKey: base64.StdEncoding.EncodeToString(wrappedEnc),
		HmacMasterKey:    base64.StdEncoding.EncodeToString(wrappedMac),
		VersionMac:       base64.StdEncoding.EncodeToString(vm),
	}
	mkBytes, err := json.MarshalIndent(mk, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("vault: serializing masterkey: %w", err)
	}
	if err := os.MkdirAll(vaultDir, 0o755); err != nil {
		return nil, fmt.Errorf("vault: creating directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(vaultDir, "masterkey.cryptomator"), mkBytes, 0o600); err != nil {
		return nil, fmt.Errorf("vault: writing masterkey.cryptomator: %w", err)
	}

	// 6. Write vault.cryptomator (compact JWT, HS256, key = encKey||macKey)
	jwtToken, err := buildVaultJWT(encKey, macKey)
	if err != nil {
		return nil, fmt.Errorf("vault: building vault.cryptomator JWT: %w", err)
	}
	if err := os.WriteFile(filepath.Join(vaultDir, "vault.cryptomator"), []byte(jwtToken), 0o644); err != nil {
		return nil, fmt.Errorf("vault: writing vault.cryptomator: %w", err)
	}

	return &Vault{encKey: encKey, macKey: macKey}, nil
}

// vaultVersionMac computes HMAC-SHA256(macKey, BE32(version)).
func vaultVersionMac(macKey []byte, version uint32) []byte {
	h := hmac.New(sha256.New, macKey)
	var vb [4]byte
	binary.BigEndian.PutUint32(vb[:], version)
	h.Write(vb[:])
	return h.Sum(nil)
}

// buildVaultJWT builds a compact JWT for vault.cryptomator.
// Header: {"kid":"masterkeyfile:masterkey.cryptomator","alg":"HS256","typ":"JWT"}
// Payload: {"jti":"<uuid>","format":8,"cipherCombo":"SIV_GCM","shorteningThreshold":220}
// Signature: HS256, key = encKey||macKey (64B).
func buildVaultJWT(encKey, macKey []byte) (string, error) {
	headerJSON := `{"kid":"masterkeyfile:masterkey.cryptomator","alg":"HS256","typ":"JWT"}`
	payloadJSON := fmt.Sprintf(`{"jti":%q,"format":8,"cipherCombo":"SIV_GCM","shorteningThreshold":220}`,
		uuid.NewString())

	headerPart := base64.RawURLEncoding.EncodeToString([]byte(headerJSON))
	payloadPart := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))

	sigInput := headerPart + "." + payloadPart
	sigKey := append(encKey, macKey...)
	h := hmac.New(sha256.New, sigKey)
	h.Write([]byte(sigInput))
	sig := h.Sum(nil)

	return sigInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// verifyVaultJWT verifies the JWT signature and checks that format=8 and cipherCombo=SIV_GCM.
func verifyVaultJWT(token string, encKey, macKey []byte) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errors.New("malformed JWT: expected 3 parts")
	}
	// Signature key = encKey ‖ macKey (64B)
	sigKey := append(encKey, macKey...)

	// Verify HS256 signature: HMAC-SHA256(header.payload)
	h := hmac.New(sha256.New, sigKey)
	h.Write([]byte(parts[0] + "." + parts[1]))
	expectedSig := h.Sum(nil)

	gotSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("decoding JWT signature: %w", err)
	}
	if !hmac.Equal(gotSig, expectedSig) {
		return errors.New("JWT signature mismatch — masterkey does not match vault")
	}

	// Parse payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("decoding JWT payload: %w", err)
	}
	var payload struct {
		Format      int    `json:"format"`
		CipherCombo string `json:"cipherCombo"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("parsing JWT payload: %w", err)
	}
	if payload.Format != 8 {
		return fmt.Errorf("unsupported vault format: %d (expected 8)", payload.Format)
	}
	if payload.CipherCombo != "SIV_GCM" {
		return fmt.Errorf("unsupported cipherCombo: %q (expected SIV_GCM)", payload.CipherCombo)
	}
	return nil
}
