// Package slog implements openagent.RunHooks with log/slog.
// Zero external dependencies — uses only the standard library.
//
// Usage:
//
//	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
//	hooks := sloghooks.New(logger)
//	agent := openagent.NewAgent("bot", openagent.WithRunHooks(hooks))
package slog

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
)

// Hooks implements openagent.RunHooks via log/slog.
type Hooks struct {
	logger *slog.Logger
}

// New creates a Hooks that logs to the given slog.Logger.
func New(logger *slog.Logger) *Hooks {
	return &Hooks{logger: logger}
}

func (h *Hooks) OnAgentStart(ctx context.Context, req openagent.ChatCompletionRequest) (any, error) {
	h.logger.InfoContext(ctx, "agent start",
		"model", req.Model,
		"messages", len(req.Messages),
		"tools", len(req.Tools),
	)
	return time.Now(), nil
}

func (h *Hooks) OnAgentEnd(ctx context.Context, req openagent.ChatCompletionRequest, resp *openagent.ChatCompletionResponse, runErr error, startState any) {
	t0, _ := startState.(time.Time)
	elapsed := time.Since(t0)

	attrs := []slog.Attr{
		slog.String("model", req.Model),
		slog.Duration("elapsed", elapsed),
	}
	if resp != nil {
		attrs = append(attrs,
			slog.Int("prompt_tokens", resp.Usage.PromptTokens),
			slog.Int("completion_tokens", resp.Usage.CompletionTokens),
			slog.Int("total_tokens", resp.Usage.TotalTokens),
		)
	}
	if runErr != nil {
		attrs = append(attrs, slog.String("error", runErr.Error()))
		h.logger.LogAttrs(ctx, slog.LevelError, "agent end", attrs...)
	} else {
		h.logger.LogAttrs(ctx, slog.LevelInfo, "agent end", attrs...)
	}
}

func (h *Hooks) OnToolStart(ctx context.Context, tool openagent.FunctionDefinition, args json.RawMessage) (any, error) {
	return time.Now(), nil
}

func (h *Hooks) OnToolEnd(ctx context.Context, tool openagent.FunctionDefinition, args json.RawMessage, result *string, err *error, startState any) {
	t0, _ := startState.(time.Time)
	elapsed := time.Since(t0)

	attrs := []slog.Attr{
		slog.String("tool", tool.Name),
		slog.Duration("elapsed", elapsed),
	}
	if err != nil && *err != nil {
		attrs = append(attrs, slog.String("error", (*err).Error()))
		h.logger.LogAttrs(ctx, slog.LevelError, "tool end", attrs...)
	} else {
		attrs = append(attrs, slog.Int("result_len", len(*result)))
		h.logger.LogAttrs(ctx, slog.LevelDebug, "tool end", attrs...)
	}
}

var _ openagent.RunHooks = (*Hooks)(nil)
