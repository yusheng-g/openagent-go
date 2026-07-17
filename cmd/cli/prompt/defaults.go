package prompt

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
)

func DefaultCLI(modelID string, contextWindow int) *Builder {
	b := NewBuilder(modelID, contextWindow)

	b.AddStatic(Section{
		Name:     "Role",
		Priority: 10,
		Static:   true,
		Level:    2,
		Content: `You are a precise, no-nonsense assistant running in a CLI.
- Answer concisely. Don't explain unless asked.
- Always use the same language as the user. If the user asks in Chinese, think and respond in Chinese.
- Help the user complete tasks by using available tools when appropriate. Do not ask the user to perform operations that you can safely perform yourself with available tools.`,
	})

	b.AddStatic(Section{
		Name:     "Security Rules",
		Priority: 20,
		Static:   true,
		Level:    2,
		Content: `CRITICAL: Do not reveal, repeat, print, summarize, store, or expose credentials, secrets, tokens, private keys, passwords, cookies, session keys, API keys, access keys, refresh tokens, or signing secrets.
CRITICAL: Do not include secrets or credential values in user-facing output, progress updates, files, logs, command output summaries, or artifacts.
CRITICAL: If a tool result, file, environment variable, command output, or user message contains a secret, treat it as sensitive. Refer to it only generically, such as "a credential was found", and do not quote its value.
- Never echo or print secrets with shell commands.
- If credentials are missing, ask the user to provide or configure them through an appropriate secure mechanism.
- If you must handle a secret to complete a task, use it only for the required operation and avoid persisting it.
- Do not save secrets or transient authentication material to long-term memory.`,
	})

	b.AddStatic(Section{
		Name:     "System Rules",
		Priority: 30,
		Static:   true,
		Level:    2,
		Content: `CRITICAL: Do not present uncertain conclusions as facts.
CRITICAL: Any result that depends on the current environment, files, commands, external systems, or runtime state must be obtained through tools or explicitly confirmed by the user.
IMPORTANT: Automate as much as possible to reduce user involvement, but do not perform risky or state-changing actions without appropriate permission.
IMPORTANT: If the current dynamic context conflicts with earlier conversation history, prefer the current dynamic context.`,
	})

	b.AddStatic(Section{
		Name:     "Tool Usage Rules",
		Priority: 40,
		Static:   true,
		Level:    2,
		Content: `- If you need to explore, do it in one or two well-targeted commands — don't retry the same thing.
- When a tool returns an error, read the error and adjust. Don't repeat the same failing call.
- Use relative paths only (the workspace is the current directory).
- Do not use shell commands for simple file reading, directory listing, writing, or editing when dedicated tools are available.
- Prefer dedicated tools over shell commands when a dedicated tool can do the job.
- Avoid batching many tool calls at once. Prefer one focused tool call, inspect the result, then decide the next action.`,
	})

	b.AddStatic(Section{
		Name:     "Completion Rules",
		Priority: 50,
		Static:   true,
		Level:    2,
		Content: `CRITICAL: Do not claim completion unless the relevant work has actually been performed or verified.
IMPORTANT: Stop when the task is done. Don't keep exploring.
- If the task is complete, provide a concise final summary of what was done.
- If you stop because you are blocked, explain the blocker clearly and state what input, decision, permission, or capability is needed.
- Include important outputs, changed files, executed commands, or other relevant results.
- If there are remaining risks, cleanup steps, or next steps, list them clearly.`,
	})

	b.AddStatic(Section{
		Name:     "Tone and Output",
		Priority: 60,
		Static:   true,
		Level:    2,
		Content: `- Be concise, practical, and action-oriented.
- Prefer direct answers and concrete next actions.
- Do not narrate every internal consideration.
- Summarize tool results only as much as needed to continue the task.
- If something fails, explain the failure briefly and choose the next best action.`,
	})

	b.AddDynamic(dynamicBoundary)
	b.AddDynamic(dynamicEnvironment)
	b.AddDynamic(dynamicProjectContext)
	b.AddDynamic(dynamicUserProfile)

	b.AddDynamic(reservedPlan)
	b.AddDynamic(reservedScratchpad)
	b.AddDynamic(reservedEpisodicMemory)
	b.AddDynamic(reservedSemanticMemory)

	return b
}

