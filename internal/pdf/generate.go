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
	marginLeft   = 25.0
	marginRight  = 25.0
	marginTop    = 20.0
	marginBottom = 15.0
	pageWidth    = 210.0 // A4
	pageHeight   = 297.0
	contentWidth = pageWidth - marginLeft - marginRight
	qrImageSize  = 65.0 // QR code display size in mm
	z85LineWidth = 60    // characters per line in Z85 block
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
		drawQRCode(p, page.QRCodePNG)

		if isFirstPage {
			drawPassphraseField(p)
		}

		drawZ85Block(p, page.Z85Text)

		if opts.ShowInstructions && isFirstPage {
			drawRestoreInstructions(p)
		}

		drawFooter(p)

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

	// "DEAD DROP" title
	p.SetFont(fontRoboto, "B", 22)
	p.SetTextColor(db[0], db[1], db[2])
	p.SetY(marginTop)
	p.SetX(marginLeft)
	p.CellFormat(contentWidth/2, 10, "DEAD DROP", "", 0, "L", false, 0, "")

	// Timestamp right-aligned
	p.SetFont(fontRoboto, "", 9)
	dg := ui.PDFDarkGray
	p.SetTextColor(dg[0], dg[1], dg[2])
	p.CellFormat(contentWidth/2, 10, opts.Date.Format("2006-01-02 15:04"), "", 1, "R", false, 0, "")

	// Horizontal rule
	p.SetDrawColor(db[0], db[1], db[2])
	p.SetLineWidth(0.5)
	y := p.GetY() + 1
	p.Line(marginLeft, y, pageWidth-marginRight, y)
	p.SetY(y + 4)

	// Title (centered, if provided)
	if opts.Title != "" {
		p.SetFont(fontRoboto, "B", 14)
		p.SetTextColor(db[0], db[1], db[2])
		p.CellFormat(contentWidth, 8, opts.Title, "", 1, "C", false, 0, "")
		p.SetY(p.GetY() + 2)
	}

	// Label and algorithm metadata
	p.SetFont(fontRoboto, "", 9)
	p.SetTextColor(dg[0], dg[1], dg[2])
	if opts.Label != "" {
		p.SetX(marginLeft)
		p.CellFormat(contentWidth, 5, fmt.Sprintf("Label: %s", opts.Label), "", 1, "L", false, 0, "")
	}
	p.SetX(marginLeft)
	p.CellFormat(contentWidth, 5, "Algorithm: age (scrypt, work factor 18)", "", 1, "L", false, 0, "")
	p.SetY(p.GetY() + 3)
}

func drawQRCode(p *fpdf.Fpdf, qrPNG []byte) {
	if len(qrPNG) == 0 {
		return
	}

	// Register the QR code image
	name := fmt.Sprintf("qr_%d", time.Now().UnixNano())
	opts := fpdf.ImageOptions{ImageType: "png", ReadDpi: true}
	p.RegisterImageOptionsReader(name, opts, bytes.NewReader(qrPNG))

	// Center the QR code
	x := (pageWidth - qrImageSize) / 2
	y := p.GetY()
	p.ImageOptions(name, x, y, qrImageSize, qrImageSize, false, opts, 0, "")
	p.SetY(y + qrImageSize + 5)
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

func drawZ85Block(p *fpdf.Fpdf, z85Text string) {
	if z85Text == "" {
		return
	}

	db := ui.PDFDeepBlue
	dg := ui.PDFDarkGray
	lg := ui.PDFLightGray

	// Section heading
	p.SetFont(fontRoboto, "B", 9)
	p.SetTextColor(db[0], db[1], db[2])
	p.SetX(marginLeft)
	p.CellFormat(contentWidth, 5, "Encoded Payload (fallback)", "", 1, "L", false, 0, "")

	// Intro text
	p.SetFont(fontRoboto, "", 7)
	p.SetTextColor(dg[0], dg[1], dg[2])
	p.SetX(marginLeft)
	p.MultiCell(contentWidth, 3.5,
		"If the QR code is damaged or unscannable, manually type this Z85-encoded text and decode it.",
		"", "L", false)
	p.SetY(p.GetY() + 2)

	// Z85 text block with alternating row backgrounds
	lines := encode.FormatZ85Block(z85Text, z85LineWidth)
	p.SetFont(fontRobotoMono, "", 7)

	blockY := p.GetY()
	lineHeight := 3.5

	// Draw background box
	blockHeight := float64(len(lines))*lineHeight + 2
	p.SetDrawColor(dg[0], dg[1], dg[2])
	p.SetLineWidth(0.2)
	p.Rect(marginLeft, blockY, contentWidth, blockHeight, "D")

	for i, line := range lines {
		y := blockY + 1 + float64(i)*lineHeight

		// Alternating row background
		if i%2 == 1 {
			p.SetFillColor(lg[0], lg[1], lg[2])
			p.Rect(marginLeft+0.2, y, contentWidth-0.4, lineHeight, "F")
		}

		p.SetTextColor(0, 0, 0)
		p.SetXY(marginLeft+2, y)
		p.CellFormat(contentWidth-4, lineHeight, line, "", 0, "L", false, 0, "")
	}

	p.SetY(blockY + blockHeight + 5)
}

func drawRestoreInstructions(p *fpdf.Fpdf) {
	db := ui.PDFDeepBlue
	dg := ui.PDFDarkGray

	// Check if we have enough space, otherwise skip
	if p.GetY() > pageHeight-60 {
		return
	}

	// Section heading
	p.SetFont(fontRoboto, "B", 9)
	p.SetTextColor(db[0], db[1], db[2])
	p.SetX(marginLeft)
	p.CellFormat(contentWidth, 5, "Restore Without Dead Drop", "", 1, "L", false, 0, "")
	p.SetY(p.GetY() + 1)

	// Steps
	p.SetFont(fontRoboto, "", 7.5)
	p.SetTextColor(dg[0], dg[1], dg[2])

	steps := []string{
		"1. Scan the QR code (or type the Z85 block above into a file)",
		"2. If using Z85 text, decode with: deaddrop restore",
		"   Or use any Z85/Base85 decoder to get the binary data",
		"3. Save the binary output to a file (e.g. secret.age)",
		"4. Run: age -d secret.age",
		"5. Enter the passphrase written above",
	}
	for _, step := range steps {
		p.SetX(marginLeft)
		p.CellFormat(contentWidth, 4, step, "", 1, "L", false, 0, "")
	}

	p.SetY(p.GetY() + 2)
	p.SetFont(fontRoboto, "", 7)
	p.SetX(marginLeft)
	p.MultiCell(contentWidth, 3.5,
		"The payload is age-encrypted (age-encryption.org) using a passphrase/scrypt recipient. "+
			"Any age-compatible tool can decrypt it.",
		"", "L", false)
}

func drawFooter(p *fpdf.Fpdf) {
	dg := ui.PDFDarkGray
	p.SetFont(fontRoboto, "", 7)
	p.SetTextColor(dg[0], dg[1], dg[2])
	p.SetXY(marginLeft, pageHeight-marginBottom)
	p.CellFormat(contentWidth, 4, "Generated by Dead Drop", "", 0, "R", false, 0, "")
}

func drawPageNumber(p *fpdf.Fpdf, page, total int) {
	dg := ui.PDFDarkGray
	p.SetFont(fontRoboto, "", 7)
	p.SetTextColor(dg[0], dg[1], dg[2])
	p.SetXY(marginLeft, pageHeight-marginBottom)
	p.CellFormat(contentWidth, 4, fmt.Sprintf("Page %d of %d", page, total), "", 0, "L", false, 0, "")
}
