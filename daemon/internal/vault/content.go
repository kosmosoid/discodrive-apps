// Package vault — content.go: file content encryption/decryption (AES-GCM, Cryptomator format 8).
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// chunkPlainSize is the plaintext chunk size (32 KiB).
	chunkPlainSize = 32 * 1024
	// headerNonceSize is the header nonce size (AES-GCM 12B).
	headerNonceSize = 12
	// headerCTSize is the header ciphertext size: 8 (0xFF marker) + 32 (contentKey) + 16 (GCM tag).
	headerCTSize = 56
	// headerTotalSize is the total file header size.
	headerTotalSize = headerNonceSize + headerCTSize
)

// EncryptContent encrypts src and writes Cryptomator SIV_GCM format to dst.
// Header 68B: headerNonce(12) ‖ AES-GCM(encKey).Seal(nil, headerNonce, [0xFF*8 ‖ contentKey(32)], nil).
// Chunk i: chunkNonce(12) ‖ AES-GCM(contentKey).Seal(nil, chunkNonce, plaintext≤32768, aad=BE64(i)‖headerNonce).
func (v *Vault) EncryptContent(dst io.Writer, src io.Reader) error {
	// Generate headerNonce and contentKey
	headerNonce := make([]byte, headerNonceSize)
	if _, err := rand.Read(headerNonce); err != nil {
		return fmt.Errorf("vault: generating headerNonce: %w", err)
	}
	contentKey := make([]byte, 32)
	if _, err := rand.Read(contentKey); err != nil {
		return fmt.Errorf("vault: generating contentKey: %w", err)
	}

	// Encrypt header: payload = [0xFF * 8] ‖ contentKey
	hBlock, err := aes.NewCipher(v.encKey)
	if err != nil {
		return fmt.Errorf("vault: AES for header: %w", err)
	}
	hGCM, err := cipher.NewGCM(hBlock)
	if err != nil {
		return fmt.Errorf("vault: GCM for header: %w", err)
	}
	headerPayload := make([]byte, 40) // 8 × 0xFF + contentKey(32)
	for i := 0; i < 8; i++ {
		headerPayload[i] = 0xFF
	}
	copy(headerPayload[8:], contentKey)
	headerCT := hGCM.Seal(nil, headerNonce, headerPayload, nil)

	// Write the header
	if _, err := dst.Write(headerNonce); err != nil {
		return fmt.Errorf("vault: writing headerNonce: %w", err)
	}
	if _, err := dst.Write(headerCT); err != nil {
		return fmt.Errorf("vault: writing headerCT: %w", err)
	}

	// Create GCM for chunks
	cBlock, err := aes.NewCipher(contentKey)
	if err != nil {
		return fmt.Errorf("vault: AES for chunks: %w", err)
	}
	cGCM, err := cipher.NewGCM(cBlock)
	if err != nil {
		return fmt.Errorf("vault: GCM for chunks: %w", err)
	}

	// Read and encrypt chunks
	buf := make([]byte, chunkPlainSize)
	var chunkIdx uint64
	for {
		n, err := io.ReadFull(src, buf)
		if err == io.EOF && n == 0 {
			break // source is empty or fully read
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			return fmt.Errorf("vault: reading chunk %d: %w", chunkIdx, err)
		}
		chunk := buf[:n]

		chunkNonce := make([]byte, 12)
		if _, err := rand.Read(chunkNonce); err != nil {
			return fmt.Errorf("vault: generating chunkNonce: %w", err)
		}

		// AAD = BE64(chunkIdx) ‖ headerNonce (8+12=20B)
		aad := make([]byte, 20)
		binary.BigEndian.PutUint64(aad[:8], chunkIdx)
		copy(aad[8:], headerNonce)

		ct := cGCM.Seal(nil, chunkNonce, chunk, aad)

		if _, err := dst.Write(chunkNonce); err != nil {
			return fmt.Errorf("vault: writing chunkNonce[%d]: %w", chunkIdx, err)
		}
		if _, err := dst.Write(ct); err != nil {
			return fmt.Errorf("vault: writing chunk CT[%d]: %w", chunkIdx, err)
		}

		chunkIdx++
		if err == io.ErrUnexpectedEOF {
			break // last (partial) chunk
		}
	}
	return nil
}

// DecryptContent decrypts a Cryptomator SIV_GCM format file.
// Format: header 68B (nonce12 + ct40+tag16) + chunks (nonce12 + ct≤32768 + tag16).
func (v *Vault) DecryptContent(dst io.Writer, src io.Reader) error {
	// 1. Read the header (68 bytes)
	header := make([]byte, headerTotalSize)
	if _, err := io.ReadFull(src, header); err != nil {
		return fmt.Errorf("vault: reading file header: %w", err)
	}
	headerNonce := header[:headerNonceSize]
	headerCT := header[headerNonceSize:]

	// Decrypt the header with encKey
	hBlock, err := aes.NewCipher(v.encKey)
	if err != nil {
		return fmt.Errorf("vault: creating AES for header: %w", err)
	}
	hGCM, err := cipher.NewGCM(hBlock)
	if err != nil {
		return fmt.Errorf("vault: creating GCM for header: %w", err)
	}
	hPayload, err := hGCM.Open(nil, headerNonce, headerCT, nil)
	if err != nil {
		return fmt.Errorf("vault: decrypting header: %w", err)
	}
	// Payload: 8 bytes of 0xFF + contentKey (32B)
	if len(hPayload) != 40 {
		return fmt.Errorf("vault: unexpected header payload size: %d", len(hPayload))
	}

	contentKey := hPayload[8:40]

	// Create GCM for chunks (contentKey)
	cBlock, err := aes.NewCipher(contentKey)
	if err != nil {
		return fmt.Errorf("vault: creating AES for chunks: %w", err)
	}
	cGCM, err := cipher.NewGCM(cBlock)
	if err != nil {
		return fmt.Errorf("vault: creating GCM for chunks: %w", err)
	}

	// 2. Read chunks
	// Full chunk frame: nonce(12) + ct(32768) + tag(16) = 32796 bytes
	// Last frame: nonce(12) + ct(<32768) + tag(16) — shorter
	chunkNonce := 12
	chunkOverhead := cGCM.Overhead() // 16
	fullFrameSize := chunkNonce + chunkPlainSize + chunkOverhead

	buf := make([]byte, fullFrameSize)
	var chunkIdx uint64

	for {
		n, err := io.ReadFull(src, buf)
		if err == io.EOF && n == 0 {
			// No data — end of file (header-only file)
			break
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			return fmt.Errorf("vault: reading chunk %d: %w", chunkIdx, err)
		}
		isLast := err == io.ErrUnexpectedEOF

		frame := buf[:n]
		if len(frame) < chunkNonce+chunkOverhead {
			return fmt.Errorf("vault: chunk %d too short: %d bytes", chunkIdx, len(frame))
		}

		nonce := frame[:chunkNonce]
		ct := frame[chunkNonce:]

		// Chunk AAD: BE64(chunkIdx) ‖ headerNonce (8+12=20B)
		aad := make([]byte, 20)
		binary.BigEndian.PutUint64(aad[:8], chunkIdx)
		copy(aad[8:], headerNonce)

		pt, err := cGCM.Open(nil, nonce, ct, aad)
		if err != nil {
			return fmt.Errorf("vault: decrypting chunk %d: %w", chunkIdx, err)
		}
		if _, err := dst.Write(pt); err != nil {
			return fmt.Errorf("vault: writing chunk %d: %w", chunkIdx, err)
		}

		chunkIdx++
		if isLast {
			break
		}
	}

	return nil
}
