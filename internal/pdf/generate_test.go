package pdf_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	goqrcode "github.com/skip2/go-qrcode"

	"github.com/jclement/deaddrop/internal/pdf"
)

func testQRPNG(t *testing.T) []byte {
	t.Helper()
	png, err := goqrcode.Encode("test payload data", goqrcode.Low, 256)
	if err != nil {
		t.Fatalf("generating test QR: %v", err)
	}
	return png
}

func TestGeneratePDFSinglePage(t *testing.T) {
	pages := []pdf.PageData{
		{
			QRCodePNG:  testQRPNG(t),
			Z85Text:    "rA&H9]o3vB%kW8mP#xQ2nL7jF!dY5tC0gU4sE6wR1bN9iM3aK8pJ2cX7hV5fT0zD",
			PageNumber: 1,
			TotalPages: 1,
		},
	}
	opts := pdf.PDFOptions{
		Label:            "test-secret",
		Title:            "My Secret Backup",
		Date:             time.Date(2026, 3, 20, 14, 30, 0, 0, time.UTC),
		ShowInstructions: true,
	}

	data, err := pdf.GeneratePDF(pages, opts)
	if err != nil {
		t.Fatalf("GeneratePDF: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Error("output does not start with %PDF header")
	}

	if len(data) < 1000 {
		t.Errorf("PDF suspiciously small: %d bytes", len(data))
	}
}

func TestGeneratePDFMultiPage(t *testing.T) {
	qrPNG := testQRPNG(t)
	pages := []pdf.PageData{
		{QRCodePNG: qrPNG, Z85Text: "page1data", PageNumber: 1, TotalPages: 2},
		{QRCodePNG: qrPNG, Z85Text: "page2data", PageNumber: 2, TotalPages: 2},
	}
	opts := pdf.PDFOptions{
		Label:            "multi-page-test",
		Title:            "Split Secret",
		Date:             time.Now(),
		ShowInstructions: true,
	}

	data, err := pdf.GeneratePDF(pages, opts)
	if err != nil {
		t.Fatalf("GeneratePDF: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Error("output does not start with %PDF header")
	}
}

func TestGeneratePDFToFile(t *testing.T) {
	pages := []pdf.PageData{
		{
			QRCodePNG:  testQRPNG(t),
			Z85Text:    "testdata",
			PageNumber: 1,
			TotalPages: 1,
		},
	}
	outPath := filepath.Join(t.TempDir(), "test.pdf")
	opts := pdf.PDFOptions{
		Label:            "file-test",
		Date:             time.Now(),
		OutputPath:       outPath,
		ShowInstructions: true,
	}

	if err := pdf.GeneratePDFToFile(pages, opts); err != nil {
		t.Fatalf("GeneratePDFToFile: %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if info.Size() < 1000 {
		t.Errorf("PDF file suspiciously small: %d bytes", info.Size())
	}
}

func TestGeneratePDFNoInstructions(t *testing.T) {
	pages := []pdf.PageData{
		{
			QRCodePNG:  testQRPNG(t),
			Z85Text:    "testdata",
			PageNumber: 1,
			TotalPages: 1,
		},
	}
	opts := pdf.PDFOptions{
		Label:            "no-instructions",
		Date:             time.Now(),
		ShowInstructions: false,
	}

	data, err := pdf.GeneratePDF(pages, opts)
	if err != nil {
		t.Fatalf("GeneratePDF: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Error("output does not start with %PDF header")
	}
}
