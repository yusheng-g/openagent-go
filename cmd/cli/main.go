// hwcloud — openagent-go CLI.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
	"github.com/yusheng-g/openagent-go/cmd/cli/server"
	"github.com/yusheng-g/openagent-go/keyring"
	plugin "github.com/yusheng-g/openagent-go/plugin/cli"
	cliwasm "github.com/yusheng-g/openagent-go/plugin/cli/wasm"
	"github.com/yusheng-g/openagent-go/version"
)

func main() {
	log.SetFlags(0)

	// 1. Paths.
	cfgPath, err := config.Path()
	if err != nil {
		log.Fatalf("config path: %v", err)
	}
	pluginPaths := []string{config.DefaultPluginsDir()}

	// 2. Read settings.json.
	raw, err := os.ReadFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("read settings: %v", err)
	}
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	var preCfg config.Config
	if err := json.Unmarshal(raw, &preCfg); err != nil {
		log.Fatalf("parse settings: %v", err)
	}
	if len(preCfg.Plugins) > 0 {
		pluginPaths = preCfg.Plugins
	}

	// 3. Plugin runtime (keyring + HTTP + logger wired internally).
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// -q/--quiet silences log output for the whole process. cobra parses
	// flags during Execute, but plugin loading happens before that — parse
	// just the persistent flags early (unknown flags tolerated: subcommand
	// locals like --port aren't in the root set yet, and this early parse
	// must not reject them). The whitelist only affects this parse; Execute
	// parses its own FlagSet normally. GetBool also handles explicit
	// negation (--quiet=false) correctly.
	fs := rootCmd.PersistentFlags()
	fs.ParseErrorsWhitelist.UnknownFlags = true
	fs.Parse(os.Args[1:])
	quiet, _ := fs.GetBool("quiet")
	if quiet {
		// Apply BEFORE plugin loading (which happens before cobra's
		// Execute), and again after SetupLog below (which reinstalls the
		// default slog handler) so quiet survives both.
		applyQuiet()
	}

	// 4. Load every .wasm and route capabilities.
	settings := raw
	if len(settings) == 0 {
		settings = []byte("{}")
	}
	hub := &cliwasm.ObserverHub{}

	// keyring subcommands don't use plugins; skip loading to avoid
	// plugin log noise on stdout/stderr.
	if !isKeyringCmd(os.Args) {
		var cleanup func()
		settings, cleanup = loadPlugins(ctx, pluginPaths, settings, hub)
		defer cleanup()
	}

	// 5. Parse final merged config.
	var cfg config.Config
	if err := json.Unmarshal(settings, &cfg); err != nil {
		log.Fatalf("parse merged settings: %v", err)
	}
	for k, v := range cfg.Env {
		os.Setenv(k, v)
	}
	// Defaults come from the single source (config.ApplyDefaults) — the
	// plugin-merged settings parse cannot use config.Load directly.
	config.ApplyDefaults(&cfg, cfgPath)

	logCleanup, err := server.SetupLog(cfg.Log)
	if err != nil {
		log.Printf("WARNING: log setup failed, using defaults: %v", err)
	}
	if logCleanup != nil {
		defer logCleanup()
	}
	if quiet {
		// SetupLog reinstalled the default slog handler — re-apply quiet.
		applyQuiet()
	}

	// 6. Build cobra tree.
	rootCmd.AddCommand(buildServeCmd(cfg))
	rootCmd.AddCommand(buildRunCmd(cfg))
	rootCmd.AddCommand(keyringCmd)

	// 7. Wrap every command's RunE to notify observers on entry/exit.
	// Treat context cancellation as normal shutdown — do not report
	// it as an error to observers.
	wrapCmd(rootCmd, func(cmd *cobra.Command) {
		hub.OnCommandStart(ctx, cmd.CommandPath())
	}, func(cmd *cobra.Command, err error) {
		if err == context.Canceled {
			err = nil
		}
		hub.OnCommandEnd(ctx, cmd.CommandPath(), err)
	})

	// 8. Notify observers: startup + defer shutdown.
	hub.OnStartup(ctx)
	defer hub.OnShutdown(context.Background())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// wrapCmd recursively wraps every RunE on a cobra Command tree so that
// beforeFn runs before execution and afterFn runs after with the error
// (nil on success). This ensures observer callbacks see errors from
// all commands.
func wrapCmd(c *cobra.Command, beforeFn func(*cobra.Command), afterFn func(*cobra.Command, error)) {
	for _, sub := range c.Commands() {
		wrapCmd(sub, beforeFn, afterFn)
	}
	if c.RunE != nil {
		orig := c.RunE
		c.RunE = func(cmd *cobra.Command, args []string) error {
			beforeFn(cmd)
			err := orig(cmd, args)
			afterFn(cmd, err)
			return err
		}
	}
}

