package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchSitePagesSameHost(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/about" {
			w.Write([]byte(`<html><head><title>About</title></head><body>About us in Lagos</body></html>`))
			return
		}
		w.Write([]byte(`<html><head><title>Home</title><meta name="description" content="Lagos books"></head><body><a href="/about">About</a><a href="https://evil.example/x">skip</a> Welcome</body></html>`))
	}))
	defer srv.Close()
	old := seoFetchClient
	seoFetchClient = srv.Client()
	defer func() { seoFetchClient = old }()

	pages := fetchSitePages(srv.URL + "/")
	if len(pages) != 2 {
		t.Fatalf("pages %d %+v", len(pages), pages)
	}
	if pages[0].Title != "Home" || !strings.Contains(pages[0].Meta, "Lagos books") {
		t.Fatalf("%+v", pages[0])
	}
	if pages[1].Title != "About" {
		t.Fatalf("%+v", pages[1])
	}
}

