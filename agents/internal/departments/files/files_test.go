package files

import (
	"context"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

type stubLLM struct{ text string }

func (s stubLLM) Complete(context.Context, llm.Request) (string, error) { return s.text, nil }

func TestLookupFolderCompanyOnly(t *testing.T) {
	if _, ok := LookupFolder("cac", false); ok {
		t.Fatal("cac is company-only")
	}
	f, ok := LookupFolder("cac", true)
	if !ok || f.Label != "CAC" {
		t.Fatalf("cac %+v ok=%v", f, ok)
	}
	if _, ok := LookupFolder("legal", false); !ok {
		t.Fatal("legal on a bet")
	}
}

func TestApplyCatalogDropsInventedSlug(t *testing.T) {
	sd := map[string]any{"project_slug": "made-up", "folder": "legal", "unsure": false}
	ApplyCatalog(sd, map[string]bool{"academy": true})
	if sd["save_filing"] != false || sd["unsure"] != true {
		t.Fatalf("%+v", sd)
	}
}

func TestApplyCatalogAcceptsCompanyCAC(t *testing.T) {
	sd := map[string]any{"project_slug": CompanySlug, "folder": "cac"}
	ApplyCatalog(sd, map[string]bool{"academy": true})
	if sd["save_filing"] != true || sd["unsure"] != false {
		t.Fatalf("%+v", sd)
	}
}

func TestHandleCoercesAndNeverFiles(t *testing.T) {
	s := stubLLM{text: `{"status":"success","output_text":"Put it on a made-up bet.","structured_data":{"project_slug":"not-a-bet","folder":"vault","save_filing":true,"cannot_file":false}}`}
	resp, err := Handle(context.Background(), s, contract.Request{
		TaskDescription: "File this PDF",
		ContextData:     []byte(`{"filename":"paper.pdf","projects":[{"slug":"academy","name":"Academy"}]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StructuredData["cannot_file"] != true || resp.StructuredData["save_filing"] != false {
		t.Fatalf("%+v", resp.StructuredData)
	}
	if got := resp.StructuredData["filename"]; got != "paper.pdf" {
		t.Fatalf("filename %v", got)
	}
}

func TestPath(t *testing.T) {
	if got := Path("academy", "legal", "nda.pdf"); got != "academy/legal/nda.pdf" {
		t.Fatal(got)
	}
	if !strings.HasPrefix(Path("", "cac", ""), CompanySlug+"/") {
		t.Fatal("company default")
	}
}
