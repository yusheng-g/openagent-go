package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/channel"
	"github.com/yusheng-g/openagent-go/channel/feishu"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
)

// RunChannels starts all configured IM channels.
func RunChannels(ctx context.Context, agent *openagent.Agent, cfg config.ChannelsConfig) error {
	var channels []channel.Channel

	if cfg.Feishu != nil {
		channels = append(channels, feishu.New(cfg.Feishu.AppID, cfg.Feishu.AppSecret))
	}

	if len(channels) == 0 {
		return nil
	}

	for _, ch := range channels {
		log.Printf("channel: starting %s", ch.Name())
		go func(ch channel.Channel) {
			handler := channel.MessageHandler(func(msgCtx context.Context, msg channel.IncomingMessage, reply channel.ReplyFunc) {
				sessionID := ch.Name() + "_" + msg.ChatID

				go func() {
					session := openagent.Session{
						ID:        sessionID,
						CreatedAt: time.Now(),
					}
					stream := agent.RunStream(msgCtx, session, openagent.UserMessage(msg.Text))
					streamReply(reply, stream)
				}()
			})

			if err := ch.Start(ctx, handler); err != nil {
				log.Printf("channel: %s stopped: %v", ch.Name(), err)
			}
		}(ch)
	}

	return nil
}

// patchQueue decouples card rendering from Feishu API calls.
// Updates to the same card within 500ms are collapsed — only the
// latest version is sent. Card creation (which returns a message ID)
// is synchronous; patches are debounced via time.AfterFunc.
//
// No background goroutine — the timer is started on first mark and
// fires once, sending all dirty cards in batch.
type patchQueue struct {
	reply   channel.ReplyFunc
	mu      sync.Mutex
	dirty   map[string]*channel.Card
	timer   *time.Timer
	stopped bool
}

func newPatchQueue(reply channel.ReplyFunc) *patchQueue {
	return &patchQueue{
		reply: reply,
		dirty: make(map[string]*channel.Card),
	}
}

func (pq *patchQueue) mark(msgID string, card *channel.Card) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if pq.stopped {
		return
	}
	pq.dirty[msgID] = card
	if pq.timer == nil {
		pq.timer = time.AfterFunc(500*time.Millisecond, pq.flush)
	}
}

func (pq *patchQueue) create(msg channel.ReplyMessage) string {
	id, _ := pq.reply(context.Background(), msg)
	return id
}

func (pq *patchQueue) flush() {
	pq.mu.Lock()
	if pq.stopped {
		pq.mu.Unlock()
		return
	}
	if len(pq.dirty) == 0 {
		pq.timer = nil
		pq.mu.Unlock()
		return
	}
	batch := pq.dirty
	pq.dirty = make(map[string]*channel.Card)
	pq.timer = nil
	pq.mu.Unlock()

	for msgID, card := range batch {
		msg := channel.ReplyMessage{UpdateID: msgID, Card: card}
		_, _ = pq.reply(context.Background(), msg)
	}
}

func (pq *patchQueue) stop() {
	pq.mu.Lock()
	pq.stopped = true
	if pq.timer != nil {
		pq.timer.Stop()
		pq.timer = nil
	}
	batch := pq.dirty
	pq.dirty = nil
	pq.mu.Unlock()

	for msgID, card := range batch {
		msg := channel.ReplyMessage{UpdateID: msgID, Card: card}
		_, _ = pq.reply(context.Background(), msg)
	}
}

