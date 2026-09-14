package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/samuel/ai-workers/agents/internal/journeys"
)

// Project is one incubated bet under TheMobileProf Technologies.
// It is not a second CAC company. Books, Legal, and parent workers stay on TMP.
type Project struct {
	ID        int64
	Name      string
	Slug      string
	OneLiner  string
	Stage     string
	URL       string
	Notes     string
	Journey   string
	Gate      string
	Proposal  *Proposal
	Members   []ProjectMember
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Proposal is an uncommitted placement. Only a human Accept/Amend writes the stamp.
type Proposal struct {
	Journey    string `json:"suggested_journey"`
	Gate       string `json:"suggested_gate"`
	Mission    string `json:"mission"`
	Rationale  string `json:"rationale"`
	Confidence int    `json:"confidence"`
	Source     string `json:"source"`
}

type ProjectMember struct {
	UserID int64
	Name   string
	Phone  string
	Email  string
	Role   string
	Seat   string
}

type stageSpec struct {
	ID    string
	Label string
	Help  string
}

func projectStages() []stageSpec {
	return []stageSpec{
		{ID: "idea", Label: "Idea", Help: "One sentence: who pays, for what pain. No Books org."},
		{ID: "testing", Label: "Testing", Help: "Falsifiable hypotheses and field missions. Evidence, not opinions."},
		{ID: "selling", Label: "Selling", Help: "A priced offer. First invoice still goes out as TheMobileProf Technologies."},
		{ID: "fundable", Label: "Fundable", Help: "Paying customers. May get a cofounder or a manager; still one legal entity."},
		{ID: "parked", Label: "Parked", Help: "Not this month. Keep the record."},
		{ID: "killed", Label: "Killed", Help: "Admin decision. Do not resurrect as a fake company."},
	}
}

func projectSeats() []string {
	return []string{"cofounder", "assistant"}
}

func deskRoles() []string {
	return []string{"owner", "bdm", "cofounder", "assistant", "accounts", "operator", "viewer"}
}

func stageByID(id string) (stageSpec, bool) {
	for _, s := range projectStages() {
		if s.ID == id {
			return s, true
		}
	}
	return stageSpec{}, false
}

func validStage(id string) bool {
	_, ok := stageByID(id)
	return ok
}

func validSeat(seat string) bool {
	for _, s := range projectSeats() {
		if s == seat {
			return true
		}
	}
	return false
}

func normalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastHyphen := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastHyphen = false
		default:
			if b.Len() > 0 && !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 64 {
		out = strings.Trim(out[:64], "-")
	}
	return out
}

