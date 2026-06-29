// Package vault — keywrap.go: AES Key Wrap (RFC 3394).
package vault

import (
	"crypto/aes"
	"encoding/binary"
	"errors"
)

// aesKWDefaultIV is the standard IV for AES Key Wrap (RFC 3394).
var aesKWDefaultIV = [8]byte{0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6, 0xA6}

// aesKWWrap wraps plaintext (multiple of 8 bytes) with key kek.
// 32B plaintext → 40B wrapped (RFC 3394).
func aesKWWrap(kek, plaintext []byte) ([]byte, error) {
	if len(plaintext)%8 != 0 {
		return nil, errors.New("keywrap: plaintext length must be a multiple of 8")
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	n := len(plaintext) / 8
	// R is the working buffer: R[0] = A (8B IV), R[1..n] = plaintext blocks
	R := make([][]byte, n+1)
	R[0] = make([]byte, 8)
	copy(R[0], aesKWDefaultIV[:])
	for i := 1; i <= n; i++ {
		R[i] = make([]byte, 8)
		copy(R[i], plaintext[(i-1)*8:i*8])
	}

	buf := make([]byte, 16)
	for j := 0; j < 6; j++ {
		for i := 1; i <= n; i++ {
			copy(buf[:8], R[0])
			copy(buf[8:], R[i])
			block.Encrypt(buf, buf)
			// A = MSB(64, B) XOR t (t = n*j + i)
			t := uint64(n*j + i)
			a := binary.BigEndian.Uint64(buf[:8])
			a ^= t
			binary.BigEndian.PutUint64(R[0], a)
			copy(R[i], buf[8:])
		}
	}

	out := make([]byte, 8*(n+1))
	copy(out, R[0])
	for i := 1; i <= n; i++ {
		copy(out[i*8:], R[i])
	}
	return out, nil
}

// aesKWUnwrap unwraps wrapped (40B) with key kek → plaintext (32B).
// Returns an error if the IV does not match (wrong password or corrupt data).
func aesKWUnwrap(kek, wrapped []byte) ([]byte, error) {
	if len(wrapped)%8 != 0 || len(wrapped) < 16 {
		return nil, errors.New("keywrap: invalid wrapped length")
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	n := len(wrapped)/8 - 1
	R := make([][]byte, n+1)
	R[0] = make([]byte, 8)
	copy(R[0], wrapped[:8])
	for i := 1; i <= n; i++ {
		R[i] = make([]byte, 8)
		copy(R[i], wrapped[i*8:(i+1)*8])
	}

	buf := make([]byte, 16)
	for j := 5; j >= 0; j-- {
		for i := n; i >= 1; i-- {
			t := uint64(n*j + i)
			a := binary.BigEndian.Uint64(R[0])
			a ^= t
			binary.BigEndian.PutUint64(buf[:8], a)
			copy(buf[8:], R[i])
			block.Decrypt(buf, buf)
			copy(R[0], buf[:8])
			copy(R[i], buf[8:])
		}
	}

	if R[0][0] != 0xA6 || R[0][1] != 0xA6 || R[0][2] != 0xA6 || R[0][3] != 0xA6 ||
		R[0][4] != 0xA6 || R[0][5] != 0xA6 || R[0][6] != 0xA6 || R[0][7] != 0xA6 {
		return nil, errors.New("keywrap: IV mismatch — wrong key or corrupt data")
	}

	out := make([]byte, n*8)
	for i := 1; i <= n; i++ {
		copy(out[(i-1)*8:], R[i])
	}
	return out, nil
}
