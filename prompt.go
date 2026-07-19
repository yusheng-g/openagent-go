package openagent

import "context"

// PromptInput carries all the data the Prompt.Builder needs to assemble
// the final message list for a model call. The Runner gathers this from
// Agent, Session, and Memory before each turn.
type PromptInput struct {
	AgentDescription string
	Instructions string // joined form of Agent.SystemPrompts, set by runner

	// From Memory
	WorkingMessages []Message
	Compressed      *CompressedContext

	// Available tools and skills
	Tools           []FunctionDefinition
	AvailableSkills []SkillInfo
	LoadedSkills    map[string]string // name → body, injected when use_skill was called

	// From Session
	UserProfile    string
	ProjectContext string
}

// RetrievalHint tells the model how to retrieve the original context.
type RetrievalHint struct {
	Description string `json:"description"`
	Query       string `json:"query"`
}

// PromptBuilder assembles the message list for a model call from PromptInput.
// nil means use the default (system instructions + working messages + skills).
type PromptBuilder func(ctx context.Context, input PromptInput) ([]Message, error)