func (s *Store) migrateProjects(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS projects (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  one_liner TEXT NOT NULL DEFAULT '',
  stage TEXT NOT NULL DEFAULT 'idea',
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS project_members (
  project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  seat TEXT NOT NULL,
  PRIMARY KEY (project_id, user_id)
);
CREATE INDEX IF NOT EXISTS project_members_user_idx ON project_members (user_id);
ALTER TABLE projects ADD COLUMN IF NOT EXISTS url TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS journey TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS gate TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN IF NOT EXISTS proposal JSONB;
`)
	return err
}

func defaultProjects() []Project {
	out := []Project{
		{
			Name:     "MomLaunchpad",
			Slug:     "momlaunchpad",
			URL:      "https://momlaunchpad.com",
			OneLiner: "Expectant parents subscribe for pregnancy chat, community, and a week-by-week calendar on their phone.",
			Notes:    "Live site + Android early access. Plans live in the MomLaunchpad admin. Fundable when subscriptions actually pay.",
		},
		{
			Name:     "TheMobileProf Academy",
			Slug:     "academy",
			URL:      "https://lms.themobileprof.com",
			OneLiner: "Learners pay for micro, mini, and professional tech courses they can finish on a phone.",
			Notes:    "The going training property. Bump to fundable when current paid enrollments are the proof, not the catalog.",
		},
		{
			Name:     "Finchest AI",
			Slug:     "finchest",
			URL:      "https://finchest.themobileprof.com",
			OneLiner: "Nigerian investors, invited by an advisor, fund a private Anchor / Buffer / Spear portfolio.",
			Notes:    "Invitation-only. Not a public checkout. Fundable when real money sits in the book.",
		},
		{
			Name:     "HomeGauge",
			Slug:     "homegauge",
			URL:      "https://themobileprof.com/mortgage",
			OneLiner: "Salaried Nigerian homebuyers get mortgage eligibility and paperwork help — HomeGauge is not the lender.",
			Notes:    "Live eligibility + calculator. Who pays (buyer, advisor, or lender) is still the hypothesis.",
		},
		{
			Name:     "Mechazone",
			Slug:     "mechazone",
			URL:      "",
			OneLiner: "Independent workshops pay for a VIN-indexed diagnostic ledger on the OpenPort kit already on the bay.",
			Notes:    "Repo sites/mechazone. Shops request registration; no self-signup. Specimen for /validate until a public URL exists.",
		},
	}
	for i := range out {
		out[i].Journey, out[i].Gate = journeys.SeedPlacement(out[i].Slug)
		if g, ok := journeys.LookupGate(out[i].Journey, out[i].Gate); ok {
			out[i].Stage = g.Stage
		}
	}
	return out
}

func (s *Store) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, name, slug, one_liner, stage, url, notes, journey, gate, proposal, created_at, updated_at
FROM projects ORDER BY
  CASE stage
    WHEN 'selling' THEN 0
    WHEN 'testing' THEN 1
    WHEN 'idea' THEN 2
    WHEN 'fundable' THEN 3
    WHEN 'parked' THEN 4
    ELSE 5
  END, name
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Project
	ids := make([]int64, 0)
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
		ids = append(ids, p.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	byID, err := s.membersForProjects(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Members = byID[out[i].ID]
	}
	return out, nil
}

func (s *Store) GetProject(ctx context.Context, id int64) (Project, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, name, slug, one_liner, stage, url, notes, journey, gate, proposal, created_at, updated_at
FROM projects WHERE id = $1
`, id)
	p, err := scanProject(row)
	if err != nil {
		return Project{}, err
	}
	byID, err := s.membersForProjects(ctx, []int64{id})
	if err != nil {
		return Project{}, err
	}
	p.Members = byID[id]
	return p, nil
}

func (s *Store) GetProjectBySlug(ctx context.Context, slug string) (Project, error) {
	slug = normalizeSlug(slug)
	if slug == "" {
		return Project{}, errors.New("slug is required")
	}
	row := s.pool.QueryRow(ctx, `
SELECT id, name, slug, one_liner, stage, url, notes, journey, gate, proposal, created_at, updated_at
FROM projects WHERE slug = $1
`, slug)
	p, err := scanProject(row)
	if err != nil {
		return Project{}, err
	}
	byID, err := s.membersForProjects(ctx, []int64{p.ID})
	if err != nil {
		return Project{}, err
	}
	p.Members = byID[p.ID]
	return p, nil
}

type projectScanner interface {
	Scan(dest ...any) error
}

func scanProject(row projectScanner) (Project, error) {
	var p Project
	var raw []byte
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.OneLiner, &p.Stage, &p.URL, &p.Notes, &p.Journey, &p.Gate, &raw, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return Project{}, err
	}
	p.Proposal = decodeProposal(raw)
	return p, nil
}

func decodeProposal(raw []byte) *Proposal {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var p Proposal
	if err := json.Unmarshal(raw, &p); err != nil || (p.Journey == "" && p.Gate == "" && p.Mission == "") {
		return nil
	}
	return &p
}

func (s *Store) CreateProject(ctx context.Context, name, slug, oneLiner, stage, notes, url, journey, gate string) (Project, error) {
	p, err := cleanProject(name, slug, oneLiner, stage, notes, url, journey, gate)
	if err != nil {
		return Project{}, err
	}
	var id int64
	err = s.pool.QueryRow(ctx, `
INSERT INTO projects (name, slug, one_liner, stage, notes, url, journey, gate)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id
`, p.Name, p.Slug, p.OneLiner, p.Stage, p.Notes, p.URL, p.Journey, p.Gate).Scan(&id)
	if err != nil {
		return Project{}, friendlyDBErr(err)
	}
	return s.GetProject(ctx, id)
}

