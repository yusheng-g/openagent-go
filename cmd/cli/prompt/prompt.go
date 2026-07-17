package prompt

import (
	"context"
	"fmt"
	"sort"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/tokenizer"
)

type Section struct {
	Name     string
	Content  string
	Priority int
	Static   bool
	Level    int
	Skip     func(input openagent.PromptInput) bool
}

type DynamicFn func(ctx context.Context, input openagent.PromptInput) []Section

type BudgetConfig struct {
	StaticRatio  float64
	DynamicRatio float64
	SummaryRatio float64
}

type Builder struct {
	staticSections []Section
	dynamicFns     []DynamicFn
	budget         BudgetConfig
	modelID        string
	contextWindow  int
}

func NewBuilder(modelID string, contextWindow int) *Builder {
	return &Builder{
		modelID:       modelID,
		contextWindow: contextWindow,
		budget: BudgetConfig{
			StaticRatio:  0.12,
			DynamicRatio: 0.08,
			SummaryRatio: 0.05,
		},
	}
}

func (b *Builder) AddStatic(s Section) *Builder {
	b.staticSections = append(b.staticSections, s)
	return b
}

func (b *Builder) AddDynamic(fn DynamicFn) *Builder {
	b.dynamicFns = append(b.dynamicFns, fn)
	return b
}

func (b *Builder) Build() openagent.PromptBuilder {
	return func(ctx context.Context, input openagent.PromptInput) ([]openagent.Message, error) {
		return b.assemble(ctx, input), nil
	}
}

func (b *Builder) assemble(ctx context.Context, input openagent.PromptInput) []openagent.Message {
	var msgs []openagent.Message

	static := b.collectStatic(input)
	if len(static) > 0 {
		content := renderSections(static)
		if content != "" {
			msgs = append(msgs, openagent.Message{
				Role:    openagent.RoleSystem,
				Content: content,
			})
		}
	}

	dynamic := b.collectDynamic(ctx, input)
	if len(dynamic) > 0 {
		content := renderSections(dynamic)
		if content != "" {
			msgs = append(msgs, openagent.Message{
				Role:    openagent.RoleSystem,
				Content: content,
			})
		}
	}

	if input.Compressed != nil && input.Compressed.Summary != "" {
		msgs = append(msgs, b.buildCompressedMsg(input.Compressed))
	}

	for name, body := range input.LoadedSkills {
		msgs = append(msgs, openagent.Message{
			Role:    openagent.RoleSystem,
			Content: "## Loaded Skill: " + name + "\n\n" + body,
		})
	}

	msgs = append(msgs, input.WorkingMessages...)

	return b.applyBudget(msgs)
}

func (b *Builder) collectStatic(input openagent.PromptInput) []Section {
	var sections []Section
	for _, s := range b.staticSections {
		if s.Skip != nil && s.Skip(input) {
			continue
		}
		if s.Content == "" {
			continue
		}
		sections = append(sections, s)
	}
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Priority < sections[j].Priority
	})
	return sections
}

func (b *Builder) collectDynamic(ctx context.Context, input openagent.PromptInput) []Section {
	var sections []Section
	for _, fn := range b.dynamicFns {
		for _, s := range fn(ctx, input) {
			if s.Skip != nil && s.Skip(input) {
				continue
			}
			if s.Content == "" {
				continue
			}
			sections = append(sections, s)
		}
	}
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Priority < sections[j].Priority
	})
	return sections
}

func (b *Builder) buildCompressedMsg(cc *openagent.CompressedContext) openagent.Message {
	content := "## Conversation Summary\n" + cc.Summary
	if len(cc.Hints) > 0 {
		content += "\n\n### Retrieval Hints\n"
		for i, h := range cc.Hints {
			content += fmt.Sprintf("%d. %s (query: %s)\n", i+1, h.Description, h.Query)
		}
	}
	return openagent.Message{Role: openagent.RoleSystem, Content: content}
}

