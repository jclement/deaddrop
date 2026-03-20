package pdf

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/jclement/deaddrop/internal/encode"
	"github.com/jclement/deaddrop/internal/fonts"
	"github.com/jclement/deaddrop/internal/ui"
)

// Layout constants (mm)
const (
	marginLeft   = 15.0
	marginRight  = 15.0
	marginTop    = 15.0
	marginBottom = 12.0
	pageWidth    = 210.0 // A4
	pageHeight   = 297.0
	contentWidth = pageWidth - marginLeft - marginRight
	z85LineWidth = 50    // characters per line in Z85 block
	z85ColWidth  = 75.0  // just wide enough for 50 monospace chars + padding
	qrZ85Gap     = 4.0   // gap between QR and Z85 columns
	qrImageSize  = contentWidth - z85ColWidth - qrZ85Gap // QR fills remaining width
)

// Font names (registered with fpdf)
const (
	fontRoboto     = "Roboto"
	fontRobotoMono = "RobotoMono"
)

// PDFOptions holds configuration for PDF generation.
type PDFOptions struct {
	Label            string
	Title            string // Centered title on the page
	Date             time.Time
	OutputPath       string
	ShowInstructions bool
}

// PageData holds the data for a single PDF page.
type PageData struct {
	QRCodePNG  []byte
	Z85Text    string
	PageNumber int
	TotalPages int
}