func (s *Store) UpdateProject(ctx context.Context, id int64, name, slug, oneLiner, stage, notes, url, journey, gate string) error {
	p, err := cleanProject(name, slug, oneLiner, stage, notes, url, journey, gate)
	if err != nil {
		return err
	}
	old, err := s.GetProject(ctx, id)
	if err != nil {
		return errors.New("project not found")
	}
	if old.Journey != p.Journey || old.Gate != p.Gate {
		_, err = s.pool.Exec(ctx, `
UPDATE projects
SET name=$1, slug=$2, one_liner=$3, stage=$4, notes=$5, url=$6, journey=$7, gate=$8, proposal=NULL, updated_at=now()
WHERE id=$9
`, p.Name, p.Slug, p.OneLiner, p.Stage, p.Notes, p.URL, p.Journey, p.Gate, id)
	} else {
		_, err = s.pool.Exec(ctx, `
UPDATE projects
SET name=$1, slug=$2, one_liner=$3, stage=$4, notes=$5, url=$6, journey=$7, gate=$8, updated_at=now()
WHERE id=$9
`, p.Name, p.Slug, p.OneLiner, p.Stage, p.Notes, p.URL, p.Journey, p.Gate, id)
	}
	if err != nil {
		return friendlyDBErr(err)
	}
	return nil
}

func (s *Store) DeleteProject(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("project not found")
	}
	return nil
}

func (s *Store) AddProjectMember(ctx context.Context, projectID, userID int64, seat string) error {
	seat = strings.TrimSpace(strings.ToLower(seat))
	if !validSeat(seat) {
		return errors.New("seat must be cofounder or assistant")
	}
	if userID <= 0 {
		return errors.New("pick a person")
	}
	if _, err := s.GetProject(ctx, projectID); err != nil {
		return errors.New("project not found")
	}
	if _, err := s.GetUser(ctx, userID); err != nil {
		return errors.New("person not found")
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO project_members (project_id, user_id, seat) VALUES ($1, $2, $3)
ON CONFLICT (project_id, user_id) DO UPDATE SET seat = EXCLUDED.seat
`, projectID, userID, seat)
	return friendlyDBErr(err)
}

func (s *Store) RemoveProjectMember(ctx context.Context, projectID, userID int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`, projectID, userID)
	return err
}

func (s *Store) membersForProjects(ctx context.Context, ids []int64) (map[int64][]ProjectMember, error) {
	out := map[int64][]ProjectMember{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT pm.project_id, u.id, u.name, u.phone, u.email, u.role, pm.seat
FROM project_members pm
JOIN users u ON u.id = pm.user_id
WHERE pm.project_id = ANY($1)
ORDER BY pm.seat, u.name
`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pid int64
		var m ProjectMember
		if err := rows.Scan(&pid, &m.UserID, &m.Name, &m.Phone, &m.Email, &m.Role, &m.Seat); err != nil {
			return nil, err
		}
		out[pid] = append(out[pid], m)
	}
	return out, rows.Err()
}

func cleanProject(name, slug, oneLiner, stage, notes, rawURL, journey, gate string) (Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Project{}, errors.New("name is required")
	}
	slug = normalizeSlug(slug)
	if slug == "" {
		slug = normalizeSlug(name)
	}
	if slug == "" {
		return Project{}, errors.New("slug is required")
	}
	oneLiner = strings.TrimSpace(oneLiner)
	if oneLiner == "" {
		return Project{}, errors.New("one-liner is required — who pays, for what pain")
	}
	journey = strings.TrimSpace(journey)
	gate = strings.TrimSpace(gate)
	if err := journeys.ValidPlacement(journey, gate); err != nil {
		return Project{}, err
	}
	stage = strings.TrimSpace(strings.ToLower(stage))
	if g, ok := journeys.LookupGate(journey, gate); ok && stage != "parked" && stage != "killed" {
		stage = g.Stage
	}
	if stage == "" {
		stage = "idea"
	}
	if !validStage(stage) {
		return Project{}, errors.New("unknown stage")
	}
	return Project{
		Name:     name,
		Slug:     slug,
		OneLiner: oneLiner,
		Stage:    stage,
		URL:      normalizeProjectURL(rawURL),
		Notes:    strings.TrimSpace(notes),
		Journey:  journey,
		Gate:     gate,
	}, nil
}

func normalizeProjectURL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "://") {
		return s
	}
	return "https://" + strings.TrimLeft(s, "/")
}

func (s *Store) seedProject(ctx context.Context, p Project) error {
	clean, err := cleanProject(p.Name, p.Slug, p.OneLiner, p.Stage, p.Notes, p.URL, p.Journey, p.Gate)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO projects (name, slug, one_liner, stage, notes, url, journey, gate)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (slug) DO UPDATE SET
  journey = CASE WHEN projects.journey = '' THEN EXCLUDED.journey ELSE projects.journey END,
  gate = CASE WHEN projects.gate = '' THEN EXCLUDED.gate ELSE projects.gate END
`, clean.Name, clean.Slug, clean.OneLiner, clean.Stage, clean.Notes, clean.URL, clean.Journey, clean.Gate)
	return err
}

