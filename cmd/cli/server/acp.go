package server

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/yusheng-g/openagent-go/acp"
	openacpsdk "github.com/yusheng-g/openagent-go/acp/sdk"
	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/summarizer"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
)

// RunACP starts the agent in ACP mode over stdio.
//
// Lifecycle:
//  1. Open memory + session store (SQLite).
//  2. Build models from config.
//  3. Create sandbox + standard tools.
//  4. Wire summarizer for long-conversation compression.
//  5. Construct the agent.
//  6. Wrap in AgentServer, launch ACP protocol mux on stdin/stdout.
func RunACP(ctx context.Context, cfg *config.Config) error {
	cfgPath, _ := config.Path()
	dataDir := filepath.Join(filepath.Dir(cfgPath), "data")

	mem, sessionStore, cleanup, err := buildMemory(filepath.Join(dataDir, "memory.db"))
	if err != nil {
		return err
	}
	defer cleanup()

	models, _ := buildModels(cfg.Provider)

	m := firstModel(models)
	if m == nil {
		log.Println("WARNING: no models configured — ACP server will start but prompt turns will fail")
	}

	workDir, _ := os.Getwd()
	var tools []openagent.Tool
	if sb, err := native.New(workDir); err == nil {
		tools = buildTools(sb, workDir, []string{"shell", "read", "write", "ls", "grep"})
	} else {
		log.Printf("WARNING: sandbox unavailable, tools disabled: %v", err)
	}

	if m != nil {
		mem.WithSummarizer(summarizer.New(m))
	}

	agent := openagent.NewAgent("openagent",
		openagent.WithModel(m),
		openagent.WithMemory(mem),
		openagent.WithSystemPrompts("You are a helpful AI assistant. Use tools to read, write, and execute code when needed."),
		openagent.WithTools(tools...),
		openagent.WithMaxTurns(10),
	)

	srv := acp.NewAgentServer(agent, mem, sessionStore)
	server := openacpsdk.NewServer("openagent-acp", "1.0.0", srv)
	log.Println("ACP server starting on stdio")
	return server.Run(ctx)
}
