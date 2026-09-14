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

	"github.com/samuel/ai-workers/agents/internal/admin"
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
	textCfg, visionCfg := llm.ConfigsFromEnv()
	textCfg.Timeout = 120 * time.Second
	visionCfg.Timeout = 120 * time.Second
	stack, completerErr := llm.NewStack(textCfg, visionCfg)
	completer := stack.Completer
	if completerErr != nil {
		log.Printf("llm disabled until env is set: %v", completerErr)
	} else if stack.VisionModel != "" {
		log.Printf("llm text=%s/%s vision=%s", stack.Provider, stack.Model, stack.VisionModel)
	} else {
		log.Printf("llm text=%s/%s (no vision)", stack.Provider, stack.Model)
	}

	mux := http.NewServeMux()
	admin.RegisterPublic(mux)
	var adminReady bool
	if dsn := strings.TrimSpace(os.Getenv("ADMIN_DATABASE_URL")); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		store, err := admin.Open(ctx, dsn)
		cancel()
		if err != nil {
			log.Printf("admin db: %v", err)
			admin.RegisterUnavailable(mux)
		} else {
			seedCtx, seedCancel := context.WithTimeout(context.Background(), 8*time.Second)
			if err := store.Seed(seedCtx, os.Getenv("ADMIN_BOOTSTRAP_NAME"), os.Getenv("ADMIN_BOOTSTRAP_PHONE"), os.Getenv("ADMIN_BOOTSTRAP_EMAIL"), os.Getenv("ADMIN_BOOTSTRAP_PASSWORD")); err != nil {
				log.Printf("admin seed: %v", err)
			}
			seedCancel()
			desk, err := admin.New(store, os.Getenv("INTERNAL_API_TOKEN"))
			if err != nil {
				log.Printf("admin ui: %v", err)
				store.Close()
				admin.RegisterUnavailable(mux)
			} else {
				desk.Register(mux)
				adminReady = true
				log.Printf("admin desk enabled")
			}
		}
	} else {
		admin.RegisterUnavailable(mux)
	}

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "ok",
			"llm_ready":    completerErr == nil,
			"llm_provider": string(stack.Provider),
			"llm_model":    stack.Model,
			"vision_model": stack.VisionModel,
			"vision_ready": stack.VisionModel != "",
			"admin_ready":  adminReady,
			"departments":  []string{"internal-ops", "accounts", "growth", "product-dev", "community", "crm"},
		})
	})
	mux.HandleFunc("POST /departments/internal-ops", departmentHandler(completer, completerErr, internalops.Handle))
	mux.HandleFunc("POST /departments/accounts", departmentHandler(completer, completerErr, internalops.HandleAccounts))
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
		r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
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
