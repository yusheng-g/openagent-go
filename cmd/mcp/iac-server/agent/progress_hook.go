package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	openagent "github.com/yusheng-g/openagent-go"
)

// progressHook is a RunHooks implementation that forwards tool-call events
// to the MCP client as progress notifications via the ProgressFunc stored in
// ctx.
//
// Each OnToolStart increments a counter and sends a progress notification
// with the tool name, so the client sees live activity during the multi-turn
// agent.Run() loop (which can take minutes). This complements the coarse
// step-level progress calls in the Planner methods.
//
// The hook is stateless aside from the counter — it reads ProgressFunc from
// ctx on each call, so it works across retries and multiple agents.
type progressHook struct {
	toolCount atomic.Int64
}

// newProgressHook returns a progressHook. A fresh instance should be used
// per agent.Run() call so the tool counter resets.
func newProgressHook() *progressHook {
	return &progressHook{}
}

func (h *progressHook) OnAgentStart(ctx context.Context, req openagent.ChatCompletionRequest) (any, error) {
	return nil, nil
}

func (h *progressHook) OnAgentEnd(ctx context.Context, req openagent.ChatCompletionRequest, resp *openagent.ChatCompletionResponse, runErr error, startState any) {
}

func (h *progressHook) OnToolStart(ctx context.Context, tool openagent.FunctionDefinition, args json.RawMessage) (any, error) {
	if progress := openagent.ProgressFromContext(ctx); progress != nil {
		n := h.toolCount.Add(1)
		msg := tool.Name
		// Include args if non-empty and not just "{}".
		if len(args) > 0 && string(args) != "{}" {
			argStr := string(args)
			if len(argStr) > 200 {
				argStr = argStr[:200] + "..."
			}
			msg = fmt.Sprintf("%s(%s)", tool.Name, argStr)
		}
		progress(msg, float64(n), 0)
	}
	return nil, nil
}

func (h *progressHook) OnToolEnd(ctx context.Context, tool openagent.FunctionDefinition, args json.RawMessage, result *string, err *error, startState any) {
}

var _ openagent.RunHooks = (*progressHook)(nil)
