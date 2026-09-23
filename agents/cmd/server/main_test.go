package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

type stubLLM struct {
	text string
	err  error
}

func (s stubLLM) Complete(context.Context, llm.Request) (string, error) {
	return s.text, s.err
}

func TestListenAddr(t *testing.T) {
	t.Setenv("PORT", "")
	if listenAddr() != ":8000" {
		t.Fatal(listenAddr())
	}
	t.Setenv("PORT", "9000")
	if listenAddr() != ":9000" {
		t.Fatal(listenAddr())
	}
	t.Setenv("PORT", "127.0.0.1:8001")
	if listenAddr() != "127.0.0.1:8001" {
		t.Fatal(listenAddr())
	}
}

func TestErrString(t *testing.T) {
	if errString(nil) != "unknown" {
		t.Fatal(errString(nil))
	}
	if errString(errors.New("no key")) != "no key" {
		t.Fatal(errString(errors.New("no key")))
	}
}

func TestDepartmentHandler(t *testing.T) {
	okFn := func(_ context.Context, _ llm.Completer, req contract.Request) (contract.Response, error) {
		return contract.Response{
			Status:         contract.StatusSuccess,
			OutputText:     "ok " + req.TaskDescription,
			StructuredData: map[string]any{"task_type": "growth"},
		}, nil
	}

	h := departmentHandler(nil, errors.New("no key"), "growth", okFn, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/departments/growth", strings.NewReader(`{"task_description":"hi","context_data":{}}`)))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured %d", rec.Code)
	}

	h = departmentHandler(stubLLM{}, nil, "growth", okFn, nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/departments/growth", strings.NewReader(`{`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/departments/growth", strings.NewReader(`{"task_description":"  ","context_data":{}}`)))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "task_description") {
		t.Fatalf("empty task %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/departments/growth", bytes.NewReader([]byte(`{"task_description":"When can we start?","context_data":{}}`)))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ok %d %s", rec.Code, rec.Body.String())
	}
	var resp contract.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Status != contract.StatusSuccess || resp.OutputText != "ok When can we start?" {
		t.Fatalf("%+v", resp)
	}

	failFn := func(context.Context, llm.Completer, contract.Request) (contract.Response, error) {
		return contract.Response{}, errors.New("llm down")
	}
	rec = httptest.NewRecorder()
	departmentHandler(stubLLM{}, nil, "growth", failFn, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/departments/growth", strings.NewReader(`{"task_description":"hi","context_data":{}}`)))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("gateway %d", rec.Code)
	}

	failWithBody := func(context.Context, llm.Completer, contract.Request) (contract.Response, error) {
		return contract.Fail("classified fail"), errors.New("upstream")
	}
	rec = httptest.NewRecorder()
	departmentHandler(stubLLM{}, nil, "growth", failWithBody, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/departments/growth", strings.NewReader(`{"task_description":"hi","context_data":{}}`)))
	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "classified fail") {
		t.Fatalf("fail body %d %s", rec.Code, rec.Body.String())
	}
}

func TestWithLogging(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	withLogging(inner).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("%d", rec.Code)
	}
}
