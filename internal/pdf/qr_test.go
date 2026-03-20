package pdf_test

import (
	"bytes"
	"crypto/rand"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/jclement/deaddrop/internal/pdf"
)

func TestEncodeDecodeQRRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"short ASCII", []byte("hello world")},
		{"DD01 header", append([]byte("DD01"), []byte("test payload")...)},
		{"binary data", []byte{0x00, 0x01, 0x80, 0xFF, 0xFE, 0x7F}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qrPNG, err := pdf.EncodeQR(tt.data)
			if err != nil {
				t.Fatalf("EncodeQR: %v", err)
			}

			img, err := png.Decode(bytes.NewReader(qrPNG))
			if err != nil {
				t.Fatalf("png.Decode: %v", err)
			}

			decoded, err := pdf.DecodeQRFromImage(img)
			if err != nil {
				t.Fatalf("DecodeQRFromImage: %v", err)
			}

			if !bytes.Equal(decoded, tt.data) {
				t.Errorf("round-trip failed:\n  got  %x\n  want %x", decoded, tt.data)
			}
		})
	}
}

func TestEncodeDecodeQRRandomBinary(t *testing.T) {
	// Test with random binary data of various sizes
	for _, size := range []int{10, 50, 100, 500, 1000} {
		data := make([]byte, size)
		rand.Read(data)

		qrPNG, err := pdf.EncodeQR(data)
		if err != nil {
			t.Fatalf("EncodeQR (size=%d): %v", size, err)
		}

		img, err := png.Decode(bytes.NewReader(qrPNG))
		if err != nil {
			t.Fatalf("png.Decode: %v", err)
		}

		decoded, err := pdf.DecodeQRFromImage(img)
		if err != nil {
			t.Fatalf("DecodeQRFromImage (size=%d): %v", size, err)
		}

		if !bytes.Equal(decoded, data) {
			t.Errorf("round-trip failed for %d random bytes", size)
		}
	}
}

func TestEncodeQRTooLarge(t *testing.T) {
	// Need data that exceeds 2953 base64 bytes after encoding.
	// 2216 raw bytes -> ceil(2216/3)*4 = 2956 base64 bytes > 2953 limit
	data := make([]byte, 2216)
	_, err := pdf.EncodeQR(data)
	if err == nil {
		t.Fatal("expected error for oversized data")
	}
}

func TestDecodeQRFromFile(t *testing.T) {
	data := []byte("test data for file-based QR decode")
	qrPNG, err := pdf.EncodeQR(data)
	if err != nil {
		t.Fatalf("EncodeQR: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test.png")
	if err := os.WriteFile(tmpFile, qrPNG, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	decoded, err := pdf.DecodeQR(tmpFile)
	if err != nil {
		t.Fatalf("DecodeQR: %v", err)
	}

	if !bytes.Equal(decoded, data) {
		t.Errorf("file round-trip failed")
	}
}

func TestSplitPayload(t *testing.T) {
	// Small payload - single chunk, no header
	small := make([]byte, 100)
	chunks := pdf.SplitPayload(small)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if !bytes.Equal(chunks[0], small) {
		t.Error("single chunk should equal original payload")
	}

	// Large payload - multiple chunks with DS headers
	large := make([]byte, pdf.MaxQRBytes*3)
	rand.Read(large)
	chunks = pdf.SplitPayload(large)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	// Each chunk should start with "DS"
	for i, chunk := range chunks {
		if chunk[0] != 'D' || chunk[1] != 'S' {
			t.Errorf("chunk %d missing DS header", i)
		}
		if int(chunk[2]) != i {
			t.Errorf("chunk %d has wrong index: %d", i, chunk[2])
		}
		if int(chunk[3]) != len(chunks) {
			t.Errorf("chunk %d has wrong total: %d", i, chunk[3])
		}
	}

	// Reassemble and verify
	reassembled, err := pdf.ReassemblePayload(chunks)
	if err != nil {
		t.Fatalf("ReassemblePayload: %v", err)
	}
	if !bytes.Equal(reassembled, large) {
		t.Error("reassembled payload does not match original")
	}
}
