package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
)

// Planner generates a PlanDef from a goal and available agents.
type Planner interface {
	// Plan analyses the goal and returns a DAG of steps.
	// agents lists the available agents with their descriptions.
	// history is optional conversation context (may be nil/empty).
	Plan(ctx context.Context, goal string, agents []openagent.AgentInfo, history []openagent.Message) (*PlanDef, error)
}

// ── LLMPlanner ──

// LLMPlanner uses an LLM to decompose a goal into a DAG of steps.
// Generated plans are validated; if validation fails, the LLM is asked
// to correct the plan (up to maxRetries times).
type LLMPlanner struct {
	model      openagent.Model
	maxRetries int
}

// NewLLMPlanner creates a Planner backed by a Model.
func NewLLMPlanner(model openagent.Model) *LLMPlanner {
	return &LLMPlanner{model: model, maxRetries: 3}
}

// WithMaxRetries sets the maximum number of correction rounds on validation failure.
func (p *LLMPlanner) WithMaxRetries(n int) *LLMPlanner {
	p.maxRetries = n
	return p
}

// Plan implements [Planner].
func (p *LLMPlanner) Plan(ctx context.Context, goal string, agents []openagent.AgentInfo, history []openagent.Message) (*PlanDef, error) {
	return p.planWithModel(ctx, goal, agents, history, false, nil)
}

// PlanStream generates a PlanDef with streaming text chunks emitted via onChunk.
// Each token/reasoning delta from the LLM is passed to onChunk as it arrives.
// The final PlanDef is returned after the full response is received and validated.
func (p *LLMPlanner) PlanStream(ctx context.Context, goal string, agents []openagent.AgentInfo, history []openagent.Message, onChunk func(string)) (*PlanDef, error) {
	return p.planWithModel(ctx, goal, agents, history, true, onChunk)
}

// planWithModel is the shared implementation for Plan and PlanStream.
// When streaming is true, ChatCompletionStream is used and deltas are
// emitted via onChunk. Otherwise a single ChatCompletion call is made.
func (p *LLMPlanner) planWithModel(ctx context.Context, goal string, agents []openagent.AgentInfo, history []openagent.Message, streaming bool, onChunk func(string)) (*PlanDef, error) {
	agentNames := make(map[string]bool)
	for _, a := range agents {
		agentNames[a.Name] = true
	}

	prompt := buildPlannerPrompt(goal, agents, history)
	var lastErr error

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		messages := []openagent.Message{
			{Role: openagent.RoleSystem, Content: plannerSystemPrompt},
			{Role: openagent.RoleUser, Content: prompt},
		}

		if attempt > 0 && lastErr != nil {
			messages = append(messages, openagent.Message{
				Role:    openagent.RoleUser,
				Content: fmt.Sprintf("Your previous plan was invalid. Please fix these issues:\n%s\n\nGenerate a corrected plan.", lastErr.Error()),
			})
		}

		var fullText string
		var err error

		if streaming {
			fullText, err = p.streamCall(ctx, messages, onChunk)
		} else {
			fullText, err = p.syncCall(ctx, messages)
		}
		if err != nil {
			return nil, fmt.Errorf("planner: model call failed: %w", err)
		}

		def, err := parsePlanJSON(fullText)
		if err != nil {
			lastErr = err
			continue
		}

		if err := Validate(def, agentNames); err != nil {
			lastErr = err
			continue
		}

		return def, nil
	}

	return nil, fmt.Errorf("planner: failed to generate a valid plan after %d attempts: %w", p.maxRetries+1, lastErr)
}

