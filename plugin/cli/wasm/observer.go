package wasm

import (
	"context"
	"log"
	"sync"
)

// ObserverHub dispatches lifecycle events to observer plugins.
type ObserverHub struct {
	mu      sync.Mutex
	modules []*Module
}

func (h *ObserverHub) Add(m *Module) {
	h.mu.Lock()
	h.modules = append(h.modules, m)
	h.mu.Unlock()
}

func (h *ObserverHub) OnStartup(ctx context.Context) {
	h.broadcast(ctx, "on_startup")
}

func (h *ObserverHub) OnShutdown(ctx context.Context) {
	h.broadcast(ctx, "on_shutdown")
}

// OnCommandStart fires before a cobra command executes. Passes the command name.
func (h *ObserverHub) OnCommandStart(ctx context.Context, cmdName string) {
	h.broadcastStr(ctx, "on_command_start", cmdName)
}

// OnCommandEnd fires after a cobra command finishes. Passes the command
// name and the error (nil if success).
func (h *ObserverHub) OnCommandEnd(ctx context.Context, cmdName string, cmdErr error) {
	payload := cmdName
	if cmdErr != nil {
		payload = cmdName + " error=" + cmdErr.Error()
	}
	h.broadcastStr(ctx, "on_command_end", payload)
}

func (h *ObserverHub) broadcast(ctx context.Context, name string) {
	h.mu.Lock()
	mods := make([]*Module, len(h.modules))
	copy(mods, h.modules)
	h.mu.Unlock()
	for _, m := range mods {
		fn := m.Mod.ExportedFunction(name)
		if fn == nil { continue }
		if _, err := fn.Call(ctx); err != nil {
			log.Printf("observer %s: %v", name, err)
		}
	}
}

func (h *ObserverHub) broadcastStr(ctx context.Context, name string, arg string) {
	h.mu.Lock()
	mods := make([]*Module, len(h.modules))
	copy(mods, h.modules)
	h.mu.Unlock()
	for _, m := range mods {
		fn := m.Mod.ExportedFunction(name)
		if fn == nil { continue }
		allocFn := m.Mod.ExportedFunction("alloc")
		if allocFn == nil { continue }
		b := []byte(arg)
		res, err := allocFn.Call(ctx, uint64(len(b)))
		if err != nil || len(res) == 0 { continue }
		ptr := uint32(res[0])
		m.Mod.Memory().Write(ptr, b)
		if _, err := fn.Call(ctx, uint64(ptr), uint64(len(b))); err != nil {
			log.Printf("observer %s: %v", name, err)
		}
	}
}
