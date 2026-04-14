// Package pdf provides tools for extracting data from PDF files.
package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

// renderPagesToImages converts every page of a PDF into a PNG file.
//
// It uses pdftoppm (from poppler-utils) because it is a mature, well-tested
// renderer that produces high-quality raster images suitable for QR scanning.
// The resolution of 200 DPI is a good balance between speed and scan accuracy.
//
// Returns:
//   - sorted list of PNG file paths (one per page)
//   - a cleanup function that removes the temporary directory
func renderPagesToImages(pdfPath string) (pages []string, cleanup func(), err error) {
	tempDir, err := os.MkdirTemp("", "pdf_render_*")
	if err != nil {
		return nil, nil, fmt.Errorf("creating temp directory: %w", err)
	}

	cleanup = func() { os.RemoveAll(tempDir) }

	outputPrefix := filepath.Join(tempDir, "page")

	// pdftoppm flags:
	//   -png       → output PNG images
	//   -r 200     → 200 DPI (good for QR code detection)
	cmd := exec.Command("pdftoppm", "-png", "-r", "200", pdfPath, outputPrefix)
	if out, cmdErr := cmd.CombinedOutput(); cmdErr != nil {
		cleanup()
		return nil, nil, fmt.Errorf(
			"pdftoppm failed (is poppler-utils installed?): %w\nOutput: %s",
			cmdErr, string(out),
		)
	}

	// pdftoppm names files like "page-01.png", "page-02.png", etc.
	files, err := filepath.Glob(filepath.Join(tempDir, "page-*.png"))
	if err != nil || len(files) == 0 {
		// Fallback: some versions omit the dash
		files, err = filepath.Glob(filepath.Join(tempDir, "page*.png"))
	}
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("finding rendered page images: %w", err)
	}
	if len(files) == 0 {
		cleanup()
		return nil, nil, fmt.Errorf("pdftoppm produced no output for the given PDF")
	}

	// Sort so pages are processed in document order.
	sort.Strings(files)

	return files, cleanup, nil
}
