// iac-server is a cloud IaC MCP server. It exposes 11 tools over
// MCP stdio so any MCP client (Claude Code, opencode, Cursor, openagent) can
// plan, update, estimate cost, apply, troubleshoot, and destroy cloud
// infrastructure, and query existing cloud resources/bills.
//
// Configuration is via environment variables:
//
//	CLOUD          cloud provider: "huaweicloud" (default), "aliyun"
//	IAC_API_KEY    server-side LLM API key
//	IAC_BASE_URL   server-side LLM base URL (OpenAI-compatible)
//	IAC_MODEL      server-side LLM model ID
//	IAC_HOME       iac-server home (default: ~/.openagent/mcp/iac-server)
//	              skills + deployments live under $IAC_HOME/<cloud>/
//	IAC_DRY_RUN    "true" = simulate, don't call terraform binary
//	TF_BINARY_MIRRORS   comma-separated terraform binary download mirror URLs
//	TF_PROVIDER_MIRRORS comma-separated provider mirror URLs or local paths
//
// Cloud credentials are read from the environment by the selected provider
// (e.g. HW_ACCESS_KEY, HW_SECRET_KEY, HW_REGION, HW_SECURITY_TOKEN for
// huaweicloud). The http_request tool uses AK/SK for SDK-HMAC-SHA256 signing
// — credentials never enter the LLM context.
//
// Skills (deployment guide + references, pricing guide, troubleshoot guide)
// are embedded at compile time (go:embed) and extracted to disk at startup.
// The server-side LLM gets the relevant SKILL.md injected into its system
// prompt and browses reference files with standard read/grep/ls tools.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yusheng-g/openagent-go/cmd/mcp/iac-server/agent"
	iacmcp "github.com/yusheng-g/openagent-go/cmd/mcp/iac-server/mcp"
	"github.com/yusheng-g/openagent-go/cmd/mcp/iac-server/provider"
	"github.com/yusheng-g/openagent-go/cmd/mcp/iac-server/provider/aliyun"
	"github.com/yusheng-g/openagent-go/cmd/mcp/iac-server/provider/huaweicloud"
	"github.com/yusheng-g/openagent-go/mcp"
	"github.com/yusheng-g/openagent-go/model/openai"
	sqlitememory "github.com/yusheng-g/openagent-go/memory/sqlite"
	skillfs "github.com/yusheng-g/openagent-go/skill/fs"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// ── Logging ──
	// Write logs to a file (stderr is captured by the MCP client and only
	// surfaced on connection failure, so it's not reliable for debugging).
	iacHome := os.Getenv("IAC_HOME")
	if iacHome == "" {
		home, _ := os.UserHomeDir()
		iacHome = filepath.Join(home, ".openagent", "mcp", "iac-server")
	}
	logPath := filepath.Join(iacHome, "iac-server.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fatal(fmt.Errorf("open log file: %w", err))
	}
	defer logFile.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelDebug})))

	// ── Select cloud provider ──
	cloud, err := selectProvider(os.Getenv("CLOUD"))
	if err != nil {
		fatal(err)
	}
	slog.Info("selected cloud provider", "cloud", cloud.Name())

	// ── Server-side LLM ──
	apiKey := os.Getenv("IAC_API_KEY")
	modelID := os.Getenv("IAC_MODEL")
	baseURL := os.Getenv("IAC_BASE_URL")
	if apiKey == "" {
		fatal(fmt.Errorf("IAC_API_KEY is required"))
	}
	if modelID == "" {
		fatal(fmt.Errorf("IAC_MODEL is required"))
	}
	model := openai.New(apiKey, modelID, baseURL)
	slog.Info("server LLM configured", "model", modelID, "base_url", baseURL)

	// ── Extract embedded skills to disk ──
	// Skills are embedded via go:embed but the standard skill loader and
	// read/grep/ls tools operate on the OS filesystem. Extract on every
	// startup, overwriting existing files so the disk copy always matches
	// the embedded version.
	cloudHome := filepath.Join(iacHome, cloud.Name())
	skillsDir := filepath.Join(cloudHome, "skills")
	if err := provider.ExtractSkills(cloud.Skills(), skillsDir); err != nil {
		fatal(fmt.Errorf("extract skills: %w", err))
	}
	slog.Info("skills directory", "path", skillsDir)

	// ── Skill loader ──
	loader := skillfs.New(skillsDir)

	// ── Deployments ──
	// Each cloud gets its own subtree: $IAC_HOME/<cloud>/d-NNN/.
	deploymentsDir := filepath.Join(cloudHome, "deployments")
	if err := os.MkdirAll(deploymentsDir, 0755); err != nil {
		fatal(fmt.Errorf("create deployments dir: %w", err))
	}
	slog.Info("deployments directory", "path", deploymentsDir)

	// ── Memory ──
	// SQLite-backed conversation memory scoped by deployment_id. This lets
	// estimate_cost see plan_deployment's reasoning, troubleshoot see prior
	// attempts, etc. FTS5 gives fast full-text search across history.
	memoryPath := filepath.Join(cloudHome, "memory.db")
	sqliteMem, err := sqlitememory.New(memoryPath)
	if err != nil {
		fatal(fmt.Errorf("create memory: %w", err))
	}
	defer sqliteMem.Close()
	slog.Info("memory database", "path", memoryPath)

	dryRun := os.Getenv("IAC_DRY_RUN") == "true"

	// ── Terraform mirrors (for networks with restricted access) ──
	binaryMirrors := splitCSV(os.Getenv("TF_BINARY_MIRRORS"))
	providerMirrors := splitCSV(os.Getenv("TF_PROVIDER_MIRRORS"))
	if len(binaryMirrors) > 0 || len(providerMirrors) > 0 {
		slog.Info("terraform mirrors configured",
			"binary_mirrors", binaryMirrors, "provider_mirrors", providerMirrors)
	}

	// ── Assemble planner + tools ──
	planner := agent.New(model, cloud, loader, sqliteMem, cloudHome, deploymentsDir, dryRun, binaryMirrors, providerMirrors)
	tools := iacmcp.NewTools(iacmcp.Config{
		Planner:         planner,
		Cloud:           cloud,
		DeploymentsDir:  deploymentsDir,
		DryRun:          dryRun,
		BinaryMirrors:   binaryMirrors,
		ProviderMirrors: providerMirrors,
	})
	slog.Info("registered tools", "count", len(tools))

	mcpServerName := cloud.Name() + "-iac-server"
	// ── Start MCP server ──
	server := mcp.NewServer(mcpServerName, "0.0.1", &mcp.ServerOptions{
		Logger: slog.Default(),
	})
	if err := server.AddTools(tools); err != nil {
		fatal(err)
	}

	slog.Info("starting iac-server on stdio")
	if err := server.Run(ctx, &mcpsdk.StdioTransport{}); err != nil {
		fatal(err)
	}
}

// selectProvider chooses a CloudProvider by name.
func selectProvider(name string) (provider.CloudProvider, error) {
	switch name {
	case "", "huaweicloud":
		return huaweicloud.New(os.Getenv("HW_REGION")), nil
	case "aliyun":
		return aliyun.New(os.Getenv("ALIYUN_REGION")), nil
	default:
		return nil, fmt.Errorf("unknown cloud provider: %s (supported: huaweicloud, aliyun)", name)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "iac-server: %v\n", err)
	os.Exit(1)
}

// splitCSV splits a comma-separated string into a slice, trimming whitespace
// and dropping empty entries.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
