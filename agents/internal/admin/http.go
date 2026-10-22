package admin

import (
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments/hr"
	"github.com/samuel/ai-workers/agents/internal/departments/legal"
	"github.com/samuel/ai-workers/agents/internal/departments/seo"
	"github.com/samuel/ai-workers/agents/internal/journeys"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

//go:embed templates/*.html static/*
var embedded embed.FS

type Server struct {
	store        *Store
	internalTok  string
	cookieSecure bool
	pages        map[string]*template.Template
	static       http.Handler
	place        PlaceFunc
	legal        PlaceFunc
	hr           PlaceFunc
	seo          PlaceFunc
	workers      map[string]PlaceFunc
	guide        llm.Completer
}

// PlaceFunc runs product-dev placement. It must only return a proposal; the desk commits the stamp.
type PlaceFunc func(ctx context.Context, req contract.Request) (contract.Response, error)

type pageData struct {
	Title          string
	Nav            string
	User           *User
	Flash          string
	Error          string
	Users          []User
	Editing        *User
	Adding         bool
	Amend          bool
	OwnerCount     int
	Allowlist      []string
	UserCount      int
	SettingCount   int
	Settings       []kv
	SettingGroups  []SettingGroupView
	Docs           []DocMeta
	Doc            *DocPage
	Projects       []Project
	Project        *Project
	Stages         []stageSpec
	Seats          []string
	Roles          []string
	Roster         []User
	Journeys       []journeys.Journey
	CanPlace       bool
	CanLegal       bool
	Drafts         []LegalDraft
	LegalTemplates []legal.Spec
	HrRoles        []HrRole
	HrApps         []HrApplication
	HrTemplates    []hr.Spec
	CanHR          bool
	NavDesks       []Desk
	DeskCatalog    []Desk
	CanPeople      bool
	CanOffice      bool
	Desk           *Desk
	Jobs           []DeskJob
	Job            *DeskJob
	Messages       []JobMessage
	CanAsk         bool
	Mandates       []CommunityMandate
	Mandate        *CommunityMandate
	WorkItems      []WorkItem
	DeliveryBets   []DeliveryBet
	SeoBriefs      []SeoBrief
	SeoKinds       []seo.Spec
	CanSEO         bool
	GuideAsks      []GuideAsk
	Events         []ProjectEvent
	EventKinds     []stageSpec
	Filings        []FileFiling
	Grants         []Grant
	Contacts       []CRMContact
	Contact        *CRMContact
	CRMEvents      []CRMEvent
	Rollup         BoardRollup
	Tab            string
	RunSteps       []RunStep
	RunLeft        int
}

type kv struct {
	Key   string
	Value string
}

func New(store *Store, internalToken string) (*Server, error) {
	if err := mustPlaybookFiles(); err != nil {
		return nil, err
	}
	if err := mustGuideFiles(); err != nil {
		return nil, err
	}
	funcMap := template.FuncMap{
		"has":          hasStoredCap,
		"canRemove":    canRemove,
		"stageLabel":   stageLabel,
		"gateLabel":    gateLabel,
		"assigned":     assignedDesk,
		"statusCount":  statusCount,
		"deskNavOn":    deskNavOn,
		"officeMenuOn": officeMenuOn,
		"guideReads":   guideReads,
	}
	pages := map[string]*template.Template{}
	for _, name := range []string{"login", "home", "users", "settings", "workflows", "docs", "docs_page", "projects", "legal", "hr", "desks", "community", "delivery", "seo", "guide", "files", "grants", "crm"} {
		t, err := template.New(name).Funcs(funcMap).ParseFS(embedded, "templates/layout.html", "templates/"+name+".html")
		if err != nil {
			return nil, err
		}
		pages[name] = t
	}
	sub, err := fs.Sub(embedded, "static")
	if err != nil {
		return nil, err
	}
	secure := os.Getenv("ADMIN_COOKIE_SECURE")
	cookieSecure := secure != "false"
	return &Server{
		store:        store,
		internalTok:  strings.TrimSpace(internalToken),
		cookieSecure: cookieSecure,
		pages:        pages,
		static:       http.FileServer(http.FS(sub)),
	}, nil
}

func (s *Server) WithPlacer(fn PlaceFunc) {
	if s != nil {
		s.place = fn
		s.WithWorker("product-dev", fn)
	}
}

// Register mounts /admin* (cookie session) and /internal/v1* (X-Internal-Token). Caddy must not publish /internal.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/static/", s.serveStatic)
	mux.HandleFunc("GET /admin/login", s.loginGET)
	mux.HandleFunc("POST /admin/login", s.loginPOST)
	mux.HandleFunc("POST /admin/logout", s.logout)
	mux.HandleFunc("GET /admin", s.home)
	mux.HandleFunc("GET /admin/", s.home)
	mux.HandleFunc("GET /admin/projects", s.projectsGET)
	mux.HandleFunc("GET /admin/projects/new", s.projectNewGET)
	mux.HandleFunc("GET /admin/projects/{id}", s.projectEditGET)
	mux.HandleFunc("POST /admin/projects", s.projectsPOST)
	mux.HandleFunc("POST /admin/projects/{id}", s.projectSave)
	mux.HandleFunc("POST /admin/projects/{id}/memory", s.projectMemoryPOST)
	mux.HandleFunc("POST /admin/projects/{id}/delete", s.projectDelete)
	mux.HandleFunc("POST /admin/projects/{id}/members", s.projectMemberAdd)
	mux.HandleFunc("POST /admin/projects/{id}/members/{uid}/delete", s.projectMemberRemove)
	mux.HandleFunc("POST /admin/projects/{id}/place", s.projectPlace)
	mux.HandleFunc("POST /admin/projects/{id}/proposal/accept", s.projectProposalAccept)
	mux.HandleFunc("POST /admin/projects/{id}/proposal/reject", s.projectProposalReject)
	mux.HandleFunc("POST /admin/projects/{id}/proposal/amend", s.projectProposalAmend)
	mux.HandleFunc("POST /admin/projects/{id}/legal", s.projectLegal)
	mux.HandleFunc("GET /admin/legal", s.legalGET)
	mux.HandleFunc("POST /admin/legal/{id}/approve", s.legalStamp("approved"))
	mux.HandleFunc("POST /admin/legal/{id}/reject", s.legalStamp("rejected"))
	mux.HandleFunc("GET /admin/files", s.filesGET)
	mux.HandleFunc("POST /admin/files/{id}/accept", s.filesStamp("accepted"))
	mux.HandleFunc("POST /admin/files/{id}/reject", s.filesStamp("rejected"))
	mux.HandleFunc("GET /admin/grants", s.grantsGET)
	mux.HandleFunc("POST /admin/grants", s.grantsPOST)
	mux.HandleFunc("POST /admin/grants/{id}/done", s.grantStamp("done"))
	mux.HandleFunc("POST /admin/grants/{id}/drop", s.grantStamp("dropped"))
	mux.HandleFunc("GET /admin/crm", s.crmGET)
	mux.HandleFunc("POST /admin/crm", s.crmPOST)
	mux.HandleFunc("GET /admin/crm/{id}", s.crmPersonGET)
	mux.HandleFunc("POST /admin/crm/{id}", s.crmPersonPOST)
	mux.HandleFunc("POST /admin/crm/{id}/log", s.crmLogPOST)
	mux.HandleFunc("POST /admin/crm/{id}/won", s.crmStamp("won"))
	mux.HandleFunc("POST /admin/crm/{id}/lost", s.crmStamp("lost"))
	mux.HandleFunc("POST /admin/crm/{id}/nurture", s.crmStamp("nurture"))
	mux.HandleFunc("GET /admin/seo", s.seoGET)
	mux.HandleFunc("POST /admin/seo/ask", s.seoAskPOST)
	mux.HandleFunc("POST /admin/seo/{id}/approve", s.seoStamp("approved"))
	mux.HandleFunc("POST /admin/seo/{id}/reject", s.seoStamp("rejected"))
	mux.HandleFunc("POST /admin/projects/{id}/seo", s.projectSEO)
	mux.HandleFunc("GET /admin/hr", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/delivery", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /admin/delivery", s.deliveryGET)
	mux.HandleFunc("POST /admin/delivery/items", s.deliveryItemPOST)
	mux.HandleFunc("POST /admin/delivery/items/{id}/done", s.deliveryItemStamp("done"))
	mux.HandleFunc("POST /admin/delivery/items/{id}/skip", s.deliveryItemStamp("skipped"))
	mux.HandleFunc("POST /admin/delivery/items/{id}/open", s.deliveryItemStamp("open"))
	mux.HandleFunc("POST /admin/delivery/ask", s.deliveryAskPOST)
	mux.HandleFunc("POST /admin/delivery/pace", s.deliveryPacePOST)
	mux.HandleFunc("POST /admin/projects/{id}/work", s.projectWorkPOST)
	mux.HandleFunc("POST /admin/projects/{id}/events", s.projectEventPOST)
	mux.HandleFunc("POST /admin/projects/{id}/events/{eid}/delivered", s.projectEventStamp("delivered"))
	mux.HandleFunc("POST /admin/projects/{id}/events/{eid}/skip", s.projectEventStamp("skipped"))
	mux.HandleFunc("POST /admin/projects/{id}/events/{eid}/task", s.projectEventTaskPOST)
	mux.HandleFunc("GET /admin/desks", s.desksIndex)
	mux.HandleFunc("GET /admin/desks/{slug}", s.deskGET)
	mux.HandleFunc("POST /admin/desks/{slug}/ask", s.deskAskPOST)
	mux.HandleFunc("POST /admin/desks/{slug}/jobs/{id}/chat", s.deskChatPOST)
	mux.HandleFunc("POST /admin/desks/{slug}/jobs/{id}/accept", s.deskStamp("accepted"))
	mux.HandleFunc("POST /admin/desks/{slug}/jobs/{id}/reject", s.deskStamp("rejected"))
	mux.HandleFunc("GET /admin/community", s.communityGET)
	mux.HandleFunc("POST /admin/community", s.communityPOST)
	mux.HandleFunc("POST /admin/community/{id}/delete", s.communityDelete)
	mux.HandleFunc("GET /admin/users", s.usersGET)
	mux.HandleFunc("GET /admin/users/new", s.userNewGET)
	mux.HandleFunc("GET /admin/users/{id}", s.userEditGET)
	mux.HandleFunc("POST /admin/users", s.usersPOST)
	mux.HandleFunc("POST /admin/users/{id}", s.userSave)
	mux.HandleFunc("POST /admin/users/{id}/delete", s.userDelete)
	mux.HandleFunc("GET /admin/settings", s.settingsGET)
	mux.HandleFunc("GET /admin/settings/edit", s.settingsEditGET)
	mux.HandleFunc("POST /admin/settings", s.settingsPOST)
	mux.HandleFunc("GET /admin/integrations", http.NotFound)
	mux.HandleFunc("GET /admin/workflows", s.workflowsGET)
	mux.HandleFunc("GET /admin/docs", s.docsIndex)
	mux.HandleFunc("GET /admin/docs/{slug}", s.docsPage)
	mux.HandleFunc("GET /admin/guide", s.guideGET)
	mux.HandleFunc("GET /admin/guide/asked.md", s.guideAskedMarkdown)
	mux.HandleFunc("POST /admin/guide/ask", s.guideAskPOST)
	mux.HandleFunc("GET /internal/v1/community/mandate", s.internalCommunityMandate)
	mux.HandleFunc("GET /internal/v1/community/mandates", s.internalCommunityMandates)
	mux.HandleFunc("GET /internal/v1/whatsapp-accounts", s.internalAllowlist)
	mux.HandleFunc("GET /internal/v1/staff", s.internalStaffGET)
	mux.HandleFunc("GET /internal/v1/mail/watchers", s.internalMailWatchersGET)
	mux.HandleFunc("GET /internal/v1/crm/contacts", s.internalCRMContactsGET)
	mux.HandleFunc("GET /internal/v1/settings", s.internalSettings)
	mux.HandleFunc("GET /internal/v1/projects", s.internalProjects)
	mux.HandleFunc("GET /internal/v1/journeys", s.internalJourneys)
	mux.HandleFunc("POST /internal/v1/projects/{id}/proposal", s.internalProposal)
	mux.HandleFunc("GET /internal/v1/legal-drafts", s.internalLegalDraftsGET)
	mux.HandleFunc("POST /internal/v1/legal-drafts", s.internalLegalDraftsPOST)
	mux.HandleFunc("GET /internal/v1/seo-briefs", s.internalSeoBriefsGET)
	mux.HandleFunc("POST /internal/v1/seo-briefs", s.internalSeoBriefsPOST)
	mux.HandleFunc("GET /internal/v1/hr/roles", s.internalHrRoles)
	mux.HandleFunc("POST /internal/v1/hr/applications", s.internalHrApplicationsPOST)
	mux.HandleFunc("GET /internal/v1/delivery/context", s.internalDeliveryContext)
	mux.HandleFunc("GET /internal/v1/delivery/digest", s.internalDeliveryDigest)
	mux.HandleFunc("POST /internal/v1/delivery/checkin", s.internalDeliveryCheckin)
	mux.HandleFunc("POST /internal/v1/jobs", s.internalJobsPOST)
	mux.HandleFunc("GET /internal/v1/jobs/due", s.internalJobsDue)
	mux.HandleFunc("POST /internal/v1/jobs/{id}/apply", s.internalJobApply)
	mux.HandleFunc("POST /internal/v1/jobs/stop-auto", s.internalJobsStopAuto)
	mux.HandleFunc("GET /internal/v1/file-filings", s.internalFileFilingsGET)
	mux.HandleFunc("POST /internal/v1/file-filings", s.internalFileFilingsPOST)
	mux.HandleFunc("POST /internal/v1/file-filings/{id}/placed", s.internalFilePlacedPOST)
}

