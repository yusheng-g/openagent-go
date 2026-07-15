package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/cmd/cli/http"
	"github.com/yusheng-g/openagent-go/cmd/cli/plugin"
	nativeplugin "github.com/yusheng-g/openagent-go/cmd/cli/plugin/native"
	"github.com/yusheng-g/openagent-go/memory/sqlite"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	opentool "github.com/yusheng-g/openagent-go/tool"
)

// ── Command registry ──

// Command is a CLI sub-command that can be registered and dispatched.
type Command struct {
	// Name is the sub-command string (e.g. "run", "goal", "serve").
	Name string
	// ShortHelp is a one-line description shown in usage.
	ShortHelp string
	// Run executes the command. args are the positional args after the sub-command name.
	Run func(ctx context.Context, cfg *Config, args []string)
}

var commands []*Command

// RegisterCommand adds a sub-command to the CLI.
// Call from an init() function in any file within this package.
func RegisterCommand(cmd *Command) {
	commands = append(commands, cmd)
}

// ── Shared configuration ──

// Config holds the resolved CLI configuration and constructed resources
// shared across sub-commands (model, memory, tools, etc.).
type Config struct {
	ModelID         string
	APIKey          string
	BaseURL         string
	WorkDir         string
	MemPath         string
	ToolList        string
	PluginDir       string
	NativePluginDir string
	MaxTurns        int

	// Constructed resources (populated by Setup).
	Model           openagent.Model
	Memory          openagent.Memory
	Tools           []openagent.Tool
	PluginMgr       *plugin.Manager
	NativePluginMgr *nativeplugin.Manager
	Instructions    string
}

// Setup constructs shared resources from the configuration.
// It must be called before using Model, Memory, Tools, or PluginMgr.
func (c *Config) Setup() {
	if c.WorkDir == "" {
		c.WorkDir, _ = filepath.Abs(".")
	}

	c.Model = openai.New(c.APIKey, c.ModelID, c.BaseURL).WithContextWindow(128_000)

	if c.MemPath != "" {
		mem, err := sqlite.New(c.MemPath)
		if err != nil {
			log.Fatalf("open memory: %v", err)
		}
		c.Memory = mem
	}

	c.Tools, c.PluginMgr, c.NativePluginMgr = buildTools(c.WorkDir, c.ToolList, c.PluginDir, c.NativePluginDir)

	if c.Instructions == "" {
		c.Instructions = `You are a precise, no-nonsense assistant running in a CLI.
- Answer concisely. Don't explain unless asked.
- If you need to explore, do it in one or two well-targeted commands — don't retry the same thing.
- When a tool returns an error, read the error and adjust. Don't repeat the same failing call.
- Use relative paths only (the workspace is the current directory).
- Stop when the task is done. Don't keep exploring.`
	}
}

// NewAgent builds an agent from the shared config.
func (c *Config) NewAgent(name string, maxTurns int) *openagent.Agent {
	return newAgent(name, c.Model, c.Memory, c.Instructions, c.Tools, maxTurns)
}

func newAgent(name string, model openagent.Model, mem openagent.Memory, instructions string, tools []openagent.Tool, maxTurns int) *openagent.Agent {
	opts := []openagent.AgentOption{
		openagent.WithModel(model),
		openagent.WithInstructions(instructions),
		openagent.WithMemory(mem),
		openagent.WithTools(tools...),
		openagent.WithMaxTurns(maxTurns),
	}
	return openagent.NewAgent(name, opts...)
}

// ── Built-in commands ──

func init() {
	RegisterCommand(&Command{
		Name:      "run",
		ShortHelp: "Run a single prompt through the agent",
		Run:       runRun,
	})
	RegisterCommand(&Command{
		Name:      "goal",
		ShortHelp: "Run the agent in autonomous goal mode",
		Run:       runGoal,
	})
	RegisterCommand(&Command{
		Name:      "serve",
		ShortHelp: "Start the HTTP API server",
		Run:       runServe,
	})
}

func runRun(ctx context.Context, cfg *Config, args []string) {
	input := strings.Join(args, " ")
	if input == "" {
		log.Fatal("no input message provided")
	}

	agent := cfg.NewAgent("cli", 20)
	session := openagent.Session{
		ID:             "cli",
		UserID:         "user",
		ModelID:        cfg.ModelID,
		ProjectContext: fmt.Sprintf("Workspace: %s", cfg.WorkDir),
	}
	printStream(agent.RunStream(ctx, session, openagent.UserMessage(input)), true)
}

func runGoal(ctx context.Context, cfg *Config, args []string) {
	input := strings.Join(args, " ")
	if input == "" {
		log.Fatal("no input message provided")
	}

	maxTurns := cfg.MaxTurns
	if maxTurns == 0 {
		maxTurns = 50
	}

	agent := cfg.NewAgent("cli", maxTurns)
	session := openagent.Session{
		ID:             "cli",
		UserID:         "user",
		ModelID:        cfg.ModelID,
		ProjectContext: fmt.Sprintf("Workspace: %s", cfg.WorkDir),
	}
	printStream(agent.RunGoalStream(ctx, session, input), false)
}

func runServe(_ context.Context, _ *Config, _ []string) {
	http.Serve()
}

// ── Main ──