// streamReply drains the agent stream and sends every message as a card.
//
// Card patches are debounced — updates to the same card within 500ms
// are collapsed so the Feishu API sees at most 2 PATCH/s per card.
// This prevents the event loop from blocking on HTTP latency.
func streamReply(reply channel.ReplyFunc, stream <-chan openagent.StreamEvent) {
	type tpend struct {
		name string
		args string
	}

	var (
		pq = newPatchQueue(reply)
		// Make sure final flush happens.
		_ = pq.stop // used via defer-like pattern below
	)

	var (
		textCardID string          // response card ID
		textBuf    strings.Builder // accumulated text since last patch
		textLast   = time.Now()    // last patch time

		thoughtCardID string          // reasoning card ID
		thoughtBuf    strings.Builder // accumulated reasoning text

		pendingTool = map[string]*tpend{} // toolCallID → {name, args}
		toolCardID  = map[string]string{} // toolCallID → card message ID
		toolBuf     = map[string]string{} // toolCallID → accumulated output
	)

	mkCard := func(title, body string, color channel.CardColor) *channel.Card {
		return &channel.Card{Header: channel.CardHeader{Title: title}, Content: body, Color: color}
	}

	// ── Periodic flush (ad-hoc, not tick-based) ──
	// Called after each event. If enough time/content has passed,
	// marks the text card dirty for the next queue tick.

	flushText := func() {
		if textBuf.Len() == 0 {
			return
		}
		body := textBuf.String()
		if textCardID == "" {
			textCardID = pq.create(channel.ReplyMessage{Card: mkCard("🧠 openagent", body, channel.CardColorGrey)})
		} else {
			pq.mark(textCardID, mkCard("🧠 openagent", body, channel.CardColorGrey))
			
		}
		textLast = time.Now()
	}

	finalizeThoughtCard := func() {
		if thoughtCardID == "" {
			return
		}
		pq.mark(thoughtCardID, mkCard("🤔 thinking — done", thoughtBuf.String(), channel.CardColorYellow))
		pq.flush()
		thoughtCardID = ""
		thoughtBuf.Reset()
		
	}

	finalizeTextCard := func() {
		if textCardID == "" {
			return
		}
		pq.mark(textCardID, mkCard("🧠 openagent", textBuf.String(), channel.CardColorGrey))
		pq.flush()
		textCardID = ""
		textBuf.Reset()
	}

	for evt := range stream {
		switch evt.Type {
		case openagent.StreamThought:
			thoughtBuf.WriteString(evt.Text)
			body := thoughtBuf.String()
			if thoughtCardID == "" {
				thoughtCardID = pq.create(channel.ReplyMessage{Card: mkCard("🤔 thinking", body, channel.CardColorYellow)})
			} else {
				pq.mark(thoughtCardID, mkCard("🤔 thinking", body, channel.CardColorYellow))
				
			}

		case openagent.StreamTextDelta:
			finalizeThoughtCard()
			textBuf.WriteString(evt.Text)
			if time.Since(textLast) >= 80*time.Millisecond || textBuf.Len() >= 50 {
				flushText()
			}

		case openagent.StreamToolCall:
			finalizeThoughtCard()
			finalizeTextCard()
			for _, tc := range evt.Message.ToolCalls {
				if tc.Function.Name == "plan_create" {
					goal, steps := parsePlanCreate(tc.Function.Arguments)
					if goal != "" {
						pq.create(channel.ReplyMessage{Card: mkCard("📋 "+goal, steps, channel.CardColorBlue)})
					}
					continue
				}
				pendingTool[tc.ID] = &tpend{name: tc.Function.Name, args: tc.Function.Arguments}
				toolBuf[tc.ID] = ""
			}

		case openagent.StreamToolProgress:
			t, ok := pendingTool[evt.ToolCallID]
			if !ok {
				continue
			}
			toolBuf[evt.ToolCallID] += evt.Text

			card := toolCard(t.name, t.args, "in_progress", toolBuf[evt.ToolCallID])
			if msgID, exists := toolCardID[evt.ToolCallID]; exists {
				pq.mark(msgID, card)
			} else {
				id := pq.create(channel.ReplyMessage{Card: card})
				if id != "" {
					toolCardID[evt.ToolCallID] = id
				}
			}

		case openagent.StreamToolResult:
			t, ok := pendingTool[evt.Message.ToolCallID]
			if !ok {
				continue
			}
			delete(pendingTool, evt.Message.ToolCallID)
			delete(toolBuf, evt.Message.ToolCallID)

			if t.name == "plan_update" {
				continue
			}

			output := evt.Message.Content
			status := "completed"
			if strings.HasPrefix(output, "error: ") {
				status = "failed"
			}

			card := toolCard(t.name, t.args, status, output)
			if msgID := toolCardID[evt.Message.ToolCallID]; msgID != "" {
				delete(toolCardID, evt.Message.ToolCallID)
				pq.mark(msgID, card)
				pq.flush()
			} else {
				pq.create(channel.ReplyMessage{Card: card})
			}

		case openagent.StreamRetrying:
			finalizeThoughtCard()
			finalizeTextCard()
			errMsg := "retrying..."
			if evt.Error != nil {
				errMsg = fmt.Sprintf("retrying: %v", evt.Error)
			}
			pq.create(channel.ReplyMessage{Card: mkCard("⚠️ retrying", errMsg, channel.CardColorYellow)})

		case openagent.StreamDone:
			finalizeThoughtCard()
			finalizeTextCard()

		case openagent.StreamError:
			finalizeThoughtCard()
			finalizeTextCard()
			if evt.Error != nil {
				pq.create(channel.ReplyMessage{Card: mkCard("❌ error", fmt.Sprintf("%v", evt.Error), channel.CardColorRed)})
			}
			pq.stop()
			return

		case openagent.StreamAborted:
			finalizeThoughtCard()
			finalizeTextCard()
			pq.stop()
			return
		}
	}

	finalizeThoughtCard()
	finalizeTextCard()
	pq.stop()
}