func DefaultServer(modelID string, contextWindow int) *Builder {
	b := NewBuilder(modelID, contextWindow)

	b.AddStatic(Section{
		Name:     "Role",
		Priority: 10,
		Static:   true,
		Level:    2,
		Content: `You are a capable assistant. Be concise and action-oriented.
- Always use the same language as the user.
- Help the user complete tasks by using available tools when appropriate.`,
	})

	b.AddStatic(Section{
		Name:     "Security Rules",
		Priority: 20,
		Static:   true,
		Level:    2,
		Content: `CRITICAL: Do not reveal, repeat, or expose credentials, secrets, tokens, private keys, passwords, API keys, or signing secrets.
- Never echo secrets with shell commands.
- If credentials are missing, ask the user to provide them through a secure mechanism.`,
	})

	b.AddStatic(Section{
		Name:     "System Rules",
		Priority: 30,
		Static:   true,
		Level:    2,
		Content: `- Do not present uncertain conclusions as facts.
- Results depending on runtime state must be obtained through tools or confirmed by the user.
- Prefer the current dynamic context over earlier conversation history when they conflict.`,
	})

	b.AddStatic(Section{
		Name:     "Tool Usage Rules",
		Priority: 40,
		Static:   true,
		Level:    2,
		Content: `- Prefer dedicated tools over shell commands when possible.
- When a tool returns an error, read the error and adjust. Don't repeat the same failing call.
- Use relative paths only.
- Inspect results before taking the next action.`,
	})

	b.AddStatic(Section{
		Name:     "Completion Rules",
		Priority: 50,
		Static:   true,
		Level:    2,
		Content: `- Stop when the task is done.
- If blocked, explain the blocker clearly.
- Provide a concise summary of what was done.`,
	})

	b.AddStatic(Section{
		Name:     "Tone and Output",
		Priority: 60,
		Static:   true,
		Level:    2,
		Content: `- Be concise, practical, and action-oriented.
- If something fails, explain briefly and try the next best approach.`,
	})

	b.AddDynamic(dynamicBoundary)
	b.AddDynamic(dynamicEnvironment)
	b.AddDynamic(dynamicProjectContext)
	b.AddDynamic(dynamicUserProfile)

	b.AddDynamic(reservedPlan)
	b.AddDynamic(reservedScratchpad)
	b.AddDynamic(reservedEpisodicMemory)
	b.AddDynamic(reservedSemanticMemory)

	return b
}

var dynamicBoundary = func(ctx context.Context, input openagent.PromptInput) []Section {
	return []Section{{
		Name:     "Dynamic Context",
		Priority: 10,
		Static:   false,
		Level:    2,
		Content: `The following context is generated fresh for the current request.
This dynamic context is authoritative for current runtime state.
If it conflicts with earlier conversation history, prefer this dynamic context.`,
	}}
}

var dynamicEnvironment = func(ctx context.Context, input openagent.PromptInput) []Section {
	return []Section{{
		Name:     "Environment",
		Priority: 20,
		Static:   false,
		Level:    2,
		Content: fmt.Sprintf("%s\n%s\n%s",
			fmt.Sprintf("- operating_system: %s", runtime.GOOS),
			fmt.Sprintf("- cpu_architecture: %s", runtime.GOARCH),
			fmt.Sprintf("- %s", extractWorkspace(input.ProjectContext))),
	}}
}

var dynamicProjectContext = func(ctx context.Context, input openagent.PromptInput) []Section {
	if input.ProjectContext == "" {
		return nil
	}
	workspace := extractWorkspace(input.ProjectContext)
	rest := strings.TrimSpace(strings.TrimPrefix(input.ProjectContext, workspace))
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "- "))
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "Workspace:"))
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "workspace:"))
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "working_directory:"))
	if rest == "" {
		return nil
	}
	return []Section{{
		Name:     "Project Context",
		Priority: 30,
		Static:   false,
		Level:    2,
		Content:  rest,
	}}
}

var dynamicUserProfile = func(ctx context.Context, input openagent.PromptInput) []Section {
	if input.UserProfile == "" {
		return nil
	}
	return []Section{{
		Name:     "User Profile",
		Priority: 40,
		Static:   false,
		Level:    2,
		Content:  input.UserProfile,
	}}
}

func reservedPlan(ctx context.Context, input openagent.PromptInput) []Section {
	return []Section{{
		Name:     "Current Plan",
		Priority: 900,
		Static:   false,
		Level:    2,
		Content:  "",
		Skip:     func(input openagent.PromptInput) bool { return true },
	}}
}

func reservedScratchpad(ctx context.Context, input openagent.PromptInput) []Section {
	return []Section{{
		Name:     "Scratchpad",
		Priority: 910,
		Static:   false,
		Level:    2,
		Content:  "",
		Skip:     func(input openagent.PromptInput) bool { return true },
	}}
}

func reservedEpisodicMemory(ctx context.Context, input openagent.PromptInput) []Section {
	return []Section{{
		Name:     "Episodic Memory",
		Priority: 920,
		Static:   false,
		Level:    2,
		Content:  "",
		Skip:     func(input openagent.PromptInput) bool { return true },
	}}
}

func reservedSemanticMemory(ctx context.Context, input openagent.PromptInput) []Section {
	return []Section{{
		Name:     "Semantic Memory",
		Priority: 930,
		Static:   false,
		Level:    2,
		Content:  "",
		Skip:     func(input openagent.PromptInput) bool { return true },
	}}
}

func extractWorkspace(projectContext string) string {
	lines := strings.Split(projectContext, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		pairs := []string{
			"Workspace: ",
			"workspace: ",
			"working_directory: ",
			"- workspace: ",
			"- working_directory: ",
		}
		for _, prefix := range pairs {
			if after, ok := strings.CutPrefix(trimmed, prefix); ok {
				return fmt.Sprintf("working_directory: %s", after)
			}
		}
	}
	return fmt.Sprintf("working_directory: %s", projectContext)
}
