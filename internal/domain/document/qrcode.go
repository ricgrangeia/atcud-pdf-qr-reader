// Package document holds the core business concepts for fiscal document processing.
package document

// QRCode represents a single QR code found and decoded from a PDF document.
// It carries the raw decoded text plus any ATCUD information found inside it.
type QRCode struct {
	// Content is the full decoded text from the QR code.
	Content string `json:"content"`

	// PageNumber is the PDF page where this QR code was found (1-based).
	PageNumber int `json:"page_number"`

	// HasATCUD is true when an ATCUD code was identified inside Content.
	HasATCUD bool `json:"has_atcud"`

	// ATCUD is the extracted ATCUD value (e.g. "ABCD1234-1").
	// Empty when HasATCUD is false.
	ATCUD string `json:"atcud,omitempty"`

}