func main() {
	log.SetFlags(0)

	// ── Flags ──
	var (
		modelID         = envOr("OPENAGENT_MODEL", "gpt-4o")
		apiKey          = envOr("OPENAGENT_API_KEY", "")
		baseURL         = os.Getenv("OPENAGENT_BASE_URL")
		memPath         string
		workDir         string
		maxTurns        int
		toolList        string
		pluginDir       string
		nativePluginDir string
	)
	flag.StringVar(&modelID, "model", modelID, "Model ID")
	flag.StringVar(&apiKey, "api-key", apiKey, "API key")
	flag.StringVar(&baseURL, "base-url", baseURL, "Base URL")
	flag.StringVar(&memPath, "memory", "", "SQLite memory path (empty = no persistence)")
	flag.StringVar(&workDir, "workspace", "", "Workspace root (empty = current dir)")
	flag.IntVar(&maxTurns, "max-turns", 0, "Max turns (0 = default 20, goal mode uses 50)")
	flag.StringVar(&toolList, "tools", "shell,read,write,ls,grep", "Built-in tools to enable (comma-separated)")
	flag.StringVar(&pluginDir, "plugin-dir", envOr("OPENAGENT_PLUGIN_DIR", ""), "Directory containing .wasm plugin files")
	flag.StringVar(&nativePluginDir, "native-plugin-dir", envOr("OPENAGENT_NATIVE_PLUGIN_DIR", ""), "Directory containing native binary plugin executables")
	flag.Parse()

	cmd, args := parseCommand(flag.Args())

	if cmd == nil {
		fmt.Fprintf(os.Stderr, "Usage: openagent <command> [flags] [args]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n\n")
		for _, c := range commands {
			fmt.Fprintf(os.Stderr, "  %-12s %s\n", c.Name, c.ShortHelp)
		}
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		os.Exit(2)
	}

	if apiKey == "" {
		log.Fatal("OPENAGENT_API_KEY not set. Use --api-key or set the environment variable.")
	}

	cfg := &Config{
		ModelID:         modelID,
		APIKey:          apiKey,
		BaseURL:         baseURL,
		WorkDir:         workDir,
		MemPath:         memPath,
		ToolList:        toolList,
		PluginDir:       pluginDir,
		NativePluginDir: nativePluginDir,
		MaxTurns:        maxTurns,
	}
	cfg.Setup()

	ctx := context.Background()
	cmd.Run(ctx, cfg, args)
}

// parseCommand finds the registered command matching the first positional arg.
// Returns the command and remaining args, or nil if not found.
func parseCommand(args []string) (*Command, []string) {
	if len(args) == 0 {
		return nil, nil
	}
	for _, c := range commands {
		if c.Name == args[0] {
			return c, args[1:]
		}
	}
	return nil, nil
}

// ── Helpers ──

func printStream(ch <-chan openagent.StreamEvent, fatal bool) {
	for evt := range ch {
		switch evt.Type {
		case openagent.StreamTextDelta:
			fmt.Print(evt.Text)
		case openagent.StreamToolCall:
			if len(evt.Message.ToolCalls) > 0 {
				fmt.Printf("\n🔧 %s", evt.Message.ToolCalls[0].Function.Name)
			}
		case openagent.StreamToolResult:
			text := evt.Message.Content
			if len(text) > 500 {
				text = text[:500] + "..."
			}
			fmt.Printf(" → %s\n", text)
		case openagent.StreamDone:
			fmt.Println()
		case openagent.StreamError:
			if fatal {
				log.Fatalf("error: %v", evt.Error)
			} else {
				log.Printf("error: %v", evt.Error)
			}
		case openagent.StreamAborted:
			if fatal {
				log.Fatalf("aborted: %v", evt.Error)
			} else {
				log.Printf("aborted: %v", evt.Error)
			}
		}
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func buildTools(workDir, toolList, pluginDir, nativePluginDir string) ([]openagent.Tool, *plugin.Manager, *nativeplugin.Manager) {
	var tools []openagent.Tool
	enabled := make(map[string]bool)
	for _, name := range strings.Split(toolList, ",") {
		enabled[strings.TrimSpace(name)] = true
	}

	// Shell tool needs native sandbox.
	if enabled["shell"] {
		sandbox, err := native.New(workDir)
		if err == nil {
			tools = append(tools, opentool.NewShell(sandbox, workDir))
		} else {
			log.Printf("WARNING: sandbox unavailable (%v), shell tool disabled", err)
		}
	}

	// File tools.
	if enabled["read"] {
		tools = append(tools, opentool.NewReadFile(workDir))
	}
	if enabled["write"] {
		tools = append(tools, opentool.NewWriteFile(workDir))
	}
	if enabled["ls"] {
		tools = append(tools, opentool.NewListDir(workDir))
	}
	if enabled["grep"] {
		tools = append(tools, opentool.NewGrep(workDir))
	}

	// ── WASM plugin tools ──
	mgr := plugin.NewManager(pluginDir)
	if pluginDir != "" {
		if err := mgr.Discover(context.Background()); err != nil {
			log.Printf("WARNING: plugin discovery failed: %v", err)
		} else {
			tools = append(tools, mgr.Tools()...)
		}
	}

	// ── Native binary plugin tools ──
	nativeMgr := nativeplugin.NewManager(nativePluginDir)
	if nativePluginDir != "" {
		if err := nativeMgr.Discover(context.Background()); err != nil {
			log.Printf("WARNING: native plugin discovery failed: %v", err)
		} else {
			tools = append(tools, nativeMgr.Tools()...)
		}
	}

	return tools, mgr, nativeMgr
}