func (p *LLMPlanner) syncCall(ctx context.Context, messages []openagent.Message) (string, error) {
	resp, err := p.model.ChatCompletion(ctx, openagent.ChatCompletionRequest{
		Messages:  messages,
		MaxTokens: 2048,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("model returned no choices")
	}
	return resp.Choices[0].Message.Content, nil
}

func (p *LLMPlanner) streamCall(ctx context.Context, messages []openagent.Message, onChunk func(string)) (string, error) {
	reader, err := p.model.ChatCompletionStream(ctx, openagent.ChatCompletionRequest{
		Messages:  messages,
		MaxTokens: 2048,
	})
	if err != nil {
		// Fall back to synchronous call if streaming is not supported.
		return p.syncCall(ctx, messages)
	}
	defer reader.Close()

	// ReasoningContent is the model's internal monologue (deepseek-r1, o1).
	// Display it to the user as "thinking" but do NOT include it in the text
	// we parse as JSON — the actual plan is in Content.
	var fullText strings.Builder
	for reader.Next() {
		chunk := reader.Current()
		for _, delta := range chunk.Choices {
			if delta.ReasoningContent != "" {
				onChunk(delta.ReasoningContent)
			}
			if delta.Content != "" {
				fullText.WriteString(delta.Content)
				onChunk(delta.Content)
			}
		}
	}
	if err := reader.Err(); err != nil {
		return "", err
	}
	return fullText.String(), nil
}

// ── Prompt ──

const plannerSystemPrompt = `You are an expert planner. Decompose a goal into a DAG of steps that maximizes parallel execution.

Output a JSON object:
{
  "goal": "the original goal",
  "steps": [
    {
      "id": "unique_descriptive_id",
      "agent": "agent_name",
      "task": "specific task for this step only",
      "depends_on": ["step_ids_this_must_wait_for"],
      "final": false
    }
  ]
}

## Rules

1. **Fork everything that can be parallel.** If two sub-tasks don't need each other's output, they go in separate branches — they WILL run concurrently.
2. **Use descriptive IDs.** Prefix with the branch: "design_api", "code_auth", "research_postgres".
3. **agent must be from the available agents list.** Never invent agent names.
4. **depends_on only lists steps this step waits for.** Empty array or omit for entry steps.
5. **final: true for the last answer-producing step(s).**

## Topology patterns (pick the one that fits the goal)

**Linear** — single task, one chain:
  A → B → C

**Fan-out** — shared entry, parallel branches:
       ┌─ B ─┐
  A ───┤     ├── (all independent, no join needed)
       └─ C ─┘

**Fan-in** — parallel research, single synthesis:
  A ∥ B ∥ C → D

**Fork-Join** — entry → parallel → join:
       ┌─ B ─┐
  A ───┤     ├── D
       └─ C ─┘
  D depends on BOTH B and C, combining their outputs.

**Multi-branch** — each independent sub-task gets its own full chain:
  A1→B1→C1 ∥ A2→B2→C2

Choose the topology that matches the goal's structure. If the goal mentions multiple independent items, use multi-branch or fork-join. If it asks for parallel analysis with a unified conclusion, use fan-in.

Reply with ONLY the JSON object. No markdown fences, no explanation.`

func buildPlannerPrompt(goal string, agents []openagent.AgentInfo, history []openagent.Message) string {
	var b strings.Builder

	b.WriteString("## Available Agents\n\n")
	for _, a := range agents {
		desc := a.Description
		if desc == "" {
			desc = "No description provided."
		}
		b.WriteString(fmt.Sprintf("- **%s**: %s\n", a.Name, desc))
	}

	b.WriteString("\n## Goal\n\n")
	b.WriteString(goal)

	if len(history) > 0 {
		b.WriteString("\n\n## Conversation History\n\n")
		for _, m := range history {
			b.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, truncateStr(m.Content, 500)))
		}
	}

	b.WriteString("\n\nGenerate the execution plan as JSON.")
	return b.String()
}

// ── JSON parsing ──

func parsePlanJSON(raw string) (*PlanDef, error) {
	// Strip markdown fences if present.
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		// Remove ```json or ``` and trailing ```
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		// Find closing fence
		if idx := strings.LastIndex(raw, "```"); idx >= 0 {
			raw = raw[:idx]
		}
		raw = strings.TrimSpace(raw)
	}

	var def PlanDef
	if err := json.Unmarshal([]byte(raw), &def); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w\nRaw response:\n%s", err, truncateStr(raw, 500))
	}

	if def.Goal == "" {
		return nil, fmt.Errorf("plan has no goal")
	}
	if len(def.Steps) == 0 {
		return nil, fmt.Errorf("plan has no steps")
	}

	return &def, nil
}

// truncateStr truncates s to at most n runes.
func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