// registerCommands recursively builds a cobra Command tree from plugin
// CommandDefs and registers them under parent. Group nodes (with Children)
// get sub-commands but no RunE; leaf nodes dispatch to the WASM plugin.
func registerCommands(parent *cobra.Command, cmds []cliwasm.CommandDef) {
	for _, cd := range cmds {
		cmd := &cobra.Command{
			Use: cd.Use, Short: cd.Short, Long: cd.Long,
			Aliases: cd.Aliases, Example: cd.Example,
			Args: parseArgRule(cd.Args),
		}

		if len(cd.Children) > 0 {
			registerCommands(cmd, cd.Children)
			parent.AddCommand(cmd)
			continue
		}

		// Leaf command — register flags and dispatch to WASM.
		name := cd.Name
		for _, f := range cd.Flags {
			switch f.Kind {
			case "bool":
				d, _ := strconv.ParseBool(f.DefaultValue)
				cmd.Flags().BoolP(f.Name, f.Short, d, f.Description)
			case "int":
				d, _ := strconv.Atoi(f.DefaultValue)
				cmd.Flags().IntP(f.Name, f.Short, d, f.Description)
			default:
				cmd.Flags().StringP(f.Name, f.Short, f.DefaultValue, f.Description)
			}
		}

		cmd.RunE = func(c *cobra.Command, args []string) error {
			flags := make(map[string]any)
			for _, f := range cd.Flags {
				switch f.Kind {
				case "bool":
					flags[f.Name], _ = c.Flags().GetBool(f.Name)
				case "int":
					flags[f.Name], _ = c.Flags().GetInt(f.Name)
				default:
					flags[f.Name], _ = c.Flags().GetString(f.Name)
				}
			}
			input, _ := json.Marshal(cliwasm.CommandInput{Args: args, Flags: flags})
			out, err := cliwasm.RunCommand(cmd.Context(), name, string(input))
			if err != nil {
				return err
			}
			fmt.Fprint(os.Stderr, out)
			return nil
		}
		parent.AddCommand(cmd)
		log.Printf("plugin: registered command %q", name)
	}
}

func parseArgRule(rule string) cobra.PositionalArgs {
	switch {
	case strings.HasPrefix(rule, "exact="):
		if n, err := strconv.Atoi(strings.TrimPrefix(rule, "exact=")); err == nil {
			return cobra.ExactArgs(n)
		}
	case strings.HasPrefix(rule, "min="):
		if n, err := strconv.Atoi(strings.TrimPrefix(rule, "min=")); err == nil {
			return cobra.MinimumNArgs(n)
		}
	case strings.HasPrefix(rule, "max="):
		if n, err := strconv.Atoi(strings.TrimPrefix(rule, "max=")); err == nil {
			return cobra.MaximumNArgs(n)
		}
	case strings.HasPrefix(rule, "range="):
		parts := strings.SplitN(strings.TrimPrefix(rule, "range="), ",", 2)
		if len(parts) == 2 {
			min, _ := strconv.Atoi(parts[0])
			max, _ := strconv.Atoi(parts[1])
			return cobra.RangeArgs(min, max)
		}
	}
	return cobra.ArbitraryArgs
}

var rootCmd = &cobra.Command{
	Use:   "hwcloud",
	Short: "openagent CLI",
}

// ── serve ──