func (b *Builder) applyBudget(msgs []openagent.Message) []openagent.Message {
	if b.contextWindow <= 0 || len(msgs) == 0 {
		return msgs
	}

	staticBudget := int(float64(b.contextWindow) * 0.95 * b.budget.StaticRatio)
	dynamicBudget := int(float64(b.contextWindow) * 0.95 * b.budget.DynamicRatio)
	summaryBudget := int(float64(b.contextWindow) * 0.95 * b.budget.SummaryRatio)

	var result []openagent.Message
	var workingMsgs []openagent.Message
	var staticIdx, dynamicIdx, compressedIdx int = -1, -1, -1

	for i, m := range msgs {
		if m.Role != openagent.RoleSystem {
			workingMsgs = append(workingMsgs, m)
			continue
		}

		if strings.HasPrefix(m.Content, "## Conversation Summary") {
			if compressedIdx < 0 {
				compressedIdx = i
			}
			continue
		}

		if strings.HasPrefix(m.Content, "## Loaded Skill:") {
			result = append(result, m)
			continue
		}

		if staticIdx < 0 {
			staticIdx = i
		} else if dynamicIdx < 0 {
			dynamicIdx = i
		}
	}

	if staticIdx >= 0 {
		content := b.trimContent(msgs[staticIdx].Content, staticBudget)
		result = append(result, openagent.Message{Role: openagent.RoleSystem, Content: content})
	}
	if dynamicIdx >= 0 {
		content := b.trimContent(msgs[dynamicIdx].Content, dynamicBudget)
		result = append(result, openagent.Message{Role: openagent.RoleSystem, Content: content})
	}
	if compressedIdx >= 0 {
		content := b.trimContent(msgs[compressedIdx].Content, summaryBudget)
		result = append(result, openagent.Message{Role: openagent.RoleSystem, Content: content})
	}

	result = append(result, workingMsgs...)
	return result
}

func (b *Builder) trimContent(content string, budget int) string {
	if budget <= 0 {
		return content
	}
	tokens := b.countTokens(content)
	if tokens <= budget {
		return content
	}

	lines := strings.Split(content, "\n")
	sections := splitByHeadings(lines)

	for len(sections) > 1 {
		sections = sections[:len(sections)-1]
		newContent := strings.Join(sections, "\n")
		if b.countTokens(newContent) <= budget {
			return newContent
		}
	}

	if len(sections) == 1 {
		return b.trimText(sections[0], budget)
	}
	return ""
}

func splitByHeadings(lines []string) []string {
	var sections []string
	var current []string
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			if len(current) > 0 {
				sections = append(sections, strings.Join(current, "\n"))
			}
			current = []string{line}
		} else {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		sections = append(sections, strings.Join(current, "\n"))
	}
	return sections
}

func (b *Builder) trimText(text string, budget int) string {
	if b.countTokens(text) <= budget {
		return text
	}

	runes := []rune(text)
	target := budget * 4
	if target > len(runes) {
		target = len(runes)
	}
	for target > 0 {
		candidate := string(runes[:target]) + "\n\n[truncated]"
		if b.countTokens(candidate) <= budget {
			return candidate
		}
		target -= target / 10
		if target < 10 {
			target = target - 1
		}
	}
	return text[:len(text)/2] + "\n\n[truncated]"
}

func (b *Builder) countTokens(text string) int {
	if text == "" {
		return 0
	}
	modelID := b.modelID
	if modelID == "" {
		modelID = "gpt-4o"
	}
	return tokenizer.Count(modelID, text)
}

func renderSections(sections []Section) string {
	var buf strings.Builder
	for i, s := range sections {
		if i > 0 {
			buf.WriteString("\n\n")
		}
		level := s.Level
		if level <= 0 {
			level = 2
		}
		buf.WriteString(strings.Repeat("#", level))
		buf.WriteString(" ")
		buf.WriteString(s.Name)
		buf.WriteString("\n\n")
		buf.WriteString(strings.TrimSpace(s.Content))
	}
	return buf.String()
}
