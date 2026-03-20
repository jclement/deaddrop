package encode

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"

	"github.com/tilinna/z85"
)

// EncodeZ85 encodes arbitrary binary data to a Z85 string.
// Format: [padCount(1)] [CRC32(4)] [data...] [zeros(0-3)]
// The CRC32 checksum (IEEE) covers the original data, enabling detection
// of transcription errors when manually typing the Z85 block.
func EncodeZ85(data []byte) (string, error) {
	checksum := crc32.ChecksumIEEE(data)
	var crcBytes [4]byte
	binary.BigEndian.PutUint32(crcBytes[:], checksum)

	// Calculate padding: (1 + 4 + len(data)) must be divisible by 4
	headerLen := 1 + 4 // padCount + CRC32
	padNeeded := (4 - ((headerLen + len(data)) % 4)) % 4

	// Build padded input: [padCount] [CRC32] [data...] [zeros...]
	padded := make([]byte, headerLen+len(data)+padNeeded)
	padded[0] = byte(padNeeded)
	copy(padded[1:5], crcBytes[:])
	copy(padded[5:], data)
	// Trailing bytes are already zero

	dst := make([]byte, z85.EncodedLen(len(padded)))
	if _, err := z85.Encode(dst, padded); err != nil {
		return "", fmt.Errorf("z85 encoding: %w", err)
	}
	return string(dst), nil
}

// DecodeZ85 decodes a Z85 string back to the original binary data,
// reversing the padding applied by EncodeZ85. Verifies the CRC32
// checksum and returns a clear error on mismatch (transcription error).
func DecodeZ85(encoded string) ([]byte, error) {
	src := []byte(encoded)
	dst := make([]byte, z85.DecodedLen(len(src)))
	if _, err := z85.Decode(dst, src); err != nil {
		return nil, fmt.Errorf("z85 decoding: %w", err)
	}

	if len(dst) < 5 {
		return nil, fmt.Errorf("z85 decoded data too short (need at least 5 bytes, got %d)", len(dst))
	}

	padCount := int(dst[0])
	if padCount > 3 {
		return nil, fmt.Errorf("invalid pad count: %d", padCount)
	}

	storedCRC := binary.BigEndian.Uint32(dst[1:5])
	data := dst[5 : len(dst)-padCount]

	// Verify checksum
	actualCRC := crc32.ChecksumIEEE(data)
	if storedCRC != actualCRC {
		return nil, fmt.Errorf("checksum mismatch: the Z85 text may have been mistyped (expected %08x, got %08x)", storedCRC, actualCRC)
	}

	// Return a copy to avoid holding the larger buffer
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// FormatZ85Block formats a Z85 string into fixed-width lines for PDF display.
func FormatZ85Block(encoded string, lineWidth int) []string {
	if lineWidth <= 0 {
		lineWidth = 50
	}
	var lines []string
	for len(encoded) > 0 {
		end := lineWidth
		if end > len(encoded) {
			end = len(encoded)
		}
		lines = append(lines, encoded[:end])
		encoded = encoded[end:]
	}
	return lines
}
