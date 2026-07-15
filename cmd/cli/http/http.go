package http

import (
	"log"
	"net/http"
	"os"

	"github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/cmd/cli/http/middleware"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/rest"
)

func Serve() {
	// Step0: 检查服务启动配置
	port := os.Getenv("OPENAGENT_PORT")
	if port == "" {
		port = "8080"
	}
	maxTurn := 10

	// Step1: Discover WASM plugins (tools, model providers)
	tr := loadTools()

	// Step2: 加载LLM (includes model provider plugins)
	models := loadModels(tr.Mgr, tr.NativeMgr)

	// Step3: 加载Memory和Store
	mem := loadMemory()
	sessionStore := loadSessionStore(mem)

	// Step4: build handler
	llm := openai.New(models[0].ApiKey, models[0].ModelId, models[0].BaseUrl).WithContextWindow(128_000)

	var agentOptions []openagent.AgentOption
	agentOptions = append(agentOptions, openagent.WithModel(llm))
	agentOptions = append(agentOptions, openagent.WithMemory(mem))
	agentOptions = append(agentOptions, openagent.WithInstructions("You are a capable assistant. Be concise and action-oriented."))
	agentOptions = append(agentOptions, openagent.WithTools(tr.Tools...))
	agentOptions = append(agentOptions, openagent.WithMaxTurns(maxTurn))

	agent := openagent.NewAgent("assistant", agentOptions...)

	handler := rest.NewHandler(agent).WithSessionStore(sessionStore)

	// Step5: register available model to handler
	for _, o := range models {
		log.Printf("DEBUG: registering model: modelId=%s, baseUrl=%s, label=%s", o.ModelId, o.BaseUrl, o.Label)
		handler.RegisterModel(o.ModelId, openai.New(o.ApiKey, o.ModelId, o.BaseUrl), o.Label)
	}

	// ── Routes ──
	mux := http.NewServeMux()

	// Health check.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	handler.Register(mux)

	// Step6: Register Middleware
	var h http.Handler = mux
	h = middleware.CorsMiddleware(h)
	h = middleware.LoggingMiddleware(h)
	h = middleware.RecoveryMiddleware(h)

	log.Println("Backend listening on :" + port)
	log.Println("  Health:  GET  /health")
	log.Println("  Single:  POST /sessions, GET /sessions, POST /sessions/{id}/chat")
	log.Println("  Team:    POST /team/sessions, ...")
	log.Println("  Plan:    POST /plan/sessions, ...")
	log.Fatal(http.ListenAndServe(":"+port, h))
}
