package chat

import (
	"strings"
	"testing"
)

// mkMarkdown builds a markdown document that mixes styled elements
// (headings, bold/inline code, fenced code, lists, tables) repeated to
// roughly the requested size.
func mkMarkdown(size int) string {
	block := "# Heading\n\nSome **bold** and `inline code` with [a link](https://x.example).\n\n" +
		"```go\nfunc main() { fmt.Println(\"hi\") }\n```\n\n" +
		"- item one\n- item two\n\n" +
		"| col A | col B |\n| ----- | ----- |\n| 1 | 2 |\n\n"
	var sb strings.Builder
	for sb.Len() < size {
		sb.WriteString(block)
	}
	return sb.String()[:size]
}

// newBenchModel is a chat model sized like a real terminal.
func newBenchModel() *Model {
	m := newTestModel()
	m.width = 120
	m.height = 40
	return m
}

// ── single 1MB markdown message ──

// BenchmarkRenderMarkdownText1M measures one markdown render of a 1MB
// assistant message (the reply path), including the surface-background pass.
func BenchmarkRenderMarkdownText1M(b *testing.B) {
	doc := mkMarkdown(1_000_000)
	b.SetBytes(int64(len(doc)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if out := renderMarkdownText(doc, 94); out == "" {
			b.Fatal("empty render")
		}
	}
}

// BenchmarkMarkdownSurfaceBG1M isolates the SGR repaint of the 1MB render
// output (what glosses every span with the card surface background).
func BenchmarkMarkdownSurfaceBG1M(b *testing.B) {
	out := renderMarkdownText(mkMarkdown(1_000_000), 94)
	b.SetBytes(int64(len(out)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := markdownSurfaceBG(out); got == "" {
			b.Fatal("empty output")
		}
	}
}

// BenchmarkRenderMessagesSingle1M runs the full message pipeline for one
// 1MB assistant message (styleMessageBlock → markdown → messageCard).
func BenchmarkRenderMessagesSingle1M(b *testing.B) {
	m := newBenchModel()
	m.messages = []ChatMessage{{Role: "assistant", Content: mkMarkdown(1_000_000)}}
	b.SetBytes(1_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.renderCache = nil
		if out := m.renderMessages(); out == "" {
			b.Fatal("empty render")
		}
	}
}

// ── 1MB spread over many messages ──

// BenchmarkRenderMessages1M_1000x1K renders 1000 messages totalling ~1MB
// (cold cache: the budget-constrained full-document path).
func BenchmarkRenderMessages1M_1000x1K(b *testing.B) {
	doc := mkMarkdown(1_000)
	m := newBenchModel()
	m.messages = make([]ChatMessage, 1000)
	for i := range m.messages {
		m.messages[i] = ChatMessage{Role: "assistant", Content: doc}
	}
	b.SetBytes(1_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.renderCache = nil
		if out := m.renderMessages(); out == "" {
			b.Fatal("empty render")
		}
	}
}

// BenchmarkRenderMessages1M_1000x1K_Warm reuses the per-message style cache
// (steady-state: unchanged messages skip re-styling).
func BenchmarkRenderMessages1M_1000x1K_Warm(b *testing.B) {
	doc := mkMarkdown(1_000)
	m := newBenchModel()
	m.messages = make([]ChatMessage, 1000)
	for i := range m.messages {
		m.messages[i] = ChatMessage{Role: "assistant", Content: doc}
	}
	m.renderMessages() // prime the cache
	b.SetBytes(1_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if out := m.renderMessages(); out == "" {
			b.Fatal("empty render")
		}
	}
}

// ── realistic single-message sizes (cold cache) ──

func benchSingleSize(b *testing.B, size int) {
	m := newBenchModel()
	m.messages = []ChatMessage{{Role: "assistant", Content: mkMarkdown(size)}}
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.renderCache = nil
		if out := m.renderMessages(); out == "" {
			b.Fatal("empty render")
		}
	}
}

func BenchmarkRenderMessagesSingle100K(b *testing.B) { benchSingleSize(b, 100_000) }
func BenchmarkRenderMessagesSingle10K(b *testing.B)  { benchSingleSize(b, 10_000) }
func BenchmarkRenderMessagesSingle1K(b *testing.B)   { benchSingleSize(b, 1_000) }

// ── virtual-scroll window over 1MB ──

// BenchmarkVirtualWindow1M renders the visible 30-row window of a 1MB
// transcript (100 messages × 10KB) — the steady path while scrolling.
func BenchmarkVirtualWindow1M(b *testing.B) {
	doc := mkMarkdown(10_000)
	m := newBenchModel()
	m.messages = make([]ChatMessage, 100)
	for i := range m.messages {
		m.messages[i] = ChatMessage{Role: "assistant", Content: doc}
	}
	b.SetBytes(1_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.renderCache = nil
		if out := m.renderVirtualDocAt(30, 5_000); out == "" {
			b.Fatal("empty render")
		}
	}
}
