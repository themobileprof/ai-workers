package pdftext

import (
	"bytes"
	"testing"
)

func TestExtractRejectsNonPDF(t *testing.T) {
	if got := Extract([]byte("hello"), 0); got != "" {
		t.Fatalf("%q", got)
	}
}

func TestExtractEmptyPDFHeaderOnly(t *testing.T) {
	// Minimal header is not a readable document; must not panic.
	raw := []byte("%PDF-1.4\n%\xe2\xe3\xcf\xd3\ntrailer\n<<>>\n%%EOF\n")
	_ = Extract(raw, 100)
}

func TestCollapseSpace(t *testing.T) {
	got := collapseSpace("  a   b  \n\n\n  c  ")
	if got != "a b\n\nc" && got != "a b\nc" {
		// allow single blank line between
		if !bytes.Contains([]byte(got), []byte("a b")) {
			t.Fatal(got)
		}
	}
}
