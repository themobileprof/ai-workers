package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments/community"
	"github.com/samuel/ai-workers/agents/internal/departments/crm"
	"github.com/samuel/ai-workers/agents/internal/departments/growth"
	"github.com/samuel/ai-workers/agents/internal/departments/internalops"
	"github.com/samuel/ai-workers/agents/internal/departments/productdev"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

func main() {
	addr := listenAddr()
	completer, completerErr := llm.New(llm.Config{
		Provider: llm.ParseProvider(os.Getenv("LLM_PROVIDER")),
		Model:    os.Getenv("LLM_MODEL"),
		APIKey:   os.Getenv("LLM_API_KEY"),
		BaseURL:  os.Getenv("LLM_BASE_URL"),
		Timeout:  120 * time.Second,
	})
	if completerErr != nil {
		log.Printf("llm disabled until env is set: %v", completerErr)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "ok",
			"llm_ready":    completerErr == nil,
			"llm_provider": strings.ToLower(os.Getenv("LLM_PROVIDER")),
			"departments":  []string{"internal-ops", "growth", "product-dev", "community", "crm"},
		})
	})
	mux.HandleFunc("POST /departments/internal-ops", departmentHandler(completer, completerErr, internalops.Handle))
	mux.HandleFunc("POST /departments/growth", departmentHandler(completer, completerErr, growth.Handle))
	mux.HandleFunc("POST /departments/product-dev", departmentHandler(completer, completerErr, productdev.Handle))
	mux.HandleFunc("POST /departments/community", departmentHandler(completer, completerErr, community.Handle))
	mux.HandleFunc("POST /departments/crm", departmentHandler(completer, completerErr, crm.Handle))

	srv := &http.Server{
		Addr:              addr,
		Handler:           withLogging(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      130 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
	log.Printf("agents listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func departmentHandler(c llm.Completer, initErr error, fn func(context.Context, llm.Completer, contract.Request) (contract.Response, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if initErr != nil || c == nil {
			writeJSON(w, http.StatusServiceUnavailable, contract.Fail("LLM is not configured: "+errString(initErr)))
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		var req contract.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, contract.Fail("invalid JSON: "+err.Error()))
			return
		}
		req.TaskDescription = strings.TrimSpace(req.TaskDescription)
		if req.TaskDescription == "" {
			writeJSON(w, http.StatusBadRequest, contract.Fail("task_description is required"))
			return
		}
		resp, err := fn(r.Context(), c, req)
		if err != nil {
			log.Printf("department error: %v", err)
			if resp.Status == "" {
				resp = contract.Fail(err.Error())
			}
			writeJSON(w, http.StatusBadGateway, resp)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func errString(err error) string {
	if err == nil {
		return "unknown"
	}
	return err.Error()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Truncate(time.Millisecond))
	})
}

func listenAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		if strings.Contains(p, ":") {
			return p
		}
		return ":" + p
	}
	return ":8000"
}