// GeneratePDF creates a PDF document with the given pages and options.
func GeneratePDF(pages []PageData, opts PDFOptions) ([]byte, error) {
	p := fpdf.New("P", "mm", "A4", "")
	p.SetMargins(marginLeft, marginTop, marginRight)
	p.SetAutoPageBreak(false, marginBottom)

	// Register embedded fonts
	p.AddUTF8FontFromBytes(fontRoboto, "", fonts.Roboto)
	p.AddUTF8FontFromBytes(fontRoboto, "B", fonts.Roboto)
	p.AddUTF8FontFromBytes(fontRobotoMono, "", fonts.RobotoMono)

	for i, page := range pages {
		p.AddPage()
		isFirstPage := i == 0

		drawHeader(p, opts)
		drawQRAndZ85(p, page)

		if isFirstPage {
			drawPassphraseField(p)
		}

		if opts.ShowInstructions && isFirstPage {
			drawRestoreInstructions(p, len(pages))
		}

		drawFooter(p, opts)

		if page.TotalPages > 1 {
			drawPageNumber(p, page.PageNumber, page.TotalPages)
		}
	}

	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, fmt.Errorf("generating PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// GeneratePDFToFile creates a PDF and writes it to the specified path.
func GeneratePDFToFile(pages []PageData, opts PDFOptions) error {
	data, err := GeneratePDF(pages, opts)
	if err != nil {
		return err
	}
	return os.WriteFile(opts.OutputPath, data, 0o644)
}

func drawHeader(p *fpdf.Fpdf, opts PDFOptions) {
	db := ui.PDFDeepBlue
	dg := ui.PDFDarkGray

	// Page heading: title > label > "Dead Drop"
	heading := "Dead Drop"
	if opts.Label != "" {
		heading = opts.Label
	}
	if opts.Title != "" {
		heading = opts.Title
	}

	p.SetFont(fontRoboto, "B", 22)
	p.SetTextColor(db[0], db[1], db[2])
	p.SetY(marginTop)
	p.SetX(marginLeft)
	p.CellFormat(contentWidth/2, 10, heading, "", 0, "L", false, 0, "")

	// Timestamp right-aligned
	p.SetFont(fontRoboto, "", 9)
	p.SetTextColor(dg[0], dg[1], dg[2])
	p.CellFormat(contentWidth/2, 10, opts.Date.Format("2006-01-02 15:04"), "", 1, "R", false, 0, "")

	// Horizontal rule
	p.SetDrawColor(db[0], db[1], db[2])
	p.SetLineWidth(0.5)
	y := p.GetY() + 1
	p.Line(marginLeft, y, pageWidth-marginRight, y)
	p.SetY(y + 4)
}

func drawQRAndZ85(p *fpdf.Fpdf, page PageData) {
	startY := p.GetY()
	bottomY := startY // tracks the bottom of whichever column is taller

	// === Left column: QR code ===
	if len(page.QRCodePNG) > 0 {
		name := fmt.Sprintf("qr_%d", time.Now().UnixNano())
		imgOpts := fpdf.ImageOptions{ImageType: "png", ReadDpi: true}
		p.RegisterImageOptionsReader(name, imgOpts, bytes.NewReader(page.QRCodePNG))
		p.ImageOptions(name, marginLeft, startY, qrImageSize, qrImageSize, false, imgOpts, 0, "")
		if startY+qrImageSize > bottomY {
			bottomY = startY + qrImageSize
		}
	}

	// === Right column: Z85 text ===
	if page.Z85Text != "" {
		db := ui.PDFDeepBlue
		dg := ui.PDFDarkGray
		lg := ui.PDFLightGray
		colX := marginLeft + qrImageSize + qrZ85Gap

		// Section heading
		p.SetFont(fontRoboto, "B", 8)
		p.SetTextColor(db[0], db[1], db[2])
		p.SetXY(colX, startY)
		p.CellFormat(z85ColWidth, 4, "Encoded Payload (fallback)", "", 1, "L", false, 0, "")

		// Intro text
		p.SetFont(fontRoboto, "", 6.5)
		p.SetTextColor(dg[0], dg[1], dg[2])
		p.SetXY(colX, p.GetY()+1)
		// Use manual word wrapping within the column
		p.SetLeftMargin(colX)
		p.MultiCell(z85ColWidth, 3,
			"If the QR code is damaged, manually type this Z85-encoded text and decode it.",
			"", "L", false)
		p.SetLeftMargin(marginLeft)
		p.SetY(p.GetY() + 1.5)

		// Z85 text block with alternating row backgrounds
		lines := encode.FormatZ85Block(page.Z85Text, z85LineWidth)
		p.SetFont(fontRobotoMono, "", 6.5)

		blockY := p.GetY()
		lineHeight := 3.2

		// Draw background box
		blockHeight := float64(len(lines))*lineHeight + 2
		p.SetDrawColor(dg[0], dg[1], dg[2])
		p.SetLineWidth(0.2)
		p.Rect(colX, blockY, z85ColWidth, blockHeight, "D")

		for i, line := range lines {
			y := blockY + 1 + float64(i)*lineHeight

			if i%2 == 1 {
				p.SetFillColor(lg[0], lg[1], lg[2])
				p.Rect(colX+0.2, y, z85ColWidth-0.4, lineHeight, "F")
			}

			p.SetTextColor(0, 0, 0)
			p.SetXY(colX+2, y)
			p.CellFormat(z85ColWidth-4, lineHeight, line, "", 0, "L", false, 0, "")
		}

		z85Bottom := blockY + blockHeight
		if z85Bottom > bottomY {
			bottomY = z85Bottom
		}
	}

	p.SetY(bottomY + 5)
}

func drawPassphraseField(p *fpdf.Fpdf) {
	db := ui.PDFDeepBlue
	dg := ui.PDFDarkGray

	y := p.GetY()
	boxHeight := 32.0

	// Draw bordered box with deep blue border
	p.SetDrawColor(db[0], db[1], db[2])
	p.SetLineWidth(0.6)
	p.Rect(marginLeft, y, contentWidth, boxHeight, "D")

	// "PASSPHRASE" heading inside the box
	p.SetY(y + 3)
	p.SetX(marginLeft + 5)
	p.SetFont(fontRoboto, "B", 11)
	p.SetTextColor(db[0], db[1], db[2])
	p.CellFormat(contentWidth-10, 6, "PASSPHRASE", "", 1, "L", false, 0, "")

	// Blank lines for handwriting
	mg := ui.PDFMedGray
	p.SetDrawColor(mg[0], mg[1], mg[2])
	p.SetLineWidth(0.3)
	lineY := p.GetY() + 3
	p.Line(marginLeft+5, lineY, pageWidth-marginRight-5, lineY)
	lineY += 7
	p.Line(marginLeft+5, lineY, pageWidth-marginRight-5, lineY)

	// Instruction text
	p.SetY(y + boxHeight - 7)
	p.SetX(marginLeft + 5)
	p.SetFont(fontRoboto, "", 7)
	p.SetTextColor(dg[0], dg[1], dg[2])
	p.CellFormat(contentWidth-10, 4, "Write the passphrase here by hand. Do NOT type or print it.", "", 1, "L", false, 0, "")

	p.SetY(y + boxHeight + 4)
}

func drawRestoreInstructions(p *fpdf.Fpdf, totalPages int) {
	db := ui.PDFDeepBlue
	dg := ui.PDFDarkGray

	const (
		fontSize = 6.5
		lineH    = 3.0
	)

	// Section heading
	p.SetFont(fontRoboto, "B", 9)
	p.SetTextColor(db[0], db[1], db[2])
	p.SetX(marginLeft)
	p.CellFormat(contentWidth, 5, "Restoring This Document", "", 1, "L", false, 0, "")
	p.SetY(p.GetY() + 1)

	p.SetFont(fontRoboto, "", fontSize)
	p.SetTextColor(dg[0], dg[1], dg[2])
	p.SetLeftMargin(marginLeft)

	paras := []string{
		"This document is self-contained. You do not need the Dead Drop tool to recover your secret -- " +
			"only a standard age decryptor (age-encryption.org) and the handwritten passphrase above.",

		"The QR code and the Z85 text block both contain the same encrypted payload. " +
			"The QR code holds the payload as a base64-encoded string; scan it with any QR reader and base64-decode the result. " +
			"The Z85 text is a ZeroMQ Base-85 encoding of the same data; type it into a file and decode it with any Z85-compatible tool.",

		"Both paths produce identical binary output. The first four bytes are the ASCII header \"DD01\", which identifies the format " +
			"and should be stripped. Everything after that is a standard age-encrypted file using a scrypt passphrase recipient " +
			"(work factor 18). Save it with an .age extension and decrypt it with any age-compatible tool using the passphrase above.",

		"Z85 padding note: the Z85 block uses a thin padding layer so the data aligns to 4 bytes. The very first byte after " +
			"Z85-decoding is a pad count (0--3). Skip that byte and the four DD01 header bytes (5 bytes total from the start), " +
			"then trim that many zero bytes from the end. The remainder is the age ciphertext.",
	}

	if totalPages > 1 {
		paras = append(paras, fmt.Sprintf(
			"Multi-page note: this document spans %d pages. Each page contains a different chunk of the payload, not a copy. "+
				"The QR codes carry a \"DS\" header with chunk index and total count. Decode all %d QR codes (or Z85 blocks), "+
				"strip the 4-byte DS header from each, and concatenate the data in page order to reconstruct the full payload. "+
				"Then proceed with DD01 stripping and age decryption as described above.",
			totalPages, totalPages))
	}

	for i, para := range paras {
		p.SetX(marginLeft)
		p.MultiCell(contentWidth, lineH, para, "", "L", false)
		if i < len(paras)-1 {
			p.SetY(p.GetY() + 1)
		}
	}
}

func drawFooter(p *fpdf.Fpdf, opts PDFOptions) {
	dg := ui.PDFDarkGray
	p.SetFont(fontRoboto, "", 7)
	p.SetTextColor(dg[0], dg[1], dg[2])
	p.SetXY(marginLeft, pageHeight-marginBottom)

	// Show label on the left if it exists (source filename, etc.)
	if opts.Label != "" {
		p.CellFormat(contentWidth/2, 4, opts.Label, "", 0, "L", false, 0, "")
	}

	p.SetXY(marginLeft+contentWidth/2, pageHeight-marginBottom)
	p.CellFormat(contentWidth/2, 4, "Generated by Dead Drop", "", 0, "R", false, 0, "")
}

func drawPageNumber(p *fpdf.Fpdf, page, total int) {
	dg := ui.PDFDarkGray
	p.SetFont(fontRoboto, "", 7)
	p.SetTextColor(dg[0], dg[1], dg[2])
	p.SetXY(marginLeft, pageHeight-marginBottom)
	p.CellFormat(contentWidth, 4, fmt.Sprintf("Page %d of %d", page, total), "", 0, "C", false, 0, "")
}