func RegisterPublic(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", publicHome)
	sub, err := fs.Sub(embedded, "static")
	if err != nil {
		log.Printf("public static: %v", err)
		return
	}
	mux.Handle("GET /site/", http.StripPrefix("/site/", http.FileServer(http.FS(sub))))
}

func publicHome(w http.ResponseWriter, r *http.Request) {
	b, err := embedded.ReadFile("templates/landing.html")
	if err != nil {
		http.Error(w, "landing missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(b)
}

func RegisterUnavailable(mux *http.ServeMux) {
	h := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "admin desk is not configured", http.StatusServiceUnavailable)
	}
	mux.HandleFunc("GET /admin", h)
	mux.HandleFunc("GET /admin/{rest...}", h)
	mux.HandleFunc("POST /admin/{rest...}", h)
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	r = r.Clone(r.Context())
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/admin/static")
	s.static.ServeHTTP(w, r)
}

func (s *Server) loginGET(w http.ResponseWriter, r *http.Request) {
	if u := s.currentUser(r); u != nil {
		http.Redirect(w, r, afterLoginPath(*u), http.StatusSeeOther)
		return
	}
	s.render(w, "login", pageData{Title: "Sign in", Error: r.URL.Query().Get("e")})
}

func (s *Server) loginPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/login?e=bad+form", http.StatusSeeOther)
		return
	}
	u, err := s.store.Authenticate(r.Context(), r.FormValue("login"), r.FormValue("password"))
	if err != nil || !HasCap(u, "web_admin") {
		http.Redirect(w, r, "/admin/login?e=invalid+login", http.StatusSeeOther)
		return
	}
	tok, err := s.store.CreateSession(r.Context(), u.ID, 7*24*time.Hour)
	if err != nil {
		http.Redirect(w, r, "/admin/login?e=session+failed", http.StatusSeeOther)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "desk_session",
		Value:    tok,
		Path:     "/admin",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})
	http.Redirect(w, r, afterLoginPath(u), http.StatusSeeOther)
}

