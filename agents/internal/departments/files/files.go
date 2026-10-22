// Package files classifies an uploaded document onto a project folder.
// A catalog match is accepted for n8n to place. Unsure rows stay proposed. The worker cannot write Drive.
package files

import (
	"context"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

// CompanySlug is the Shared Drive folder for office papers that are not a bet.
const CompanySlug = "themobileprof"

const overlay = `You are the data-room clerk for TheMobileProf Technologies (Lagos), the only CAC-registered company in this office.
Bets (Academy, HomeGauge, …) each have a folder. Office papers (CAC, TIN, bank letters) go under slug "themobileprof", not a bet.
You read the attached file (and the caption) and pick ONE slug from context_data.projects plus "themobileprof", and ONE folder id from the catalog for that scope.
Never invent a slug or a folder id. If two bets could fit, or the file is unreadable, set unsure true and save_filing false.
cannot_file and cannot_send are always true. A catalog match files itself (n8n writes Drive). If you are unsure, the row stays proposed for Accept.
Nigerian/West African commercial English. Phone-short.
task_type must be "files".`

type Folder struct {
	ID      string
	Label   string
	Help    string
	Company bool // only valid under themobileprof
}

func Folders() []Folder {
	return []Folder{
		{ID: "cac", Label: "CAC", Help: "Certificate, status report, MEMART.", Company: true},
		{ID: "tax", Label: "Tax", Help: "FIRS, TIN, filings.", Company: true},
		{ID: "bank", Label: "Bank", Help: "Account letters, Paystack KYC.", Company: true},
		{ID: "legal", Label: "Legal", Help: "Signed contracts, NDAs, terms."},
		{ID: "finance", Label: "Finance", Help: "Invoices and payment proofs kept as records. Not a Books post."},
		{ID: "people", Label: "People", Help: "Signed IP assignment or contractor paper."},
		{ID: "product", Label: "Product", Help: "Decks, specs, screenshots of the product."},
		{ID: "evidence", Label: "Evidence", Help: "Field photos, quotes, named desks."},
		{ID: "other", Label: "Other", Help: "Does not fit. Use sparingly."},
	}
}

func LookupFolder(id string, company bool) (Folder, bool) {
	id = strings.TrimSpace(strings.ToLower(id))
	for _, f := range Folders() {
		if f.ID != id {
			continue
		}
		if f.Company && !company {
			return Folder{}, false
		}
		return f, true
	}
	return Folder{}, false
}

func Handle(ctx context.Context, c llm.Completer, req contract.Request) (contract.Response, error) {
	data := contract.ContextObject(req.ContextData)
	images := departments.TakeImages(data)
	system := overlay + `

structured_data MUST include:
- project_slug: one id from context_data.projects[].slug, or "themobileprof"
- folder: one catalog id allowed for that slug
- filename: keep the original name if given; do not invent a new one
- unsure: true if the file, caption, or project is ambiguous
- save_filing: true only when unsure is false and both slug and folder are catalog ids
- cannot_file: true
- cannot_send: true
- rationale: one short sentence for the human
output_text: the path as "slug / folder / filename". Say if it matched the catalog (files itself) or stays proposed. Do not dump the file.`

	user := departments.UserPrompt(req.TaskDescription, data)
	if len(images) > 0 {
		user = "A file is attached. Read it. Caption is extra context, not a substitute.\n\n" + user
	}
	resp, err := departments.Run(ctx, c, system, user, images...)
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	resp.StructuredData["task_type"] = "files"
	resp.StructuredData["cannot_file"] = true
	resp.StructuredData["cannot_send"] = true
	if name := strVal(data["filename"]); name != "" {
		if strVal(resp.StructuredData["filename"]) == "" {
			resp.StructuredData["filename"] = name
		}
	}
	ApplyCatalog(resp.StructuredData, allowedSlugs(data))
	return resp, err
}

// ApplyCatalog drops invented slugs/folders. Mutates sd.
func ApplyCatalog(sd map[string]any, slugs map[string]bool) {
	if sd == nil {
		return
	}
	slug := strings.ToLower(strings.TrimSpace(strVal(sd["project_slug"])))
	if slug == "" {
		slug = strings.ToLower(strings.TrimSpace(strVal(sd["slug"])))
	}
	company := slug == CompanySlug
	okSlug := company || slugs[slug]
	folderID := strings.ToLower(strings.TrimSpace(strVal(sd["folder"])))
	_, okFolder := LookupFolder(folderID, company)
	unsure := boolVal(sd["unsure"]) || !okSlug || !okFolder
	sd["project_slug"] = slug
	sd["folder"] = folderID
	sd["unsure"] = unsure
	sd["cannot_file"] = true
	sd["cannot_send"] = true
	if unsure {
		sd["save_filing"] = false
		return
	}
	sd["save_filing"] = true
}

func Path(slug, folder, filename string) string {
	slug = strings.TrimSpace(slug)
	folder = strings.TrimSpace(folder)
	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = "document"
	}
	if slug == "" {
		slug = CompanySlug
	}
	return slug + "/" + folder + "/" + filename
}

func allowedSlugs(data map[string]any) map[string]bool {
	out := map[string]bool{CompanySlug: true}
	raw, ok := data["projects"].([]any)
	if !ok {
		return out
	}
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if s := strings.ToLower(strings.TrimSpace(strVal(m["slug"]))); s != "" {
			out[s] = true
		}
	}
	return out
}

func strVal(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func boolVal(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "true" || s == "1" || s == "yes"
	default:
		return false
	}
}
