package encode

import (
	"fmt"

	"github.com/tilinna/z85"
)

// EncodeZ85 encodes arbitrary binary data to a Z85 string.
// Handles padding: prepends a 1-byte pad-length indicator, then pads the
// combined data to a multiple of 4 bytes before Z85 encoding.
func EncodeZ85(data []byte) (string, error) {
	// Calculate padding needed: (data + 1 byte for pad count) must be divisible by 4
	padNeeded := (4 - ((len(data) + 1) % 4)) % 4

	// Build padded input: [padCount] [data...] [zeros...]
	padded := make([]byte, 1+len(data)+padNeeded)
	padded[0] = byte(padNeeded)
	copy(padded[1:], data)
	// Trailing bytes are already zero

	dst := make([]byte, z85.EncodedLen(len(padded)))
	if _, err := z85.Encode(dst, padded); err != nil {
		return "", fmt.Errorf("z85 encoding: %w", err)
	}
	return string(dst), nil
}

// DecodeZ85 decodes a Z85 string back to the original binary data,
// reversing the padding applied by EncodeZ85.
func DecodeZ85(encoded string) ([]byte, error) {
	src := []byte(encoded)
	dst := make([]byte, z85.DecodedLen(len(src)))
	if _, err := z85.Decode(dst, src); err != nil {
		return nil, fmt.Errorf("z85 decoding: %w", err)
	}

	if len(dst) == 0 {
		return nil, fmt.Errorf("z85 decoded data is empty")
	}

	padCount := int(dst[0])
	if padCount > 3 {
		return nil, fmt.Errorf("invalid pad count: %d", padCount)
	}

	data := dst[1 : len(dst)-padCount]
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
