package wasm

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero"

	"github.com/yusheng-g/openagent-go/cmd/cli/plugin"
)

// Runtime wraps a wazero runtime with the host module pre-registered.
// All CLI plugins (.wasm files) share this runtime.
type Runtime struct {
	rt   wazero.Runtime
	host *hostAPI
}

// NewRuntime creates a wazero runtime and registers the "host" module.
// CLI plugins are compiled with tinygo -target=wasi. WASI is instantiated
// so the plugin runtime (stdout/stderr) doesn't panic.
func NewRuntime(ctx context.Context, kr plugin.Keyring, httpc plugin.HTTPClient, logger plugin.Logger) (*Runtime, error) {
	rt := wazero.NewRuntime(ctx)


	h := &hostAPI{
		keyring: kr,
		http:    httpc,
		logger:  logger,
	}
	if err := h.registerModule(ctx, rt); err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("register host module: %w", err)
	}

	return &Runtime{rt: rt, host: h}, nil
}

// Close releases the wazero runtime.
func (r *Runtime) Close(ctx context.Context) error {
	return r.rt.Close(ctx)
}
