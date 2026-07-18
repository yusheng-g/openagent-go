package wasm

import (
	"context"
	"encoding/json"
	"fmt"

	openagent "github.com/yusheng-g/openagent-go"
)

// wasmObserver wraps a WASM observer plugin. It stores the filter (stage name + phase)
// and calls the guest's run() when events match.
type wasmObserver struct {
	mod  *module
	meta PluginMeta
	name string // human label for logging
}

// matches returns true if this plugin should fire for the given event.
func (s *wasmObserver) matches(event openagent.StageEvent) bool {
	if s.meta.Stage != "" && s.meta.Stage != PhaseAll && s.meta.Stage != event.Name {
		return false
	}
	if s.meta.Phase == "" || s.meta.Phase == PhaseAll {
		return true
	}
	return s.meta.Phase == event.Phase
}

// invoke calls the guest's run() and returns the parsed stage output.
func (s *wasmObserver) invoke(ctx context.Context, event openagent.StageEvent) (*StageOutput, error) {
	errStr := ""
	if event.Err != nil {
		errStr = event.Err.Error()
	}

	input := StageInput{
		Name:   event.Name,
		Phase:  event.Phase,
		Detail: event.Detail,
		Error:  errStr,
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("wasm stage %q: marshal: %w", s.meta.Name, err)
	}

	outputJSON, err := s.mod.invoke(ctx, "run", inputJSON)
	if err != nil {
		return nil, fmt.Errorf("wasm stage %q: %w", s.meta.Name, err)
	}

	var out StageOutput
	if err := json.Unmarshal(outputJSON, &out); err != nil {
		return nil, fmt.Errorf("wasm stage %q: parse output: %w", s.meta.Name, err)
	}

	if out.Action == ActionAbort {
		return &out, fmt.Errorf("wasm stage %q aborted: %s", s.meta.Name, out.Reason)
	}
	return &out, nil
}
