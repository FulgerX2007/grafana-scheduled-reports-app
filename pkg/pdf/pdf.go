package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// Options holds PDF generation options
type Options struct {
	Title       string
	Orientation string // "P" or "L"
	PageSize    string // "A4", "Letter"
	Header      string
	Footer      string
}

// Generator handles PDF generation
type Generator struct{}

// NewGenerator creates a new PDF generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate creates a PDF from PNG images
func (g *Generator) Generate(images [][]byte, opts Options) ([]byte, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("no images provided")
	}

	orientation := "L"
	if opts.Orientation == "portrait" {
		orientation = "P"
	}

	pageSize := opts.PageSize
	if pageSize == "" {
		pageSize = "A4"
	}

	pdf := gofpdf.New(orientation, "mm", pageSize, "")

	// Set document metadata
	pdf.SetTitle(opts.Title, true)
	pdf.SetCreator("Grafana Reporting Plugin", true)
	pdf.SetCreationDate(time.Now())

	// Set header and footer if provided
	if opts.Header != "" {
		pdf.SetHeaderFunc(func() {
			pdf.SetFont("Arial", "", 10)
			pdf.Cell(0, 10, opts.Header)
			pdf.Ln(12)
		})
	}

	if opts.Footer != "" {
		pdf.SetFooterFunc(func() {
			pdf.SetY(-15)
			pdf.SetFont("Arial", "I", 8)
			pdf.Cell(0, 10, fmt.Sprintf("%s - Page %d", opts.Footer, pdf.PageNo()))
		})
	}

	// Add each image as a page
	for i, imgData := range images {
		pdf.AddPage()

		// Create a temporary image reader
		reader := bytes.NewReader(imgData)
		imgOpts := gofpdf.ImageOptions{
			ImageType: "PNG",
			ReadDpi:   true,
		}

		// Calculate image dimensions to fit page
		var pageWidth, pageHeight float64
		if orientation == "L" {
			pageWidth, pageHeight = 297, 210 // A4 landscape
		} else {
			pageWidth, pageHeight = 210, 297 // A4 portrait
		}

		if pageSize == "Letter" {
			if orientation == "L" {
				pageWidth, pageHeight = 279.4, 215.9
			} else {
				pageWidth, pageHeight = 215.9, 279.4
			}
		}

		// Add some margins
		margin := 10.0
		imgWidth := pageWidth - (2 * margin)
		imgHeight := pageHeight - (2 * margin)

		// Register and insert the image
		imgName := fmt.Sprintf("image_%d", i)
		pdf.RegisterImageOptionsReader(imgName, imgOpts, reader)
		pdf.ImageOptions(imgName, margin, margin, imgWidth, imgHeight, false, imgOpts, 0, "")
	}

	// Output PDF to buffer
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}
