package vault

import (
	"bytes"
	"strings"
	"testing"
)

// canonicalPhrase is the real recovery phrase from Cryptomator for testdata/cmvault (password: password123).
const canonicalPhrase = "kit sample strongly teens mode level autonomy precise closely hurry violence buddy delegate fishing poor burn now wash centre talented lifestyle standard ashamed scrutiny grid packet luck elegant factor fierce charter say fleet blind generic outfit estimate forever aged adjacent bent harvest once away"

// TestRecoveryKeyMatchesCanonical — oracle test:
// RecoveryKey() must return exactly the same phrase that Cryptomator produces for this vault.
func TestRecoveryKeyMatchesCanonical(t *testing.T) {
	v, err := Open("testdata/cmvault", "password123")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got := v.RecoveryKey()
	if got != canonicalPhrase {
		t.Fatalf("RecoveryKey() did not match the Cryptomator reference\nGOT:  %s\nWANT: %s", got, canonicalPhrase)
	}
	t.Logf("RecoveryKey() == canonical ✓ (%d words)", len(strings.Fields(got)))
}

// TestRecoveryToKeysFromCanonical: decode the canonical phrase → encKey‖macKey must
// match the keys from the actual opened vault.
func TestRecoveryToKeysFromCanonical(t *testing.T) {
	v, err := Open("testdata/cmvault", "password123")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	enc, mac, err := RecoveryToKeys(canonicalPhrase)
	if err != nil {
		t.Fatalf("RecoveryToKeys: %v", err)
	}

	if !bytes.Equal(enc, v.encKey) {
		t.Fatalf("encKey did not match\ngot:  %x\nwant: %x", enc, v.encKey)
	}
	if !bytes.Equal(mac, v.macKey) {
		t.Fatalf("macKey did not match\ngot:  %x\nwant: %x", mac, v.macKey)
	}
	t.Log("RecoveryToKeys → encKey and macKey matched the vault keys ✓")
}

// TestRecoveryRoundTrip: round-trip wordEncode/wordDecode.
func TestRecoveryRoundTrip(t *testing.T) {
	// 66-byte payload (arbitrary, multiple of 3)
	payload := make([]byte, 66)
	for i := range payload {
		payload[i] = byte(i * 3)
	}
	encoded := wordEncode(payload)
	decoded, err := wordDecode(encoded)
	if err != nil {
		t.Fatalf("wordDecode: %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatalf("round-trip failed\ngot:  %x\nwant: %x", decoded, payload)
	}
	t.Logf("round-trip OK, %d words", len(strings.Fields(encoded)))
}

// TestRecoveryToKeysBadCRC: a corrupted phrase (one word changed) → CRC error.
func TestRecoveryToKeysBadCRC(t *testing.T) {
	// Change the last word to something else
	words := strings.Fields(canonicalPhrase)
	words[43] = "ad" // replace the last word
	badPhrase := strings.Join(words, " ")

	_, _, err := RecoveryToKeys(badPhrase)
	if err == nil {
		t.Fatal("expected an error for a corrupted phrase, but err == nil")
	}
	t.Logf("expected error: %v", err)
}

// TestRecoveryToKeysBadWord: unknown word → error.
func TestRecoveryToKeysBadWord(t *testing.T) {
	_, _, err := RecoveryToKeys("notaword " + strings.Repeat("ad ", 43))
	if err == nil {
		t.Fatal("expected an error for an unknown word")
	}
	t.Logf("expected error: %v", err)
}
