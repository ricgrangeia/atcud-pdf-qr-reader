package pdf

import (
	"fmt"
	"image"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"os"

	"github.com/makiuchi-d/gozxing"
	qrmulti "github.com/makiuchi-d/gozxing/multi/qrcode"
)

// RawQRCode is the output of the scanner: the decoded text and which page it came from.
// The application layer turns this into a domain QRCode after applying business rules.
type RawQRCode struct {
	Content    string
	PageNumber int
}

// ExtractQRCodes scans every page of a PDF and returns all decoded QR codes.
//
// Flow:
//  1. Write the PDF bytes to a temp file.
//  2. Render each page to a PNG image with pdftoppm.
//  3. Scan each PNG for QR codes using the ZXing library (gozxing).
//  4. Return all decoded results, preserving the page number.
func ExtractQRCodes(pdfBytes []byte) ([]RawQRCode, error) {
	// Write PDF to disk so pdftoppm can process it.
	tmpFile, err := os.CreateTemp("", "input_*.pdf")
	if err != nil {
		return nil, fmt.Errorf("creating temp PDF file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err = tmpFile.Write(pdfBytes); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("writing PDF to temp file: %w", err)
	}
	tmpFile.Close()

	// Render all PDF pages to PNG images.
	pageFiles, cleanup, err := renderPagesToImages(tmpFile.Name())
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var results []RawQRCode

	for pageIdx, imagePath := range pageFiles {
		pageNumber := pageIdx + 1

		codes, scanErr := scanImageForQRCodes(imagePath)
		if scanErr != nil {
			// A single bad page should not abort the whole document.
			// The caller will see fewer results, not an error.
			continue
		}

		for _, code := range codes {
			results = append(results, RawQRCode{
				Content:    code,
				PageNumber: pageNumber,
			})
		}
	}

	return results, nil
}

// scanImageForQRCodes finds every QR code in one image file and returns their decoded text.
func scanImageForQRCodes(imagePath string) ([]string, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("opening image %s: %w", imagePath, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decoding image %s: %w", imagePath, err)
	}

	// Convert the image to the binary bitmap format that ZXing expects.
	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil, fmt.Errorf("creating bitmap from image: %w", err)
	}

	// TRY_HARDER makes ZXing attempt more aggressive scanning strategies,
	// which improves detection for small or slightly skewed QR codes.
	hints := map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_TRY_HARDER: true,
	}

	// QRCodeMultiReader finds all QR codes in one image (not just the first one).
	reader := qrmulti.NewQRCodeMultiReader()
	decoded, err := reader.DecodeMultiple(bitmap, hints)
	if err != nil {
		// "not found" is not an error — the page just has no QR codes.
		return nil, nil
	}

	texts := make([]string, 0, len(decoded))
	for _, result := range decoded {
		texts = append(texts, result.GetText())
	}

	return texts, nil
}
