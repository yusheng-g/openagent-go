package wasm

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero"

	"github.com/yusheng-g/openagent-go/plugin/wasmhost"
)

// Runtime wraps a wazero runtime with the host module pre-registered.
// All CLI plugins (.wasm files) share this runtime.
type Runtime struct {
	rt   wazero.Runtime
	host *wasmhost.HostAPI
}

// NewRuntime creates a wazero runtime and registers the "host" module.
func NewRuntime(ctx context.Context, kr wasmhost.Keyring, httpc wasmhost.HTTPClient, logger wasmhost.Logger) (*Runtime, error) {
	rt := wazero.NewRuntime(ctx)

	h := &wasmhost.HostAPI{Keyring: kr, HTTP: httpc, Logger: logger}
	if err := h.RegisterHostModule(ctx, rt); err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("register host module: %w", err)
	}

	return &Runtime{rt: rt, host: h}, nil
}

// Close releases the wazero runtime.
func (r *Runtime) Close(ctx context.Context) error {
	return r.rt.Close(ctx)
}
