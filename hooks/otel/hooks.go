// Package otel implements openagent.RunHooks with OpenTelemetry tracing.
//
// Usage:
//
//	tracer := otel.Tracer("openagent")
//	hooks := otelhooks.New(tracer)
//	agent := openagent.NewAgent("bot", openagent.WithRunHooks(hooks))
package otel

import (
	"context"
	"encoding/json"
	"fmt"

	openagent "github.com/yusheng-g/openagent-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Hooks implements openagent.RunHooks via OpenTelemetry spans.
type Hooks struct {
	tracer trace.Tracer
}

// New creates a Hooks that creates spans with the given tracer.
func New(tracer trace.Tracer) *Hooks {
	return &Hooks{tracer: tracer}
}

func (h *Hooks) OnAgentStart(ctx context.Context, req openagent.ChatCompletionRequest) (any, error) {
	ctx, span := h.tracer.Start(ctx, "agent.run",
		trace.WithAttributes(
			attribute.String("agent.model", req.Model),
			attribute.Int("agent.messages", len(req.Messages)),
			attribute.Int("agent.tools", len(req.Tools)),
		),
	)
	// Start a root span and defer End to OnAgentEnd so the duration
	// covers the entire run loop (including all tool calls).
	return span, nil
}

func (h *Hooks) OnAgentEnd(ctx context.Context, req openagent.ChatCompletionRequest, resp *openagent.ChatCompletionResponse, runErr error, startState any) {
	span, _ := startState.(trace.Span)
	if span == nil {
		return
	}
	defer span.End()

	if resp != nil {
		span.SetAttributes(
			attribute.Int("agent.prompt_tokens", resp.Usage.PromptTokens),
			attribute.Int("agent.completion_tokens", resp.Usage.CompletionTokens),
			attribute.Int("agent.total_tokens", resp.Usage.TotalTokens),
		)
	}
	if runErr != nil {
		span.SetStatus(codes.Error, runErr.Error())
		span.RecordError(runErr)
	}
}

func (h *Hooks) OnToolStart(ctx context.Context, tool openagent.FunctionDefinition, args json.RawMessage) (any, error) {
	ctx, span := h.tracer.Start(ctx, fmt.Sprintf("tool.%s", tool.Name),
		trace.WithAttributes(
			attribute.String("tool.name", tool.Name),
			attribute.String("tool.args", string(args)),
		),
	)
	return span, nil
}

func (h *Hooks) OnToolEnd(ctx context.Context, tool openagent.FunctionDefinition, args json.RawMessage, result *string, err *error, startState any) {
	span, _ := startState.(trace.Span)
	if span == nil {
		return
	}
	defer span.End()

	if err != nil && *err != nil {
		span.SetStatus(codes.Error, (*err).Error())
		span.RecordError(*err)
	}
	if result != nil {
		span.SetAttributes(attribute.Int("tool.result_len", len(*result)))
	}
}

var _ openagent.RunHooks = (*Hooks)(nil)
