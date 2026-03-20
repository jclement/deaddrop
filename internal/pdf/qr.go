package pdf

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	goqrcode "github.com/skip2/go-qrcode"
)

// MaxQRBytes is the maximum raw payload size for a single QR code.
// We base64-encode data before QR encoding for reliable round-trips.
// 1000 binary bytes -> ~1336 base64 chars, fitting in QR version ~22.
// This keeps QR density low enough for reliable scanning from photos/screens.
const MaxQRBytes = 1000

// QRSize is the pixel dimension of generated QR code images.
// Large size ensures high-density QR codes remain scannable.
const QRSize = 1024

// EncodeQR generates a QR code PNG image from binary data.
// The data is base64-encoded before QR encoding to ensure reliable
// round-trips through QR encode/decode (binary mode is lossy in some decoders).
func EncodeQR(data []byte) ([]byte, error) {
	encoded := base64.StdEncoding.EncodeToString(data)
	if len(encoded) > 2953 {
		return nil, fmt.Errorf("data too large for single QR code: %d encoded bytes (max 2953)", len(encoded))
	}
	png, err := goqrcode.Encode(encoded, goqrcode.Low, QRSize)
	if err != nil {
		return nil, fmt.Errorf("encoding QR code: %w", err)
	}
	return png, nil
}

// DecodeQR reads an image file and decodes the QR code found in it.
// Returns the raw binary data (base64-decoded from QR content).
func DecodeQR(imagePath string) ([]byte, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("opening image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	return DecodeQRFromImage(img)
}

// DecodeQRFromImage decodes a QR code from an in-memory image.
func DecodeQRFromImage(img image.Image) ([]byte, error) {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil, fmt.Errorf("creating bitmap: %w", err)
	}

	reader := qrcode.NewQRCodeReader()
	result, err := reader.DecodeWithoutHints(bmp)
	if err != nil {
		return nil, fmt.Errorf("scanning QR code: %w", err)
	}

	text := result.GetText()
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(text))
	if err != nil {
		return nil, fmt.Errorf("base64 decoding QR content: %w", err)
	}
	return data, nil
}

// SplitPayload splits a payload into chunks that each fit in a single QR code.
// For a single chunk, returns the payload as-is.
// For multiple chunks, each chunk is prefixed with "DS" + index byte + total byte.
func SplitPayload(payload []byte) [][]byte {
	if len(payload) <= MaxQRBytes {
		return [][]byte{payload}
	}

	// Each chunk has a 4-byte header: "DS" + index + total
	chunkDataSize := MaxQRBytes - 4
	totalChunks := (len(payload) + chunkDataSize - 1) / chunkDataSize

	chunks := make([][]byte, totalChunks)
	for i := range totalChunks {
		start := i * chunkDataSize
		end := start + chunkDataSize
		if end > len(payload) {
			end = len(payload)
		}

		chunk := make([]byte, 0, 4+end-start)
		chunk = append(chunk, 'D', 'S', byte(i), byte(totalChunks))
		chunk = append(chunk, payload[start:end]...)
		chunks[i] = chunk
	}
	return chunks
}

// ReassemblePayload reassembles split chunks back into the original payload.
// If the data starts with "DD01" it's a complete single payload.
// If it starts with "DS" it's a split chunk.
func ReassemblePayload(chunks [][]byte) ([]byte, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks to reassemble")
	}

	// Single complete payload
	if len(chunks) == 1 && bytes.HasPrefix(chunks[0], []byte("DD01")) {
		return chunks[0], nil
	}

	// Validate and sort split chunks
	total := int(chunks[0][3])
	if len(chunks) != total {
		return nil, fmt.Errorf("expected %d chunks, got %d", total, len(chunks))
	}

	ordered := make([][]byte, total)
	for _, chunk := range chunks {
		if !bytes.HasPrefix(chunk, []byte("DS")) {
			return nil, fmt.Errorf("invalid chunk header")
		}
		idx := int(chunk[2])
		if idx >= total {
			return nil, fmt.Errorf("chunk index %d out of range (total %d)", idx, total)
		}
		ordered[idx] = chunk[4:] // strip the DS header
	}

	var payload bytes.Buffer
	for i, data := range ordered {
		if data == nil {
			return nil, fmt.Errorf("missing chunk %d", i)
		}
		payload.Write(data)
	}
	return payload.Bytes(), nil
}
