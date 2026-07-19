// openagent-cli — openagent-go CLI.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
	"github.com/yusheng-g/openagent-go/cmd/cli/keyring"
	"github.com/yusheng-g/openagent-go/plugin/cli"
	cliwasm "github.com/yusheng-g/openagent-go/plugin/cli/wasm"
	"github.com/yusheng-g/openagent-go/cmd/cli/server"
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
	var preCfg config.Config
	json.Unmarshal(raw, &preCfg)
	if len(preCfg.Plugins) > 0 {
		pluginPaths = preCfg.Plugins
	}

	// 3. Keyring + runtime.
	kr := openKeyring()
	httpClient := &defaultHTTPClient{client: http.DefaultClient}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	wasmRuntime, err := cliwasm.NewRuntime(ctx, kr, httpClient, &stdLogger{})
	if err != nil {
		log.Fatalf("wasm runtime: %v", err)
	}
	defer wasmRuntime.Close(ctx)

	// 4. Load every .wasm and route capabilities.
	settings := raw
	if len(settings) == 0 {
		settings = []byte("{}")
	}
	mgr := plugin.NewManager(pluginPaths)
	hub := &cliwasm.ObserverHub{}
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
				for _, cd := range cmds {
					name := cd.Name
					rootCmd.AddCommand(&cobra.Command{
						Use: cd.Use, Short: cd.Short, Long: cd.Long,
						RunE: func(cmd *cobra.Command, args []string) error {
							argsJSON, _ := json.Marshal(args)
							out, err := cliwasm.RunCommand(ctx, name, string(argsJSON))
							if err != nil {
								return err
							}
							fmt.Print(out)
							return nil
						},
					})
					log.Printf("plugin: registered command %q", name)
				}
			}

			if meta.Is("observers") {
				hub.Add(mod)
			}
		}
	}

	// 5. Parse final merged config.
	var cfg config.Config
	if err := json.Unmarshal(settings, &cfg); err != nil {
		log.Fatalf("parse merged settings: %v", err)
	}
	for k, v := range cfg.Env {
		os.Setenv(k, v)
	}
	pretty, _ := json.MarshalIndent(&cfg, "", "  ")
	log.Printf("Merged settings:\n%s", string(pretty))

	// 6. Build cobra tree.
	rootCmd.AddCommand(buildServeCmd(cfg))
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

// ── root ──

var rootCmd = &cobra.Command{
	Use:   "openagent-cli",
	Short: "openagent CLI",
}

// ── serve ──

func buildServeCmd(cfg config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "serve",
		Short:        "Start the server (REST by default, or --acp for ACP)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			isACP, _ := cmd.Flags().GetBool("acp")
			p, _ := cmd.Flags().GetInt("port")
			if p > 0 {
				cfg.Server.Port = p
			}
			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()
			if isACP {
				return server.RunACP(ctx, &cfg)
			}
			return server.RunREST(ctx, &cfg)
		},
	}
	cmd.Flags().Bool("acp", false, "ACP mode over stdio")
	cmd.Flags().Int("port", 0, "REST port (overrides settings)")
	return cmd
}

// ── keyring ──

// openKeyring returns the system keyring, falling back to an in-memory
// store with a warning when the system keychain is unavailable. Intended
// for read-only / fall-through callers (e.g. `serve`) that can still
// operate without persisted secrets.
func openKeyring() plugin.Keyring {
	sysKr, err := keyring.Open()
	if err != nil {
		log.Printf("WARNING: keyring unavailable, using in-memory fallback (secrets will not persist): %v", err)
		return keyring.NewMemStore()
	}
	return sysKr
}

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
		kr := openKeyring()
		v, err := kr.Get("openagent", args[0])
		if err != nil {
			return fmt.Errorf("keyring get: %w", err)
		}
		if v == "" { fmt.Println("(not found)") } else { fmt.Println(v) }
		return nil
	},
}
var keyringDeleteCmd = &cobra.Command{
	Use: "delete <key>", Short: "Remove a credential", Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return keyringOrFail().Delete("openagent", args[0])
	},
}

func init() { keyringCmd.AddCommand(keyringSetCmd, keyringGetCmd, keyringDeleteCmd) }

// ── HTTP / logger ──

type defaultHTTPClient struct{ client *http.Client }

func (c *defaultHTTPClient) Do(method, url string, headers map[string]string, body []byte) (int, []byte, error) {
	req, _ := http.NewRequest(method, url, bytes.NewReader(body))
	for k, v := range headers { req.Header.Set(k, v) }
	resp, err := c.client.Do(req)
	if err != nil { return 0, nil, err }
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody, nil
}

type stdLogger struct{}

func (l *stdLogger) Info(msg string)  { log.Printf("[plugin] %s", msg) }
func (l *stdLogger) Warn(msg string)  { log.Printf("[plugin] WARN: %s", msg) }
func (l *stdLogger) Error(msg string) { log.Printf("[plugin] ERROR: %s", msg) }
