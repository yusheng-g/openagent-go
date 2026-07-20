package server

import (
	"context"
	"log"
	"path/filepath"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/acp"
	openacpsdk "github.com/yusheng-g/openagent-go/acp/sdk"
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

	_, modelInfos := buildModels(cfg.Provider)
	if len(modelInfos) == 0 {
		log.Println("WARNING: no models configured — ACP server will start but prompt turns will fail")
	}

	modelMap := make(map[string]openagent.Model, len(modelInfos))
	for _, mi := range modelInfos {
		key := mi.ID
		if mi.Provider != "" {
			key = mi.Provider + "/" + mi.ID
		}
		modelMap[key] = mi.Model
	}

	// Enable summarizer with the first model as summarization backend.
	if len(modelInfos) > 0 {
		m := modelInfos[0].Model
		mem.WithSummarizer(summarizer.New(m))
	}

	// Tools and sandbox are created per-turn in agentForTurn so they
	// use the session's cwd rather than the process working directory.
	// This is essential when the agent runs in a container with a
	// different mount path than the host.
	agent := openagent.NewAgent("openagent",
		openagent.WithMemory(mem),
		openagent.WithSystemPrompts(resolveProfiles(cfg.Profiles)...),
		openagent.WithMaxTurns(100),
	)

	srv := acp.NewAgentServer(agent, mem, sessionStore, modelMap)
	srv.ToolFactory = func(cwd string) []openagent.Tool {
		if sb, err := native.New(cwd); err == nil {
			return buildTools(sb, cwd, []string{"shell", "read", "write", "ls", "grep"})
		}
		return nil
	}
	server := openacpsdk.NewServer("openagent-acp", "1.0.0", srv)
	log.Println("ACP server starting on stdio")
	return server.Run(ctx)
}
