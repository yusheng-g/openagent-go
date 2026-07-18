package wasm

import (
	"context"
	"encoding/json"
	"fmt"

	openagent "github.com/yusheng-g/openagent-go"
)

// wasmTool adapts a WASM tool plugin to openagent.Tool.
type wasmTool struct {
	mod  *module
	meta PluginMeta
}

var _ openagent.Tool = (*wasmTool)(nil)

func (t *wasmTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        t.meta.Name,
		Description: t.meta.Description,
		Parameters:  t.meta.Parameters,
	}
}

func (t *wasmTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	input, err := json.Marshal(ToolInput{Args: args})
	if err != nil {
		return "", fmt.Errorf("wasm tool %q: marshal input: %w", t.meta.Name, err)
	}

	output, err := t.mod.invoke(ctx, "execute", input)
	if err != nil {
		return "", fmt.Errorf("wasm tool %q: %w", t.meta.Name, err)
	}

	var out ToolOutput
	if err := json.Unmarshal(output, &out); err != nil {
		return "", fmt.Errorf("wasm tool %q: parse output: %w", t.meta.Name, err)
	}

	if out.Error != "" {
		return "", fmt.Errorf("wasm tool %q: %s", t.meta.Name, out.Error)
	}
	return out.Result, nil
}
