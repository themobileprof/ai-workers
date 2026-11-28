// Package pdftext extracts plain text from PDF bytes for ops Telegram attachments.
package pdftext

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
)

const defaultMaxRunes = 24000

// Extract returns readable text from a PDF. Empty when the file has no text layer
// (scanned pages) or is not a PDF. maxRunes clips the result; 0 uses the default.
func Extract(raw []byte, maxRunes int) string {
	if len(raw) < 5 || !bytes.HasPrefix(raw, []byte("%PDF")) {
		return ""
	}
	if maxRunes <= 0 {
		maxRunes = defaultMaxRunes
	}
	r, err := pdf.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil || r == nil {
		return ""
	}
	n := r.NumPage()
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	for i := 1; i <= n; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(text)
		if utf8.RuneCountInString(b.String()) >= maxRunes {
			break
		}
	}
	out := collapseSpace(b.String())
	if out == "" {
		return ""
	}
	runes := []rune(out)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "\n…"
	}
	return out
}

func collapseSpace(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	var b strings.Builder
	prevBlank := false
	for _, line := range strings.Split(s, "\n") {
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			if prevBlank {
				continue
			}
			prevBlank = true
			b.WriteByte('\n')
			continue
		}
		prevBlank = false
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	return strings.TrimSpace(b.String())
}