func (s *Store) SetProposal(ctx context.Context, id int64, prop Proposal) error {
	if err := journeys.ValidPlacement(prop.Journey, prop.Gate); err != nil {
		return err
	}
	prop.Source = strings.TrimSpace(prop.Source)
	if prop.Source == "" {
		prop.Source = "place_on_journey"
	}
	raw, err := json.Marshal(prop)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
UPDATE projects SET proposal = $2::jsonb, updated_at = now() WHERE id = $1
`, id, raw)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("project not found")
	}
	return nil
}

func (s *Store) ClearProposal(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE projects SET proposal = NULL, updated_at = now() WHERE id = $1`, id)
	return err
}

func (s *Store) AcceptProposal(ctx context.Context, id int64) error {
	p, err := s.GetProject(ctx, id)
	if err != nil {
		return errors.New("project not found")
	}
	if p.Proposal == nil {
		return errors.New("no open proposal")
	}
	return s.CommitPlacement(ctx, id, p.Proposal.Journey, p.Proposal.Gate, p.Proposal.Mission)
}

func (s *Store) CommitPlacement(ctx context.Context, id int64, journey, gate, mission string) error {
	p, err := s.GetProject(ctx, id)
	if err != nil {
		return errors.New("project not found")
	}
	clean, err := cleanProject(p.Name, p.Slug, p.OneLiner, p.Stage, firstNonEmpty(mission, p.Notes), p.URL, journey, gate)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
UPDATE projects
SET journey=$1, gate=$2, stage=$3, notes=$4, proposal=NULL, updated_at=now()
WHERE id=$5
`, clean.Journey, clean.Gate, clean.Stage, clean.Notes, id)
	return err
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func proposalFromStructured(sd map[string]any) (Proposal, error) {
	p := Proposal{
		Journey:    strVal(sd["suggested_journey"]),
		Gate:       strVal(sd["suggested_gate"]),
		Mission:    strVal(sd["mission"]),
		Rationale:  strVal(sd["rationale"]),
		Confidence: intVal(sd["confidence"]),
		Source:     firstNonEmpty(strVal(sd["action"]), "place_on_journey"),
	}
	if p.Mission == "" {
		return Proposal{}, errors.New("proposal missing mission")
	}
	if err := journeys.ValidPlacement(p.Journey, p.Gate); err != nil {
		return Proposal{}, err
	}
	return p, nil
}

func strVal(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return ""
	}
}

func intVal(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	default:
		return 0
	}
}

func (p Project) StageLabel() string {
	if s, ok := stageByID(p.Stage); ok {
		return s.Label
	}
	return p.Stage
}

func (p Project) StageStamp() string {
	switch p.Stage {
	case "fundable", "selling":
		return "ok"
	case "killed":
		return "dark"
	case "parked":
		return "dark"
	default:
		return "brass"
	}
}

func (p Project) GateLabel() string {
	if g, ok := journeys.LookupGate(p.Journey, p.Gate); ok {
		return g.Label
	}
	return p.Gate
}

func (p Project) JourneyLabel() string {
	if j, ok := journeys.Lookup(p.Journey); ok {
		return j.Name
	}
	return p.Journey
}

func (p Proposal) GateLabel() string {
	if g, ok := journeys.LookupGate(p.Journey, p.Gate); ok {
		return g.Label
	}
	return p.Gate
}

func (p Proposal) JourneyLabel() string {
	if j, ok := journeys.Lookup(p.Journey); ok {
		return j.Name
	}
	return p.Journey
}

func (p Project) MemberNames() string {
	if len(p.Members) == 0 {
		return ""
	}
	parts := make([]string, 0, len(p.Members))
	for _, m := range p.Members {
		parts = append(parts, m.Name+" ("+m.Seat+")")
	}
	return strings.Join(parts, ", ")
}