func buildServeCmd(cfg config.Config) *cobra.Command {
	// Capabilities come from settings.json; flags override per capability.
	caps := cfg.Capabilities
	cmd := &cobra.Command{
		Use:          "serve",
		Short:        "Start the server (REST by default, or --acp for ACP)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Process-level soft memory limit: beyond this the Go GC
			// runs aggressively instead of letting the heap grow toward
			// the host's physical memory. Soft — it never rejects an
			// allocation; it only pressures the GC. WASM plugin memory
			// is long-lived (never collected), so the real per-plugin
			// fence is the 512 MiB linear-memory cap in the plugin
			// loaders; this bounds the rest of the process.
			debug.SetMemoryLimit(512 << 20)
			parseCapabilities(cmd, &caps)
			isACP, _ := cmd.Flags().GetBool("acp")
			channelFlag, _ := cmd.Flags().GetString("channel")
			p, _ := cmd.Flags().GetInt("port")
			if p > 0 {
				cfg.Server.Port = p
			}
			if sandboxEnabled, _ := cmd.Flags().GetBool("sandbox"); sandboxEnabled {
				cfg.Sandbox.Enabled = true
			}

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

			// --channel <name> marks that channel as explicitly requested:
			// auto-connect at startup with fail-fast when it cannot start.
			// All settings credentials are KEPT (settings is the credential
			// store; other channels stay configured-but-disconnected and
			// the frontend can connect them via POST /connect).
			if channelFlag != "" {
				switch channelFlag {
				case "feishu":
					if cfg.Channels.Feishu == nil {
						cfg.Channels.Feishu = &config.FeishuConfig{}
					}
					cfg.Channels.Feishu.Explicit = true
				case "wechat":
					if cfg.Channels.Wechat == nil {
						cfg.Channels.Wechat = &config.WechatConfig{}
					}
					cfg.Channels.Wechat.Explicit = true
				case "wecom":
					if cfg.Channels.Wecom == nil {
						cfg.Channels.Wecom = &config.WecomConfig{}
					}
					cfg.Channels.Wecom.Explicit = true
				default:
					return fmt.Errorf("unknown channel %q (supported: feishu, wechat, wecom)", channelFlag)
				}
			}
			defer cancel()
			if isACP {
				return server.RunACP(ctx, &cfg, caps)
			}
			return server.RunREST(ctx, &cfg, caps)
		},
	}
	cmd.Flags().Bool("acp", false, "ACP mode over stdio")
	cmd.Flags().String("channel", "", "Enable IM channel (\"feishu\", \"wechat\", or \"wecom\")")
	cmd.Flags().Int("port", 0, "REST port (overrides settings)")
	cmd.Flags().Bool("sandbox", false, "Enable OS-native sandbox (bwrap/seatbelt) for shell commands")
	addCapabilityFlags(cmd)
	return cmd
}

func addCapabilityFlags(cmd *cobra.Command) {
	for _, f := range []struct{ name, def, desc string }{
		{"memory", "on", "Memory backend"},
		{"summarizer", "on", "Conversation summarizer"},
		{"skills", "on", "Skill loader"},
		{"mcp", "on", "MCP tool servers"},
		{"guard", "off", "LLM content guard"},
		{"approver", "off", "Human-in-the-Loop tool approval"},
		{"embedder", "on", "Built-in BGE embedding (semantic recall)"},
	} {
		cmd.Flags().String(f.name, f.def, f.desc+` ("on" or "off")`)
	}
}

func parseCapabilities(cmd *cobra.Command, caps *config.Capabilities) {
	set := func(name string, field **bool) {
		if !cmd.Flags().Changed(name) {
			return // not explicitly set — Capabilities.on() uses the default
		}
		v, _ := cmd.Flags().GetString(name)
		switch strings.ToLower(v) {
		case "on":
			b := true
			*field = &b
		case "off":
			b := false
			*field = &b
		default:
			log.Printf("WARNING: --%s=%q is invalid (expected on/off), using default", name, v)
		}
	}
	set("memory", &caps.Memory)
	set("summarizer", &caps.Summarizer)
	set("skills", &caps.Skills)
	set("mcp", &caps.MCP)
	set("guard", &caps.Guard)
	set("approver", &caps.Approver)
	set("embedder", &caps.Embedder)
}

// ── run ──

