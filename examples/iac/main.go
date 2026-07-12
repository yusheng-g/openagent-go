// IaC Agent — Infrastructure as Code powered by openagent-go.
//
// Multi-stage cloud deployment via Plan handler + SSE streaming.
//
// Run:
//
//	export OPENAGENT_API_KEY=sk-xxx
//	export OPENAGENT_MODEL=deepseek-v3
//	export OPENAGENT_BASE_URL=https://api.deepseek.com/v1
//	go run ./examples/iac/
//
// Frontend (optional):
//
//	export FRONTEND_DIR=examples/frontend/vue-app/dist
//
// Real deployments (HuaweiCloud):
//
//	export DRY_RUN=false
//	export HW_ACCESS_KEY=xxx HW_SECRET_KEY=xxx HW_REGION=cn-north-4
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/memory/file"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/rest"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	opentool "github.com/yusheng-g/openagent-go/tool"

	iactools "github.com/yusheng-g/openagent-go/examples/iac/tools"
)

func main() {
	// ── Configuration ──
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAGENT_API_KEY not set")
	}
	modelID := os.Getenv("OPENAGENT_MODEL")
	if modelID == "" {
		modelID = "deepseek-v3"
	}
	baseURL := os.Getenv("OPENAGENT_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dryRun := os.Getenv("DRY_RUN") != "false"

	workDir, _ := filepath.Abs(".")
	iacDir := filepath.Join(workDir, "iac-workspace")

	// ── Model ──
	model := openai.New(apiKey, modelID, baseURL).WithContextWindow(128_000)

	// ── Memory ──
	mem, err := file.New(filepath.Join(iacDir, "memory"))
	if err != nil {
		log.Fatalf("memory: %v", err)
	}
	defer mem.Close()

	// ── File tools ──
	// Agents use read_file/write_file/ls to inspect templates and write .tf files.
	var fileTools []openagent.Tool
	if _, err := native.New(iacDir); err == nil {
		fileTools = []openagent.Tool{
			opentool.NewReadFile(iacDir),
			opentool.NewWriteFile(iacDir),
			opentool.NewListDir(iacDir),
		}
	} else {
		log.Printf("WARNING: sandbox unavailable (%v), file tools disabled", err)
	}

	// ── Extract embedded assets into workspace ──
	if err := extractTemplates(iacDir); err != nil {
		log.Fatalf("templates: %v", err)
	}
	skillDir, err := extractSkills(iacDir)
	if err != nil {
		log.Printf("WARNING: skills: %v (continuing without)", err)
	}

	// ── Terraform Tool ──
	tfTool := iactools.NewTerraformTool(iacDir, dryRun)
	if err := tfTool.EnsureDir(); err != nil {
		log.Fatalf("terraform: %v", err)
	}

	// ── Agents ──
	agentDefs := buildIACAgents(model, mem, tfTool, fileTools, skillDir)

	var planTemplates []rest.PlanAgentTemplate
	for _, def := range agentDefs {
		planTemplates = append(planTemplates, rest.PlanAgentTemplate{
			Name: def.Name, Description: def.Description, Runner: def.Runner,
		})
	}

	ph := rest.NewPlanHandler(mem, model, planTemplates...)
	ph.WithSessionTTL(6 * time.Hour)

	// ── Routes ──
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","dry_run":%v,"mode":"iac"}`, dryRun)
	})
	ph.Register(mux)
	mux.HandleFunc("GET /sessions", stubList)
	mux.HandleFunc("GET /team/sessions", stubList)

	if frontendDir := os.Getenv("FRONTEND_DIR"); frontendDir != "" {
		serveFrontend(mux, frontendDir)
		log.Printf("   frontend: %s", frontendDir)
	}

	log.Printf("🏗️  openagent IaC Agent :%s  dry_run=%v  model=%s  agents=%d",
		port, dryRun, modelID, len(agentDefs))

	srv := &http.Server{
		Addr: ":" + port, Handler: recoveryMiddleware(loggingMiddleware(corsMiddleware(mux))),
		ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Minute, IdleTimeout: 2 * time.Minute,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// ── Embedded asset extraction ──

func extractTemplates(workspace string) error {
	tmplDir := filepath.Join(workspace, "templates")
	if err := os.MkdirAll(tmplDir, 0755); err != nil {
		return err
	}
	entries, err := templatesFS.ReadDir("templates")
	if err != nil {
		return err
	}
	for _, e := range entries {
		b, err := templatesFS.ReadFile("templates/" + e.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(tmplDir, e.Name()), b, 0644); err != nil {
			return err
		}
	}
	log.Printf("   templates: %d → %s", len(entries), tmplDir)
	return nil
}

func extractSkills(workspace string) (string, error) {
	entries, err := skillsFS.ReadDir("skills")
	if err != nil {
		return "", err
	}
	skillDir := filepath.Join(workspace, "skills")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := skillsFS.ReadFile("skills/" + e.Name() + "/SKILL.md")
		if err != nil {
			continue
		}
		os.MkdirAll(filepath.Join(skillDir, e.Name()), 0755)
		os.WriteFile(filepath.Join(skillDir, e.Name(), "SKILL.md"), b, 0644)
	}
	log.Printf("   skills: %d → %s", len(entries), skillDir)
	return skillDir, nil
}

// ── Middleware ──

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(wrapped, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start).Round(time.Millisecond))
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) { rw.statusCode = code; rw.ResponseWriter.WriteHeader(code) }
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
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

func stubList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("[]"))
}

func serveFrontend(mux *http.ServeMux, frontendDir string) {
	fsrv := http.FileServer(http.Dir(frontendDir))
	mux.Handle("GET /{path...}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.PathValue("path")
		full := filepath.Join(frontendDir, path)
		if _, err := os.Stat(full); os.IsNotExist(err) {
			for _, ext := range []string{".js", ".css", ".svg", ".ico", ".png", ".woff", ".woff2"} {
				if len(path) >= len(ext) && path[len(path)-len(ext):] == ext {
					http.NotFound(w, r)
					return
				}
			}
			http.ServeFile(w, r, filepath.Join(frontendDir, "index.html"))
			return
		}
		fsrv.ServeHTTP(w, r)
	}))
}
