package document

import (
	"regexp"
	"strings"
)

// atcudFieldRegex matches the H: field in Portuguese fiscal QR codes.
// Portuguese invoices encode ATCUD as "H:VALIDATIONCODE-SEQUENTIALNUM".
// The full QR string is a sequence of key:value pairs separated by "*".
// Examples:
//   - H:CSDF7T5H-1      (normal invoice, sequence 1)
//   - H:0-1             (series not yet registered with AT)
var atcudFieldRegex = regexp.MustCompile(`(?:^|\*)H:([A-Z0-9]+-\d+)(?:\*|$)`)

// atcudPrefixRegex matches the plain "ATCUD:CODE-NUM" format sometimes
// used in non-structured QR codes or document annotations.
var atcudPrefixRegex = regexp.MustCompile(`[A-Z0-9]+-\d+`)

// DetectATCUD inspects the decoded text of a QR code and returns the
// ATCUD value when one is found.
//
// It supports two formats:
//  1. Structured Portuguese fiscal QR (field H:)  — e.g. "A:NIF*...*H:XXXX-1*..."
//  2. Plain prefix format                         — e.g. "ATCUD:XXXX-1"
func DetectATCUD(content string) (atcud string, found bool) {
	// --- Format 1: structured fiscal QR code (AT specification) ---
	if matches := atcudFieldRegex.FindStringSubmatch(content); len(matches) > 1 {
		return matches[1], true
	}

	// --- Format 2: plain "ATCUD:" prefix ---
	idx := strings.Index(content, "ATCUD:")
	if idx >= 0 {
		rest := content[idx+len("ATCUD:"):]
		if match := atcudPrefixRegex.FindString(rest); match != "" {
			return match, true
		}
	}

	return "", false
}
