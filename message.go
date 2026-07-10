package openagent

// Role represents the role of a message in a conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is a single message in a conversation. Follows OpenAI message format.
// For text-only messages, use Content. For multimodal (images, audio), use
// ContentParts — the model implementation will serialize accordingly.
type Message struct {
	Role             Role          `json:"role"`
	Content          string        `json:"content,omitempty"`
	ContentParts     []ContentPart `json:"content_parts,omitempty"`
	ReasoningContent string        `json:"reasoning_content,omitempty"` // reasoning/thinking tokens (o1, deepseek-r1, etc.)
	Name             string        `json:"name,omitempty"`
	ToolCalls        []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID       string        `json:"tool_call_id,omitempty"`
}

// IsMultimodal returns true if this message carries multimodal content parts.
func (m Message) IsMultimodal() bool { return len(m.ContentParts) > 0 }

// ContentPart represents one part of a multimodal message content.
// Follows OpenAI content part format.
type ContentPart struct {
	Type       string      `json:"type"`                   // "text", "image_url", "input_audio"
	Text       string      `json:"text,omitempty"`
	ImageURL   *ImageURL   `json:"image_url,omitempty"`
	InputAudio *InputAudio `json:"input_audio,omitempty"`
}

// ImageURL references an image by URL or base64 data.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "low", "high", "auto"
}

// InputAudio carries audio data for speech input.
type InputAudio struct {
	Data   string `json:"data"`   // base64-encoded audio
	Format string `json:"format"` // "wav", "mp3"
}

// TextPart creates a text content part.
func TextPart(text string) ContentPart { return ContentPart{Type: "text", Text: text} }

// ImagePart creates an image content part from a URL.
func ImagePart(url string) ContentPart {
	return ContentPart{Type: "image_url", ImageURL: &ImageURL{URL: url}}
}

// UserMessage is a convenience constructor for a text user message.
func UserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

// UserMessageWithImage constructs a multimodal user message with text and an image URL.
func UserMessageWithImage(text, imageURL string) Message {
	return Message{
		Role: RoleUser,
		ContentParts: []ContentPart{
			TextPart(text),
			ImagePart(imageURL),
		},
	}
}

// ToolCallFunction is the function definition within a tool call.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolCall represents a function call requested by the model.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function ToolCallFunction `json:"function"`
}
