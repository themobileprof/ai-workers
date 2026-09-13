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
)

//go:embed templates/*.html static/*
var embedded embed.FS

type Server struct {
	store        *Store
	internalTok  string
	cookieSecure bool
	pages        map[string]*template.Template
	static       http.Handler
}

type pageData struct {
	Title        string
	Nav          string
	User         *User
	Flash        string
	Error        string
	Users        []User
	Editing      *User
	Adding       bool
	Amend        bool
	OwnerCount   int
	Allowlist    []string
	UserCount    int
	SettingCount int
	Settings     []kv
}

type kv struct {
	Key   string
	Value string
}

func New(store *Store, internalToken string) (*Server, error) {
	funcMap := template.FuncMap{
		"has":       hasStoredCap,
		"canRemove": canRemove,
	}
	pages := map[string]*template.Template{}
	for _, name := range []string{"login", "home", "users", "settings"} {
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

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/static/", s.serveStatic)
	mux.HandleFunc("GET /admin/login", s.loginGET)
	mux.HandleFunc("POST /admin/login", s.loginPOST)
	mux.HandleFunc("POST /admin/logout", s.logout)
	mux.HandleFunc("GET /admin", s.home)
	mux.HandleFunc("GET /admin/", s.home)
	mux.HandleFunc("GET /admin/users", s.usersGET)
	mux.HandleFunc("GET /admin/users/new", s.userNewGET)
	mux.HandleFunc("GET /admin/users/{id}", s.userEditGET)
	mux.HandleFunc("POST /admin/users", s.usersPOST)
	mux.HandleFunc("POST /admin/users/{id}", s.userSave)
	mux.HandleFunc("POST /admin/users/{id}/delete", s.userDelete)
	mux.HandleFunc("GET /admin/settings", s.settingsGET)
	mux.HandleFunc("GET /admin/settings/edit", s.settingsEditGET)
	mux.HandleFunc("POST /admin/settings", s.settingsPOST)
	mux.HandleFunc("GET /internal/v1/whatsapp-accounts", s.internalAllowlist)
	mux.HandleFunc("GET /internal/v1/settings", s.internalSettings)
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
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
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
	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
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

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	users, _ := s.store.ListUsers(r.Context())
	allow, _ := s.store.WhatsAppAccounts(r.Context())
	settings, _ := s.store.Settings(r.Context())
	s.render(w, "home", pageData{
		Title:        "Board",
		Nav:          "home",
		User:         u,
		Users:        users,
		Allowlist:    allow,
		UserCount:    len(users),
		SettingCount: len(settings),
	})
}

func (s *Server) usersGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	users, err := s.store.ListUsers(r.Context())
	owners, _ := s.store.CountOwners(r.Context())
	data := pageData{Title: "People", Nav: "users", User: u, Users: users, OwnerCount: owners}
	if err != nil {
		data.Error = err.Error()
	}
	s.render(w, "users", data)
}

func (s *Server) userNewGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	s.render(w, "users", pageData{Title: "Add person", Nav: "users", User: u, Adding: true})
}

func (s *Server) userEditGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
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
	s.render(w, "users", pageData{Title: "Amend " + person.Name, Nav: "users", User: u, Editing: &person, OwnerCount: s.owners(r.Context())})
}

func (s *Server) usersPOST(w http.ResponseWriter, r *http.Request) {
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
	_, err := s.store.CreateUser(r.Context(), r.FormValue("name"), r.FormValue("phone"), r.FormValue("email"), r.FormValue("password"), r.FormValue("role"), r.Form["cap"])
	if err != nil {
		s.render(w, "users", pageData{Title: "Add person", Nav: "users", User: u, Adding: true, Error: err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) userSave(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
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
		s.render(w, "users", pageData{Title: "Amend person", Nav: "users", User: u, Editing: &person, OwnerCount: owners, Error: err.Error()})
		return
	}
	if err := s.store.UpdateUser(r.Context(), id, r.FormValue("name"), r.FormValue("phone"), r.FormValue("email"), r.FormValue("password"), r.FormValue("role"), active, r.Form["cap"]); err != nil {
		s.render(w, "users", pageData{Title: "Amend person", Nav: "users", User: u, Editing: &person, OwnerCount: owners, Error: err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) userDelete(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
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
		s.render(w, "users", pageData{Title: "People", Nav: "users", User: u, Users: users, OwnerCount: owners, Error: err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) settingsGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	raw, err := s.store.Settings(r.Context())
	data := pageData{Title: "Defaults", Nav: "settings", User: u, Settings: toKV(raw)}
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
	data := pageData{Title: "Amend defaults", Nav: "settings", User: u, Settings: toKV(raw), Amend: true}
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
	for k, vs := range r.Form {
		if k == "" || len(vs) == 0 {
			continue
		}
		if err := s.store.SetSetting(r.Context(), k, vs[0]); err != nil {
			raw, _ := s.store.Settings(r.Context())
			s.render(w, "settings", pageData{Title: "Amend defaults", Nav: "settings", User: u, Settings: toKV(raw), Amend: true, Error: err.Error()})
			return
		}
	}
	http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
}

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
	if s.internalTok == "" {
		return false
	}
	got := r.Header.Get("X-Internal-Token")
	if got == "" {
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			got = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	return got == s.internalTok
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) *User {
	u := s.currentUser(r)
	if u == nil || !HasCap(*u, "web_admin") {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
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

func (s *Server) render(w http.ResponseWriter, name string, data pageData) {
	t := s.pages[name]
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
