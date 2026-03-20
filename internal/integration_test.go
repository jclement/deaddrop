package internal_test

import (
	"bytes"
	crand "crypto/rand"
	"image/png"
	"testing"
	"time"

	"github.com/jclement/deaddrop/internal/crypto"
	"github.com/jclement/deaddrop/internal/encode"
	"github.com/jclement/deaddrop/internal/pdf"
)

// TestFullRoundTripViaQR tests the complete pipeline:
// secret -> encrypt -> QR PNG -> decode QR -> decrypt -> verify original
func TestFullRoundTripViaQR(t *testing.T) {
	secrets := []struct {
		name   string
		secret []byte
	}{
		{"short text", []byte("correct horse battery staple")},
		{"seed phrase", []byte("abandon ability able about above absent absorb abstract absurd abuse access accident")},
		{"binary key", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0x80, 0x7F}},
		{"multiline", []byte("line1\nline2\nline3\n")},
	}

	for _, tt := range secrets {
		t.Run(tt.name, func(t *testing.T) {
			// Generate passphrase
			passphrase, err := crypto.GeneratePassphrase(6)
			if err != nil {
				t.Fatalf("GeneratePassphrase: %v", err)
			}

			// Encrypt (work factor 1 for speed)
			payload, err := crypto.Encrypt(tt.secret, passphrase, 1)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}

			// Verify DD01 header
			if !bytes.HasPrefix(payload, crypto.MagicHeader) {
				t.Fatal("payload missing DD01 header")
			}

			// Generate QR code
			qrPNG, err := pdf.EncodeQR(payload)
			if err != nil {
				t.Fatalf("EncodeQR: %v", err)
			}

			// Decode QR from the PNG
			img, err := png.Decode(bytes.NewReader(qrPNG))
			if err != nil {
				t.Fatalf("png.Decode: %v", err)
			}
			decoded, err := pdf.DecodeQRFromImage(img)
			if err != nil {
				t.Fatalf("DecodeQRFromImage: %v", err)
			}

			// Decrypt
			result, err := crypto.Decrypt(decoded, passphrase)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}

			// Verify
			if !bytes.Equal(result, tt.secret) {
				t.Errorf("round-trip failed:\n  got  %q\n  want %q", result, tt.secret)
			}
		})
	}
}

