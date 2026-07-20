package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/rest"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/summarizer"
	opentool "github.com/yusheng-g/openagent-go/tool"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
)

// ── REST server ──

// RunREST starts the REST API server (HTTP + SSE).
func RunREST(ctx context.Context, cfg *config.Config) error {
	models, modelInfos := buildModels(cfg.Provider)
	m := firstModel(models)

	workDir, _ := os.Getwd()
	sb, err := native.NewWithPolicy(workDir, sandboxPolicy(cfg.Sandbox))
	var tools []openagent.Tool
	if err == nil {
		tools = buildTools(sb, workDir, []string{"shell", "read", "write", "ls", "grep"})
	} else {
		log.Printf("WARNING: sandbox unavailable, tools disabled: %v", err)
	}

	cfgPath, _ := config.Path()
	dataDir := filepath.Join(filepath.Dir(cfgPath), "data")
	mem, store, cleanup, err := buildMemory(filepath.Join(dataDir, "memory.db"))
	if err != nil {
		return err
	}
	defer cleanup()
	if m != nil {
		mem.WithSummarizer(summarizer.New(m))
	}

	agent := openagent.NewAgent("openagent",
		openagent.WithModel(m),
		openagent.WithMemory(mem),
		openagent.WithSystemPrompts(resolveProfiles(cfg.Profiles)...),
		openagent.WithTools(tools...),
		openagent.WithMaxTurns(100),
	)

	handler := rest.NewHandler(agent).
		WithSessionStore(store).
		WithCleanupDir(func(sessionID string) {
			dir := filepath.Join(opentool.ArtifactRoot(), sessionID)
			_ = os.RemoveAll(dir)
		})
	handler.StartJanitor(ctx, 1*time.Hour, 24*time.Hour)
	for _, mi := range modelInfos {
		handler.RegisterModel(mi.ID, mi.Model, mi.Provider)
	}

	// Start IM channels in the background.
	if err := RunChannels(ctx, agent, cfg.Channels); err != nil {
		log.Printf("channel: %v", err)
	}

	mux := http.NewServeMux()
	handler.Register(mux)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: withMiddleware(mux)}

	go func() { <-ctx.Done(); log.Println("shutting down..."); srv.Shutdown(context.Background()) }()

	log.Printf("REST server listening on http://localhost%s", addr)
	err = srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// ── Middleware ──

func withMiddleware(next http.Handler) http.Handler {
	return recoveryMiddleware(corsMiddleware(next))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC %s %s: %v", r.Method, r.URL.Path, rec)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
