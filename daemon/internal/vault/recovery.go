// Package vault — recovery.go: Cryptomator-compatible vault recovery key.
// Format: WordEncoder (4096 words, 3 bytes → 2 words) over a 66-byte payload:
//
//	rawKey(64) = encKey(32) ‖ macKey(32) + 2-byte CRC32-IEEE (LE low 16 bits).
//
// Compatible with Cryptomator (verified against a real reference vault).
package vault

import (
	_ "embed"
	"errors"
	"fmt"
	"hash/crc32"
	"strings"
)

//go:embed i18n/4096words_en.txt
var wordsRaw string

var (
	wordList  []string
	wordIndex map[string]int
)

func init() {
	lines := strings.Split(strings.TrimRight(wordsRaw, "\n"), "\n")
	wordList = make([]string, 0, 4096)
	for _, l := range lines {
		w := strings.TrimSpace(l)
		if w != "" {
			wordList = append(wordList, w)
		}
	}
	if len(wordList) != 4096 {
		panic(fmt.Sprintf("vault recovery: word list must contain 4096 words, got %d", len(wordList)))
	}
	wordIndex = make(map[string]int, 4096)
	for i, w := range wordList {
		wordIndex[w] = i
	}
}

// wordEncode encodes data (length must be a multiple of 3) into a space-separated word string.
// 3 bytes → 2 words of 12 bits each:
//
//	w1 = (b1 << 4) | (b2 >> 4)   (12 bits)
//	w2 = ((b2 & 0x0F) << 8) | b3  (12 bits)
func wordEncode(data []byte) string {
	if len(data)%3 != 0 {
		panic("wordEncode: data length must be a multiple of 3")
	}
	out := make([]string, 0, len(data)/3*2)
	for i := 0; i < len(data); i += 3 {
		b1, b2, b3 := int(data[i]), int(data[i+1]), int(data[i+2])
		w1 := (b1 << 4) | (b2 >> 4)
		w2 := ((b2 & 0x0F) << 8) | b3
		out = append(out, wordList[w1], wordList[w2])
	}
	return strings.Join(out, " ")
}

// wordDecode decodes a space-separated word string back to bytes.
// Word count must be even; each pair of words → 3 bytes.
func wordDecode(phrase string) ([]byte, error) {
	phraseWords := strings.Fields(phrase)
	if len(phraseWords) == 0 {
		return nil, errors.New("vault recovery: empty phrase")
	}
	if len(phraseWords)%2 != 0 {
		return nil, fmt.Errorf("vault recovery: odd word count: %d", len(phraseWords))
	}
	out := make([]byte, len(phraseWords)/2*3)
	for i := 0; i < len(phraseWords); i += 2 {
		w1, ok1 := wordIndex[phraseWords[i]]
		w2, ok2 := wordIndex[phraseWords[i+1]]
		if !ok1 {
			return nil, fmt.Errorf("vault recovery: unknown word %q", phraseWords[i])
		}
		if !ok2 {
			return nil, fmt.Errorf("vault recovery: unknown word %q", phraseWords[i+1])
		}
		j := i / 2 * 3
		out[j] = byte(0xFF & (w1 >> 4))
		out[j+1] = byte((0xF0 & (w1 << 4)) | (0x0F & (w2 >> 8)))
		out[j+2] = byte(0xFF & w2)
	}
	return out, nil
}

// RecoveryKey computes the Cryptomator-compatible recovery key for vault v.
// Returns a 44-word phrase (space-separated).
func (v *Vault) RecoveryKey() string {
	rawKey := make([]byte, 64)
	copy(rawKey[:32], v.encKey)
	copy(rawKey[32:], v.macKey)

	crc := crc32.ChecksumIEEE(rawKey)
	// 2 bytes = low 16 bits of CRC32 in little-endian
	payload := append(rawKey, byte(crc), byte(crc>>8))
	return wordEncode(payload)
}

// RecoveryToKeys restores encKey and macKey from a Cryptomator recovery phrase.
// Verifies the CRC; returns a descriptive error on failure.
func RecoveryToKeys(phrase string) (encKey, macKey []byte, err error) {
	payload, err := wordDecode(phrase)
	if err != nil {
		return nil, nil, err
	}
	if len(payload) != 66 {
		return nil, nil, fmt.Errorf("vault recovery: expected 66-byte payload, got %d", len(payload))
	}

	rawKey := payload[:64]
	gotCRC := payload[64:]

	crc := crc32.ChecksumIEEE(rawKey)
	wantCRC := []byte{byte(crc), byte(crc >> 8)}
	if gotCRC[0] != wantCRC[0] || gotCRC[1] != wantCRC[1] {
		return nil, nil, fmt.Errorf("vault recovery: checksum mismatch (phrase corrupted or incorrect)")
	}

	return rawKey[:32], rawKey[32:64], nil
}