func (s *Server) workflowsGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	s.render(w, "workflows", pageData{Title: "n8n", Nav: "flows", User: u})
}

func (s *Server) docsIndex(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	s.render(w, "docs", pageData{Title: "Docs", Nav: "docs", User: u, Docs: allDocs()})
}

func (s *Server) docsPage(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	page, ok := lookupDoc(r.PathValue("slug"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.render(w, "docs_page", pageData{Title: page.Title, Nav: "docs", User: u, Docs: allDocs(), Doc: &page})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	if c, err := r.Cookie("desk_session"); err == nil {
		s.store.DeleteSession(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "desk_session", Value: "", Path: "/admin", MaxAge: -1})
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (s *Server) usersGET(w http.ResponseWriter, r *http.Request) {
	u := s.requirePeople(w, r)
	if u == nil {
		return
	}
	users, err := s.store.ListUsers(r.Context())
	owners, _ := s.store.CountOwners(r.Context())
	data := pageData{Title: "People", Nav: "users", User: u, Users: users, OwnerCount: owners, Roles: deskRoles()}
	if err != nil {
		data.Error = err.Error()
	}
	s.render(w, "users", data)
}

func (s *Server) userNewGET(w http.ResponseWriter, r *http.Request) {
	u := s.requirePeople(w, r)
	if u == nil {
		return
	}
	s.render(w, "users", pageData{Title: "Add person", Nav: "users", User: u, Adding: true, Roles: deskRoles()})
}

func (s *Server) userEditGET(w http.ResponseWriter, r *http.Request) {
	u := s.requirePeople(w, r)
	if u == nil {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	person, err := s.store.GetUser(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	person.HoursPerWeek = s.store.UserPace(r.Context(), person.ID)
	s.render(w, "users", pageData{Title: "Amend " + person.Name, Nav: "users", User: u, Editing: &person, OwnerCount: s.owners(r.Context()), Roles: deskRoles()})
}

func (s *Server) usersPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requirePeople(w, r)
	if u == nil {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	caps := withLoginCap(r.Form["cap"], r.Form["desk"])
	created, err := s.store.CreateUser(r.Context(), r.FormValue("name"), r.FormValue("phone"), r.FormValue("email"), r.FormValue("password"), r.FormValue("role"), caps, r.Form["desk"])
	if err != nil {
		s.render(w, "users", pageData{Title: "Add person", Nav: "users", User: u, Adding: true, Roles: deskRoles(), Error: err.Error()})
		return
	}
	if err := s.store.SetUserTelegram(r.Context(), created.ID, r.FormValue("telegram_id")); err != nil {
		s.render(w, "users", pageData{Title: "Add person", Nav: "users", User: u, Adding: true, Roles: deskRoles(), Error: err.Error()})
		return
	}
	_ = s.store.SetUserPace(r.Context(), created.ID, parsePace(r.FormValue("hours_per_week")))
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) userSave(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requirePeople(w, r)
	if u == nil {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	active := r.FormValue("active") == "1"
	person, err := s.store.GetUser(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	owners, _ := s.store.CountOwners(r.Context())
	if err := lastOwnerLocked(person, r.FormValue("role"), active, owners); err != nil {
		s.render(w, "users", pageData{Title: "Amend person", Nav: "users", User: u, Editing: &person, OwnerCount: owners, Roles: deskRoles(), Error: err.Error()})
		return
	}
	caps := withLoginCap(r.Form["cap"], r.Form["desk"])
	if err := s.store.UpdateUser(r.Context(), id, r.FormValue("name"), r.FormValue("phone"), r.FormValue("email"), r.FormValue("password"), r.FormValue("role"), active, caps, r.Form["desk"]); err != nil {
		s.render(w, "users", pageData{Title: "Amend person", Nav: "users", User: u, Editing: &person, OwnerCount: owners, Roles: deskRoles(), Error: err.Error()})
		return
	}
	if err := s.store.SetUserTelegram(r.Context(), id, r.FormValue("telegram_id")); err != nil {
		s.render(w, "users", pageData{Title: "Amend person", Nav: "users", User: u, Editing: &person, OwnerCount: owners, Roles: deskRoles(), Error: err.Error()})
		return
	}
	_ = s.store.SetUserPace(r.Context(), id, parsePace(r.FormValue("hours_per_week")))
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) userDelete(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requirePeople(w, r)
	if u == nil {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.store.DeleteUser(r.Context(), u, id); err != nil {
		users, _ := s.store.ListUsers(r.Context())
		owners, _ := s.store.CountOwners(r.Context())
		s.render(w, "users", pageData{Title: "People", Nav: "users", User: u, Users: users, OwnerCount: owners, Roles: deskRoles(), Error: err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func settingsTab(r *http.Request, groups []SettingGroupView) string {
	tab := strings.TrimSpace(r.URL.Query().Get("tab"))
	if tab == "" {
		tab = strings.TrimSpace(r.FormValue("group"))
	}
	for _, g := range groups {
		if g.ID == tab {
			return tab
		}
	}
	if len(groups) > 0 {
		return groups[0].ID
	}
	return ""
}

func (s *Server) settingsGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	raw, err := s.store.Settings(r.Context())
	groups := settingGroupsFrom(raw)
	data := pageData{Title: "Defaults", Nav: "settings", User: u, SettingGroups: groups, Tab: settingsTab(r, groups)}
	if err != nil {
		data.Error = err.Error()
	}
	s.render(w, "settings", data)
}

func (s *Server) settingsEditGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	raw, err := s.store.Settings(r.Context())
	groups := settingGroupsFrom(raw)
	data := pageData{Title: "Amend defaults", Nav: "settings", User: u, SettingGroups: groups, Amend: true, Tab: settingsTab(r, groups)}
	if err != nil {
		data.Error = err.Error()
	}
	s.render(w, "settings", data)
}

func (s *Server) settingsPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	raw, _ := s.store.Settings(r.Context())
	allowed := catalogSettingKeys()
	for k := range raw {
		allowed[k] = true
	}
	for k, vs := range r.Form {
		if k == "" || len(vs) == 0 || !allowed[k] {
			continue
		}
		if err := s.store.SetSetting(r.Context(), k, vs[0]); err != nil {
			groups := settingGroupsFrom(raw)
			s.render(w, "settings", pageData{Title: "Amend defaults", Nav: "settings", User: u, SettingGroups: groups, Amend: true, Tab: settingsTab(r, groups), Error: err.Error()})
			return
		}
	}
	loc := "/admin/settings"
	if group := strings.TrimSpace(r.FormValue("group")); group != "" {
		loc += "?tab=" + url.QueryEscape(group)
	}
	http.Redirect(w, r, loc, http.StatusSeeOther)
}

// Leftover: customer WhatsApp is not gated. Staff use GET /internal/v1/staff.
func (s *Server) internalAllowlist(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	nums, err := s.store.WhatsAppAccounts(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"numbers": nums})
}

func (s *Server) internalSettings(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	st, err := s.store.Settings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, st)
}

func (s *Server) internalOK(r *http.Request) bool {
	if s == nil {
		return false
	}
	return InternalTokenOK(InternalRequestToken(r), s.internalTok)
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) *User {
	u := s.currentUser(r)
	if u == nil || !HasCap(*u, "web_admin") {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return nil
	}
	return u
}

func (s *Server) requireOffice(w http.ResponseWriter, r *http.Request) *User {
	u := s.requireAdmin(w, r)
	if u == nil {
		return nil
	}
	if !canOffice(u) {
		http.Redirect(w, r, afterLoginPath(*u), http.StatusSeeOther)
		return nil
	}
	return u
}

func (s *Server) currentUser(r *http.Request) *User {
	c, err := r.Cookie("desk_session")
	if err != nil || c.Value == "" {
		return nil
	}
	sess, err := s.store.Session(r.Context(), c.Value)
	if err != nil || !sess.User.Active {
		return nil
	}
	u := sess.User
	return &u
}

func withQuery(path, key, val string) string {
	if path == "" || key == "" {
		return path
	}
	u, err := url.Parse(path)
	if err != nil {
		return path
	}
	q := u.Query()
	q.Set(key, val)
	u.RawQuery = q.Encode()
	return u.String()
}

func statusCount(items any, status string) int {
	n := 0
	switch xs := items.(type) {
	case []LegalDraft:
		for _, d := range xs {
			if d.Status == status {
				n++
			}
		}
	case []SeoBrief:
		for _, d := range xs {
			if d.Status == status {
				n++
			}
		}
	case []Integration:
		for _, it := range xs {
			if it.Status == status {
				n++
			}
		}
	case []FileFiling:
		for _, d := range xs {
			if status == "accepted" {
				if d.Status == "accepted" || d.Status == "filed" {
					n++
				}
			} else if d.Status == status {
				n++
			}
		}
	case []DeskJob:
		for _, j := range xs {
			if status == "proposed" {
				if j.Proposed() {
					n++
				}
			} else if j.Status == status {
				n++
			}
		}
	case []WorkItem:
		for _, it := range xs {
			if status == "open" {
				if it.Open() {
					n++
				}
			} else if it.Status == status {
				n++
			}
		}
	case []Grant:
		for _, g := range xs {
			if g.Status == status || (status == "open" && g.Open()) {
				n++
			}
		}
	case []CRMContact:
		for _, c := range xs {
			if status == "open" {
				if c.Open() {
					n++
				}
			} else if status == "closed" {
				if !c.Open() {
					n++
				}
			} else if c.Stage == status {
				n++
			}
		}
	case []GuideAsk:
		for _, a := range xs {
			if status == "unsure" && a.Unsure {
				n++
			}
		}
	}
	return n
}

func (s *Server) render(w http.ResponseWriter, name string, data pageData) {
	t := s.pages[name]
	data.NavDesks = visibleDesks(data.User)
	data.CanPeople = isOwner(data.User)
	data.CanOffice = canOffice(data.User)
	data.DeskCatalog = AllDesks()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := t.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("admin template %s: %v", name, err)
	}
}

func toKV(m map[string]string) []kv {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]kv, 0, len(keys))
	for _, k := range keys {
		out = append(out, kv{Key: k, Value: m[k]})
	}
	return out
}

func canRemove(actor *User, target any, owners int) bool {
	var t User
	switch v := target.(type) {
	case User:
		t = v
	case *User:
		if v == nil {
			return false
		}
		t = *v
	default:
		return false
	}
	return removalBlocked(actor, t, owners) == nil
}

func (s *Server) owners(ctx context.Context) int {
	n, _ := s.store.CountOwners(ctx)
	return n
}

func hasStoredCap(u any, cap string) bool {
	var user User
	switch t := u.(type) {
	case User:
		user = t
	case *User:
		if t == nil {
			return false
		}
		user = *t
	default:
		return false
	}
	for _, c := range user.Caps {
		if c == cap {
			return true
		}
	}
	return false
}

func assignedDesk(u any, slug string) bool {
	var user User
	switch t := u.(type) {
	case User:
		user = t
	case *User:
		if t == nil {
			return false
		}
		user = *t
	default:
		return false
	}
	for _, d := range user.Desks {
		if d == slug {
			return true
		}
	}
	return false
}

func withLoginCap(caps, desks []string) []string {
	if len(desks) == 0 {
		return caps
	}
	for _, c := range caps {
		if c == "web_admin" {
			return caps
		}
	}
	return append(append([]string{}, caps...), "web_admin")
}

func sameOrigin(w http.ResponseWriter, r *http.Request) bool {
	host := r.Host
	origin := r.Header.Get("Origin")
	if origin != "" {
		u, err := url.Parse(origin)
		if err != nil || !strings.EqualFold(u.Host, host) {
			http.Error(w, "bad origin", http.StatusForbidden)
			return false
		}
		return true
	}
	ref := r.Header.Get("Referer")
	if ref == "" {
		http.Error(w, "bad origin", http.StatusForbidden)
		return false
	}
	u, err := url.Parse(ref)
	if err != nil || !strings.EqualFold(u.Host, host) {
		http.Error(w, "bad origin", http.StatusForbidden)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
