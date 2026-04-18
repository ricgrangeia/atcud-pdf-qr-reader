package pdf

import (
	"os/exec"
	"regexp"
	"strings"
)

var (
	// reInvoiceLine matches the structured invoice line used in Via Verde / AT documents:
	// "Nº de Fatura: FT BR2026/003227524 Data de Emissão: 2026-03-31 ATCUD: J6FJT2C9-003227524"
	reInvoiceLine = regexp.MustCompile(`Nº de Fatura:\s*(.+?)\s+Data de Emissão:\s*(\d{4}-\d{2}-\d{2})\s+ATCUD:\s*([A-Z0-9]+-\d+)`)

	// reNIFCliente matches "NIF Cliente: 213195755"
	reNIFCliente = regexp.MustCompile(`NIF Cliente:\s*(\d{9})`)

	// reNIPC matches "MCRC ... - NIPC 502790024"
	reNIPC = regexp.MustCompile(`NIPC\s+(\d{9})`)

	// reTotalLine matches "Total em Portagens 41,35" capturing the amount
	reTotalLine = regexp.MustCompile(`^Total em \S.*\s+([\d]+[,\.][\d]+)\s*$`)

	// reIVALine matches "IVA incluído à taxa normal em vigor 7,73" capturing the amount
	reIVALine = regexp.MustCompile(`IVA inclu[ií]do.*\s+([\d]+[,\.][\d]+)\s*$`)

	// reATCUDSimple is the fallback: any line containing "ATCUD: CODE-NUM"
	reATCUDSimple = regexp.MustCompile(`ATCUD:\s*[A-Z0-9]+-\d+`)
)

// extractATCUDsFromText uses pdftotext to read the PDF text layer.
// For each entry it tries to reconstruct an AT fiscal QR format string from the
// surrounding lines; unstructured ATCUD lines fall back to the raw text content.
func extractATCUDsFromText(pdfPath string) []RawQRCode {
	out, err := exec.Command("pdftotext", "-layout", pdfPath, "-").Output()
	if err != nil {
		return nil
	}

	var results []RawQRCode
	// pdftotext separates pages with a form-feed character.
	for pageIdx, pageText := range strings.Split(string(out), "\f") {
		if strings.TrimSpace(pageText) == "" {
			continue
		}
		pageNumber := pageIdx + 1
		lines := strings.Split(pageText, "\n")

		structured := make(map[int]bool)

		// Pass 1: structured extraction — reconstruct AT QR format from invoice lines.
		for i, line := range lines {
			if !reInvoiceLine.MatchString(line) {
				continue
			}
			structured[i] = true
			results = append(results, RawQRCode{
				Content:    reconstructATFormat(lines, i),
				PageNumber: pageNumber,
			})
		}

		// Pass 2: fallback for any remaining ATCUD-bearing lines not already captured.
		for i, line := range lines {
			if structured[i] {
				continue
			}
			if reATCUDSimple.MatchString(line) {
				results = append(results, RawQRCode{
					Content:    strings.TrimSpace(line),
					PageNumber: pageNumber,
				})
			}
		}
	}
	return results
}

// reconstructATFormat builds an AT fiscal QR string ("A:NIF*B:NIF*C:PT*...")
// from the invoice line at atcudLineIdx and the supporting lines that follow it.
func reconstructATFormat(lines []string, atcudLineIdx int) string {
	m := reInvoiceLine.FindStringSubmatch(lines[atcudLineIdx])
	if m == nil {
		return strings.TrimSpace(lines[atcudLineIdx])
	}

	invoiceNum := strings.TrimSpace(m[1])           // "FT BR2026/003227524"
	date := strings.ReplaceAll(m[2], "-", "")       // "20260331"
	atcud := m[3]                                    // "J6FJT2C9-003227524"
	docType := invoiceNum
	if idx := strings.Index(invoiceNum, " "); idx > 0 {
		docType = invoiceNum[:idx] // "FT"
	}

	var adquirenteNIF, emitenteNIF, total, iva string

	end := atcudLineIdx + 1 + 15
	if end > len(lines) {
		end = len(lines)
	}
	for _, l := range lines[atcudLineIdx+1 : end] {
		// Stop when the next sub-document begins.
		if reInvoiceLine.MatchString(l) {
			break
		}
		t := strings.TrimSpace(l)
		if adquirenteNIF == "" {
			if mm := reNIFCliente.FindStringSubmatch(t); mm != nil {
				adquirenteNIF = mm[1]
			}
		}
		if emitenteNIF == "" {
			if mm := reNIPC.FindStringSubmatch(t); mm != nil {
				emitenteNIF = mm[1]
			}
		}
		if total == "" {
			if mm := reTotalLine.FindStringSubmatch(t); mm != nil {
				total = strings.ReplaceAll(mm[1], ",", ".")
			}
		}
		if iva == "" {
			if mm := reIVALine.FindStringSubmatch(t); mm != nil {
				iva = strings.ReplaceAll(mm[1], ",", ".")
			}
		}
	}

	var parts []string
	if emitenteNIF != "" {
		parts = append(parts, "A:"+emitenteNIF)
	}
	if adquirenteNIF != "" {
		parts = append(parts, "B:"+adquirenteNIF)
	}
	parts = append(parts, "C:PT")
	if docType != "" {
		parts = append(parts, "D:"+docType+" ") // AT spec includes trailing space after type
	}
	parts = append(parts, "F:"+date)
	parts = append(parts, "G:"+invoiceNum)
	parts = append(parts, "H:"+atcud)
	if iva != "" {
		parts = append(parts, "N:"+iva)
	}
	if total != "" {
		parts = append(parts, "O:"+total)
	}

	return strings.Join(parts, "*")
}