// ── Tool card ──

func toolCard(name, args, status, output string) *channel.Card {
	title := toolEmoji(name) + " " + name
	color := channel.CardColorGrey
	switch status {
	case "completed":
		title = toolEmoji(name) + " " + name + " ✓"
		color = channel.CardColorGreen
	case "failed":
		title = toolEmoji(name) + " " + name + " ✗"
		color = channel.CardColorRed
	case "in_progress":
		color = channel.CardColorPurple
	}

	body := formatInput(name, args)
	if output != "" {
		body += "\n```\n" + output + "\n```"
	}

	return &channel.Card{
		Header:  channel.CardHeader{Title: title},
		Content: body,
		Color:   color,
	}
}

func formatInput(name, args string) string {
	m := jsonMap(args)
	switch name {
	case "shell", "terminal_create":
		cmd := jsonStr(m, "command")
		if cmd != "" {
			return "```\n" + trunc(cmd, 500) + "\n```"
		}
	case "read", "read_client_file":
		path := jsonStr(m, "path")
		if path == "" {
			path = jsonStr(m, "uri")
		}
		if path != "" {
			return "`" + path + "`"
		}
	case "write", "write_client_file":
		path := jsonStr(m, "path")
		if path == "" {
			path = jsonStr(m, "uri")
		}
		if path != "" {
			return "`" + path + "`"
		}
	case "grep":
		q := jsonStr(m, "query")
		if q == "" {
			q = jsonStr(m, "pattern")
		}
		path := jsonStr(m, "path")
		if path == "" {
			path = jsonStr(m, "dir")
		}
		if q != "" {
			return "`" + q + "`" + pathStr(path)
		}
	case "recall":
		q := jsonStr(m, "query")
		if q != "" {
			return "`" + q + "`"
		}
	case "ls":
		path := jsonStr(m, "path")
		if path == "" {
			path = jsonStr(m, "dir")
		}
		if path != "" {
			return "`" + path + "`"
		}
	case "subagent":
		n := jsonStr(m, "name")
		t := jsonStr(m, "task")
		if n != "" {
			return "**" + n + "** — " + trunc(t, 200)
		}
		return trunc(t, 200)
	}
	return "```\n" + trunc(args, 200) + "\n```"
}

func pathStr(p string) string {
	if p != "" {
		return " in `" + p + "`"
	}
	return ""
}

func toolEmoji(name string) string {
	switch name {
	case "read", "read_client_file":
		return "📖"
	case "write", "write_client_file":
		return "✏️"
	case "shell", "terminal_create":
		return "💻"
	case "grep":
		return "🔍"
	case "ls":
		return "📂"
	case "recall":
		return "🧠"
	case "subagent":
		return "🤖"
	case "use_skill":
		return "📦"
	default:
		return "🔧"
	}
}

// ── Helpers ──

func jsonMap(raw string) map[string]any {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	return m
}

func jsonStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func parsePlanCreate(args string) (goal string, steps string) {
	var p struct {
		Goal  string `json:"goal"`
		Steps []struct {
			Content  string `json:"content"`
			Priority string `json:"priority"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil || p.Goal == "" {
		return "", ""
	}

	var b strings.Builder
	for i, s := range p.Steps {
		emoji := "⬜"
		switch s.Priority {
		case "high":
			emoji = "🔴"
		case "medium":
			emoji = "🟡"
		case "low":
			emoji = "🟢"
		}
		fmt.Fprintf(&b, "%s **Step %d:** %s\n", emoji, i+1, s.Content)
	}
	return p.Goal, b.String()
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
