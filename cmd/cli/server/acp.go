package server

import (
	"context"
	"log"
	"log/slog"
	"os"
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
func RunACP(ctx context.Context, cfg *config.Config, caps Capabilities) error {
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

	// Summarizer and Memory are enabled by default; allow --summarizer=off
	// and --memory=off to disable them.
	var firstM openagent.Model
	if len(modelInfos) > 0 {
		firstM = modelInfos[0].Model
		if caps.OnSummarizer() {
			mem.WithSummarizer(summarizer.New(firstM))
		}
	}

	// Tools and sandbox are created per-turn in agentForTurn so they
	// use the session's cwd rather than the process working directory.
	opts := []openagent.AgentOption{
		openagent.WithSystemPrompts(resolveProfiles(cfg.Profiles)...),
		openagent.WithMaxTurns(100),
	}
	if caps.OnMemory() {
		opts = append(opts, openagent.WithMemory(mem))
	}
	opts = buildOpts(opts, caps, firstM)
	agent := openagent.NewAgent("openagent", opts...)

	// Pass nil Mem when --memory=off so the AgentServer skips history
	// replay and memory cleanup (all s.Mem uses are nil-guarded). The
	// sessionStore (session metadata) is separate and unaffected.
	var serverMem openagent.Memory = mem
	if !caps.OnMemory() {
		serverMem = nil
	}
	srv := acp.NewAgentServer(agent, serverMem, sessionStore, modelMap)
	srv.MCPEnabled = caps.OnMCP()
	policy := sandboxPolicy(cfg.Sandbox)
	if caps.OnTools() {
		srv.ToolFactory = func(cwd string) []openagent.Tool {
			if sb, err := native.NewWithPolicy(cwd, policy); err == nil {
				return buildTools(sb, cwd, []string{"shell", "read", "write", "ls", "grep"})
			}
			return nil
		}
	}
	server := openacpsdk.NewServer("openagent-acp", "1.0.0", srv)
	server.SetLogger(slog.Default())

	// Channel agent: clone the template and inject a default Model + Tools
	// so the IM bot can run standalone (ACP path injects Model per-turn in
	// agentForTurn, but channels call agent.RunStream() directly).
	channelAgent := agent.Clone()
	for _, mi := range modelInfos {
		if mi.Model != nil {
			channelAgent.Model = mi.Model
			break
		}
	}
	if caps.OnTools() {
		cwd, _ := os.Getwd()
		if sb, err := native.NewWithPolicy(cwd, policy); err == nil {
			channelAgent.Tools = buildTools(sb, cwd, []string{"shell", "read", "write", "ls", "grep"})
		}
	}

	if err := RunChannels(ctx, channelAgent, cfg.Channels); err != nil {
		log.Printf("channel: %v", err)
	}

	log.Println("ACP server starting on stdio")
	return server.Run(ctx)
}