func buildRunCmd(cfg config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <message>",
		Short: "Send a message and stream the response",
		Long: `Send a message to the AI Agent and stream the response to stdout in real time.

Wrap your message in quotes when it contains spaces:
  hwcloud run "analyze main.go"`,
		Example: `  hwcloud run "Hello, introduce yourself briefly"
  hwcloud run "analyze cmd/cli/main.go and summarize"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.RunCLI(cmd.Context(), &cfg, args[0])
		},
	}
	return cmd
}

// ── keyring ──

// keyringOrFail returns the system keyring or exits with a clear message
// when no persistent backend is available. Used by `keyring set` /
// `keyring delete` — silently storing in MemStore would be data loss
// (user sees exit 0 but secrets evaporate on process exit).
func keyringOrFail() plugin.Keyring {
	sysKr, err := keyring.Open()
	if err != nil {
		log.Fatalf("no keyring backend available: install `dbus-x11` (Linux desktop) "+
			"or run the container with `--cap-add=keyutils` for kernel-keyring "+
			"fallback; original error: %v", err)
	}
	return sysKr
}

// isKeyringCmd reports whether the first non-flag argument is "keyring",
// so plugin loading can be skipped for keyring subcommands.
func isKeyringCmd(args []string) bool {
	for _, a := range args[1:] {
		if strings.HasPrefix(a, "-") {
			continue
		}
		return a == "keyring"
	}
	return false
}

// applyQuiet silences all log output (slog and the standard log package)
// by discarding every message. Note this must NOT write to os.Stderr:
// the ACP protocol uses stderr as a control pipe and any log output
// there fills the pipe buffer and blocks the process.
func applyQuiet() {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	log.SetOutput(io.Discard)
}

// loadPlugins instantiates the WASM runtime, loads every .wasm under
// pluginPaths, and routes capabilities (settings merge, command
// registration, observer wiring). Returns the possibly merged settings
// and a cleanup func that closes the WASM runtime.
func loadPlugins(ctx context.Context, pluginPaths []string, settings []byte, hub *cliwasm.ObserverHub) ([]byte, func()) {
	wasmRuntime, err := cliwasm.NewRuntime(ctx)
	if err != nil {
		log.Fatalf("wasm runtime: %v", err)
	}

	mgr := plugin.NewManager(pluginPaths)
	for _, p := range pluginPaths {
		files, _ := mgr.ResolveWasmFiles(p)
		for _, f := range files {
			wasmBytes, err := os.ReadFile(f)
			if err != nil {
				log.Printf("plugin: read %s: %v", f, err)
				continue
			}
			mod, meta, err := wasmRuntime.Instantiate(ctx, wasmBytes, f)
			if err != nil {
				log.Printf("plugin: load %s: %v", f, err)
				continue
			}

			// The plugins dir is shared with agent plugins (agent:tools,
			// agent:observers) — skip them quietly, they are loaded by the
			// serve-side agent loader. Only cli:* types are ours.
			if !isCLIPlugin(meta.Type) {
				slog.Debug("plugin: skipping non-cli plugin", "name", meta.Name, "type", meta.Type)
				continue
			}

			log.Printf("plugin: loaded %s (%s) type=%s", meta.Name, meta.Description, meta.Type)

			if meta.Is("settings") {
				merged, err := mod.CallInit(ctx, settings)
				if err != nil {
					log.Printf("plugin %s init: %v", meta.Name, err)
					continue
				}
				settings = merged
			}

			if meta.Is("commands") {
				cmds, err := mod.ReadCommands(ctx)
				if err != nil {
					log.Printf("plugin %s commands: %v", meta.Name, err)
					continue
				}
				registerCommands(rootCmd, cmds)
			}

			if meta.Is("http") {
				if err := cliwasm.RegisterHTTPRoutes(mod, meta); err != nil {
					log.Printf("plugin %s http: %v", meta.Name, err)
					continue
				}
				log.Printf("plugin: registered %d http route(s) for %s", len(meta.Routes), meta.Name)
			}

			if meta.Is("observers") {
				hub.Add(mod)
			}
		}
	}
	return settings, func() { wasmRuntime.Close(ctx) }
}

var keyringCmd = &cobra.Command{Use: "keyring", Short: "Manage credentials in the system keyring"}
var keyringSetCmd = &cobra.Command{
	Use: "set <key> <value>", Short: "Store a credential", Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return keyringOrFail().Set("openagent", args[0], args[1])
	},
}
var keyringGetCmd = &cobra.Command{
	Use: "get <key>", Short: "Read a credential", Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Same backend as set/delete — reading from a silent in-memory
		// store would always miss what set wrote to the real keyring.
		v, err := keyringOrFail().Get("openagent", args[0])
		if err != nil {
			return fmt.Errorf("keyring get: %w", err)
		}
		if v == "" {
			fmt.Fprintln(os.Stderr, "(not found)")
		} else {
			fmt.Fprintln(os.Stderr, v)
		}
		return nil
	},
}
var keyringDeleteCmd = &cobra.Command{
	Use: "delete <key>", Short: "Remove a credential", Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return keyringOrFail().Delete("openagent", args[0])
	},
}

func init() {
	keyringCmd.AddCommand(keyringSetCmd, keyringGetCmd, keyringDeleteCmd)

	// Wire the build version into cobra's built-in --version path. We add
	// the flag ourselves so it gets the -v shorthand; cobra sees the flag
	// already exists and skips its default (shorthand-less) one, while
	// still short-circuiting to print the version template.
	rootCmd.Version = version.Version
	rootCmd.Flags().BoolP("version", "v", false, "show version")
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// -q/--quiet silences all log output (plugin loading logs included).
	// Persistent so it works on every subcommand; applied manually in
	// main() because plugin loading happens before cobra's Execute.
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "suppress all log output")
}

// isCLIPlugin reports whether a plugin's comma-separated type list
// contains any cli:* entry (this loader's namespace). Agent plugins
// (agent:tools, agent:observers) live in the same directory and are
// loaded by the serve-side agent loader.
func isCLIPlugin(pluginType string) bool {
	for _, part := range strings.Split(pluginType, ",") {
		if strings.HasPrefix(strings.TrimSpace(part), "cli:") {
			return true
		}
	}
	return false
}
