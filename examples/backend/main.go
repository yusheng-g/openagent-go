// Full-stack backend example: serves REST + SSE endpoints for single-agent,
// team, and plan modes. Designed to work with separate frontend deployment
// (CORS enabled).
//
//	go run ./examples/backend/
//
//	# Environment:
//	OPENAGENT_API_KEY=sk-... OPENAGENT_MODEL=gpt-4o go run ./examples/backend/
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/memory/sqlite"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/rest"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/tool"
)

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")
	port := os.Getenv("OPENAGENT_PORT")
	if apiKey == "" || modelID == "" {
		log.Fatal("set OPENAGENT_API_KEY and OPENAGENT_MODEL")
	}
	if port == "" {
		port = "8080"
	}

	llm := openai.New(apiKey, modelID, baseURL).WithContextWindow(128_000)

	// ── Memory ──
	mem, err := sqlite.New("./backend-memory.db")
	if err != nil {
		log.Fatalf("memory: %v", err)
	}
	defer mem.Close()

	// ── Sandbox + tools ──
	workDir, _ := filepath.Abs(".")
	sandbox, err := native.New(workDir)
	var sandboxTools []openagent.Tool
	if err == nil {
		sandboxTools = []openagent.Tool{
			tool.NewShell(sandbox, workDir),
			tool.NewReadFile(workDir),
			tool.NewWriteFile(workDir),
			tool.NewListDir(workDir),
			tool.NewGrep(workDir),
		}
	} else {
		log.Printf("WARNING: sandbox unavailable: %v", err)
	}

	// ── Single agent ──
	agent := openagent.NewAgent("assistant",
		openagent.WithModel(llm),
		openagent.WithMemory(mem),
		openagent.WithInstructions("You are a capable assistant. Use shell, read, write, ls, and grep tools to explore, build, and edit the codebase. Be concise and action-oriented."),
		openagent.WithTools(sandboxTools...),
		openagent.WithMaxTurns(10),
	)
	handler := rest.NewHandler(agent)

	// ── Team ──
	analyst := openagent.NewAgent("analyst",
		openagent.WithModel(llm),
		openagent.WithInstructions("You are a requirements analyst. Understand the user's request, break it into clear requirements, then hand off to the designer with a structured spec. Include constraints and acceptance criteria. Use transfer_to_designer when done."),
		openagent.WithMaxTurns(2),
	)
	designer := openagent.NewAgent("designer",
		openagent.WithModel(llm),
		openagent.WithInstructions("You are a software designer. Take the specification, design the architecture with components and interfaces, then hand off to the coder with a clear design document. Use transfer_to_coder when done."),
		openagent.WithMaxTurns(2),
	)
	coder := openagent.NewAgent("coder",
		openagent.WithModel(llm),
		openagent.WithInstructions("You are a software developer. Take the design and write clean, idiomatic Go code. Include error handling and comments. Hand off to the reviewer with your complete implementation. Use transfer_to_reviewer when done."),
		openagent.WithMaxTurns(5),
	)
	reviewer := openagent.NewAgent("reviewer",
		openagent.WithModel(llm),
		openagent.WithInstructions("You are a code reviewer. Review the implementation for correctness, style, and bugs. If issues found, hand off back to the coder with specific feedback. If approved, produce the final summary for the user. Do NOT hand off further — you are the final gate."),
		openagent.WithMaxTurns(2),
	)
	teamHandler := rest.NewTeamHandler(nil,
		rest.TeamAgentTemplate{Name: "analyst", Description: "Understands requirements and produces specifications", Agent: analyst},
		rest.TeamAgentTemplate{Name: "designer", Description: "Designs architecture, components, and data flow", Agent: designer},
		rest.TeamAgentTemplate{Name: "coder", Description: "Writes clean, idiomatic Go code with error handling", Agent: coder},
		rest.TeamAgentTemplate{Name: "reviewer", Description: "Reviews code for correctness, style, and security", Agent: reviewer},
	)

	// ── Plan agents ──
	planResearcher := openagent.NewAgent("researcher",
		openagent.WithModel(llm),
		openagent.WithInstructions("You research technical topics thoroughly. Use read/ls/grep tools to explore the codebase, shell to run commands. Be objective and data-driven."),
		openagent.WithMaxTurns(2),
		openagent.WithTools(sandboxTools...),
	)
	planArchitect := openagent.NewAgent("architect",
		openagent.WithModel(llm),
		openagent.WithInstructions("You design software architecture. Use read/ls tools to understand existing code. Only output your design — no follow-up questions."),
		openagent.WithMaxTurns(1),
		openagent.WithTools(sandboxTools...),
	)
	planCoder := openagent.NewAgent("coder",
		openagent.WithModel(llm),
		openagent.WithInstructions("You write production-quality Go code. Use read/write to edit files, grep to search, shell to build and test. Output ONLY code — no explanations outside code comments."),
		openagent.WithMaxTurns(3),
		openagent.WithTools(sandboxTools...),
	)
	planReviewer := openagent.NewAgent("reviewer",
		openagent.WithModel(llm),
		openagent.WithInstructions("You review code for correctness, style, and potential bugs. Use read/grep to examine the code. List specific issues and suggestions. Be constructive."),
		openagent.WithMaxTurns(1),
		openagent.WithTools(sandboxTools...),
	)
	planWriter := openagent.NewAgent("writer",
		openagent.WithModel(llm),
		openagent.WithInstructions("You write clear documentation. Use read/ls to understand the codebase. Use markdown formatting."),
		openagent.WithMaxTurns(1),
		openagent.WithTools(sandboxTools...),
	)
	planHandler := rest.NewPlanHandler(nil, llm,
		rest.PlanAgentTemplate{Name: "researcher", Description: "Researches technical topics — provides comprehensive analysis with pros/cons, alternatives, and data-driven recommendations", Runner: planResearcher},
		rest.PlanAgentTemplate{Name: "architect", Description: "Designs software architecture — produces structured design documents with components, interfaces, and data flow", Runner: planArchitect},
		rest.PlanAgentTemplate{Name: "coder", Description: "Writes production-quality Go code with error handling and comments", Runner: planCoder},
		rest.PlanAgentTemplate{Name: "reviewer", Description: "Reviews code for correctness, style, and potential bugs — produces a list of issues and suggestions", Runner: planReviewer},
		rest.PlanAgentTemplate{Name: "writer", Description: "Writes clear, professional documentation: README, API docs, reports. Uses markdown formatting", Runner: planWriter},
	)

	// ── Routes ──
	mux := http.NewServeMux()

	// Health check.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	handler.Register(mux)
	teamHandler.Register(mux)
	planHandler.Register(mux)

	// ── Middleware stack (outermost first) ──
	var h http.Handler = mux
	h = corsMiddleware(h)
	h = loggingMiddleware(h)
	h = recoveryMiddleware(h)

	log.Println("Backend listening on :" + port)
	log.Println("  Health:  GET  /health")
	log.Println("  Single:  POST /sessions, GET /sessions, POST /sessions/{id}/chat")
	log.Println("  Team:    POST /team/sessions, ...")
	log.Println("  Plan:    POST /plan/sessions, ...")
	log.Fatal(http.ListenAndServe(":"+port, h))
}

// ── Middleware ──

// recoveryMiddleware catches panics in downstream handlers, logs the stack
// trace, and returns 500 instead of letting the connection hang.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ── Middleware ──

// corsMiddleware sets permissive CORS headers for development.
// In production, restrict Access-Control-Allow-Origin to the frontend's origin.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs each request with method, path, status, and duration.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s → %d (%s)", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Microsecond))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher so SSE streaming works through the
// logging middleware. http.Flusher is not part of http.ResponseWriter,
// so embedding the interface does not automatically promote it.
func (sw *statusWriter) Flush() {
	if f, ok := sw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