// TestFullRoundTripViaZ85 tests the complete pipeline via Z85 fallback:
// secret -> encrypt -> Z85 encode -> Z85 decode -> decrypt -> verify original
func TestFullRoundTripViaZ85(t *testing.T) {
	secret := []byte("this is my super secret recovery key for everything important")

	passphrase, err := crypto.GeneratePassphrase(6)
	if err != nil {
		t.Fatalf("GeneratePassphrase: %v", err)
	}

	payload, err := crypto.Encrypt(secret, passphrase, 1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Z85 encode
	z85Text, err := encode.EncodeZ85(payload)
	if err != nil {
		t.Fatalf("EncodeZ85: %v", err)
	}

	// Z85 decode
	decoded, err := encode.DecodeZ85(z85Text)
	if err != nil {
		t.Fatalf("DecodeZ85: %v", err)
	}

	// Decrypt
	result, err := crypto.Decrypt(decoded, passphrase)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(result, secret) {
		t.Errorf("Z85 round-trip failed:\n  got  %q\n  want %q", result, secret)
	}
}

// TestFullRoundTripPDFGeneration tests the complete pipeline including PDF generation.
// Generates a full PDF and verifies it's a valid PDF document.
func TestFullRoundTripPDFGeneration(t *testing.T) {
	secret := []byte("test secret for PDF generation round-trip")

	passphrase, err := crypto.GeneratePassphrase(6)
	if err != nil {
		t.Fatalf("GeneratePassphrase: %v", err)
	}

	payload, err := crypto.Encrypt(secret, passphrase, 1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	z85Text, err := encode.EncodeZ85(payload)
	if err != nil {
		t.Fatalf("EncodeZ85: %v", err)
	}

	qrPNG, err := pdf.EncodeQR(payload)
	if err != nil {
		t.Fatalf("EncodeQR: %v", err)
	}

	// Generate PDF
	pages := []pdf.PageData{
		{
			QRCodePNG:  qrPNG,
			Z85Text:    z85Text,
			PageNumber: 1,
			TotalPages: 1,
		},
	}
	opts := pdf.PDFOptions{
		Label:            "e2e-test",
		Title:            "E2E Test Document",
		Date:             time.Now(),
		ShowInstructions: true,
	}

	pdfData, err := pdf.GeneratePDF(pages, opts)
	if err != nil {
		t.Fatalf("GeneratePDF: %v", err)
	}

	if !bytes.HasPrefix(pdfData, []byte("%PDF")) {
		t.Error("generated PDF does not start with %PDF")
	}

	if len(pdfData) < 5000 {
		t.Errorf("PDF suspiciously small: %d bytes", len(pdfData))
	}

	// Now verify QR -> decrypt round-trip from the QR we generated
	img, err := png.Decode(bytes.NewReader(qrPNG))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	qrDecoded, err := pdf.DecodeQRFromImage(img)
	if err != nil {
		t.Fatalf("DecodeQRFromImage: %v", err)
	}
	result, err := crypto.Decrypt(qrDecoded, passphrase)
	if err != nil {
		t.Fatalf("Decrypt from QR: %v", err)
	}
	if !bytes.Equal(result, secret) {
		t.Errorf("PDF QR round-trip failed")
	}

	// Also verify Z85 -> decrypt round-trip
	z85Decoded, err := encode.DecodeZ85(z85Text)
	if err != nil {
		t.Fatalf("DecodeZ85: %v", err)
	}
	result2, err := crypto.Decrypt(z85Decoded, passphrase)
	if err != nil {
		t.Fatalf("Decrypt from Z85: %v", err)
	}
	if !bytes.Equal(result2, secret) {
		t.Errorf("PDF Z85 round-trip failed")
	}
}

// TestFullRoundTripLargePayload tests multi-page split and reassembly.
func TestFullRoundTripLargePayload(t *testing.T) {
	// Use random bytes so zlib can't compress them, ensuring we exceed QR capacity
	secret := make([]byte, 3000)
	if _, err := crand.Read(secret); err != nil {
		t.Fatalf("generating random secret: %v", err)
	}

	passphrase, err := crypto.GeneratePassphrase(6)
	if err != nil {
		t.Fatalf("GeneratePassphrase: %v", err)
	}

	payload, err := crypto.Encrypt(secret, passphrase, 1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Should need multiple chunks
	chunks := pdf.SplitPayload(payload)
	if len(chunks) < 2 {
		t.Skipf("payload (%d bytes) fits in single QR, skipping multi-page test", len(payload))
	}

	t.Logf("Payload: %d bytes, split into %d chunks", len(payload), len(chunks))

	// Encode and decode each chunk via QR
	decodedChunks := make([][]byte, len(chunks))
	for i, chunk := range chunks {
		qrPNG, err := pdf.EncodeQR(chunk)
		if err != nil {
			t.Fatalf("EncodeQR chunk %d: %v", i, err)
		}

		img, err := png.Decode(bytes.NewReader(qrPNG))
		if err != nil {
			t.Fatalf("png.Decode chunk %d: %v", i, err)
		}

		decoded, err := pdf.DecodeQRFromImage(img)
		if err != nil {
			t.Fatalf("DecodeQRFromImage chunk %d: %v", i, err)
		}

		decodedChunks[i] = decoded
	}

	// Reassemble
	reassembled, err := pdf.ReassemblePayload(decodedChunks)
	if err != nil {
		t.Fatalf("ReassemblePayload: %v", err)
	}

	// Decrypt
	result, err := crypto.Decrypt(reassembled, passphrase)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(result, secret) {
		t.Errorf("large payload round-trip failed: got %d bytes, want %d", len(result), len(secret))
	}
}
