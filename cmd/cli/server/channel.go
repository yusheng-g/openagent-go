package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/agent"
	"github.com/yusheng-g/openagent-go/channel"
	"github.com/yusheng-g/openagent-go/kernel"
	"github.com/yusheng-g/openagent-go/session"
	opentool "github.com/yusheng-g/openagent-go/tool"

	"github.com/yusheng-g/openagent-go/cmd/cli/config"
)

// ChannelEnv carries the shared runtime environment for all channel
// connection managers. Each manager reads only the fields it needs.
type ChannelEnv struct {
	Ctx         context.Context
	Cfg         *agent.Agent
	Deps        kernel.Deps
	DefaultMode string // feishu approval mode ("manual" | "auto"; empty = "manual")
	WorkDir     string // workspace root for channel-specific tools (feishu SendFile)
	MetaStore   session.Store // session metadata store (nil = no meta tagging)
}

// RunChannels wires the feishu, wechat, and wecom connection managers.
// All managers are ALWAYS created with their settings credentials — the
// frontend control panel needs the status/connect endpoints even when no
// channel is configured, and a settings-configured channel connects on
// demand via POST /connect.
//
// Connection NEVER auto-starts from settings alone: settings is the
// credential store, not a connect instruction. The only auto-connect
// entry points are --channel <name> (Explicit, fail-fast: the user asked
// for the bot, so running silently without it would read as "connected"
// while delivering nothing) and the frontend's POST /connect.
func RunChannels(env ChannelEnv, channelsCfg config.ChannelsConfig) (*FeishuManager, *WechatManager, *WecomManager, error) {
	feishuMgr := NewFeishuManager(env, channelsCfg.Feishu)
	if channelsCfg.Feishu != nil && channelsCfg.Feishu.Explicit {
		if err := feishuMgr.Connect(); err != nil {
			return nil, nil, nil, err
		}
	}

	wechatMgr := NewWechatManager(env, channelsCfg.Wechat)
	if channelsCfg.Wechat != nil && channelsCfg.Wechat.Explicit {
		if err := wechatMgr.Connect(); err != nil {
			return nil, nil, nil, err
		}
	}

	wecomMgr := NewWecomManager(env, channelsCfg.Wecom)
	if channelsCfg.Wecom != nil && channelsCfg.Wecom.Explicit {
		if err := wecomMgr.Connect(); err != nil {
			return nil, nil, nil, err
		}
	}
	return feishuMgr, wechatMgr, wecomMgr, nil
}

// ensureChannelMeta tags a channel session's metadata and title —
// mirroring the ACP path (saveMeta + first-prompt auto-title): a newly
// seen session is created with full timestamps, the meta map carries
// kind="channel" (ACP counterpart: kind="acp") plus the originating
// channel in "_channel" (feishu/wechat/wecom), and an empty title is
// set from the first user message's first line (the ACP behavior for
// first prompts). meta/title are written once per session; best-effort
// (a missing store or a failed write only costs a re-tag on the next
// message).
func ensureChannelMeta(store session.Store, sessionID, channelName, firstMessage string) {
	if store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	now := time.Now()
	info, err := store.Get(ctx, sessionID)
	if err != nil || info == nil {
		info = &session.SessionInfo{ID: sessionID, CreatedAt: now, UpdatedAt: now}
	}
	if _, ok := session.GetMeta[string](*info, "_channel"); ok {
		return // already tagged
	}
	info.SetMeta("kind", "channel")
	info.SetMeta("_channel", channelName)
	if info.Title == "" && strings.TrimSpace(firstMessage) != "" {
		info.Title = firstLine(firstMessage, 80)
	}
	info.UpdatedAt = now
	if err := store.Save(ctx, *info); err != nil {
		slog.Warn("channel session meta save failed", "session", sessionID, "error", err)
	}
}

// firstLine takes the first line of s, truncated to maxLen runes —
// shared title derivation for channel sessions (ACP keeps its own copy
// in acp/server.go).
func firstLine(s string, maxLen int) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "..."
	}
	return s
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
	id, err := pq.reply(context.Background(), msg)
	if err != nil {
		slog.Warn("pq.create failed", "error", err)
	}
	slog.Info("pq.create", "msgID", id)
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

	slog.Info("pq.flush", "batchSize", len(batch))
	for msgID, card := range batch {
		msg := channel.ReplyMessage{UpdateID: msgID, Card: card}
		if _, err := pq.reply(context.Background(), msg); err != nil {
			slog.Warn("patch card failed", "msgID", msgID, "error", err)
		}
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
		if _, err := pq.reply(context.Background(), msg); err != nil {
			slog.Warn("patch card failed", "msgID", msgID, "error", err)
		}
	}
}

// runCardUpdater is implemented by channels that support embedding approval
// buttons in the run card instead of sending a separate approval card.
// Currently only *feishu.Channel.
type runCardUpdater interface {
	SetRunCardUpdater(sessionID string, cb func(toolName, args, approvalID string))
	ClearRunCardUpdater(sessionID string)
}

// CardSizer returns the serialized size of a card for platforms that have a
// content size limit (e.g. Feishu ~30KB interactive cards). When non-nil,
// streamReply uses it to detect oversize cards and truncate tool history.
// Currently only *feishu.Channel.
type CardSizer interface {
	CardSize(card *channel.Card) int
}

// maxCardBytes is the threshold at which streamReply abandons the current
// card and starts a fresh one with only the latest tool call. Feishu's
// interactive card limit is ~30KB; 28KB leaves a 2KB safety margin.
const maxCardBytes = 28000

// maxThoughtDisplay caps the thinking content shown in a run card panel.
// The full buffer is kept for session persistence; only the card view is
// trimmed to the most recent bytes (recent reasoning is most relevant).
const maxThoughtDisplay = maxCardBytes / 3

// ModeController is implemented by channels that support per-chat mode
// switching (manual / auto). The handler intercepts /mode text commands
// and card-action button clicks to switch modes without going through
// the ACP server. Currently only *feishu.Channel.
type ModeController interface {
	GetMode(chatID string) string
	SetMode(chatID, mode string)
	BuildModeCard(chatID string) *channel.Card
}

// isModeCommand reports whether the text is a /mode invocation (with or
// without an explicit mode argument).
func isModeCommand(text string) bool {
	t := strings.TrimSpace(text)
	return t == "/mode" || strings.HasPrefix(t, "/mode ")
}

// isClearCommand reports whether the text is a /clear invocation.
func isClearCommand(text string) bool {
	t := strings.TrimSpace(text)
	return t == "/clear" || strings.HasPrefix(t, "/clear ")
}

// handleClearCommand deletes the session identified by sessionID and sends
// a confirmation reply. Shared by feishu, wechat, and wecom handlers.
// The caller has already verified isClearCommand(msg.Text) is true.
func handleClearCommand(deps kernel.Deps, reply channel.ReplyFunc, msgCtx context.Context, sessionID string) {
	if deps.SessionStore != nil {
		if err := deps.SessionStore.DeleteSession(msgCtx, sessionID); err != nil {
			_, _ = reply(msgCtx, channel.ReplyMessage{Text: "❌ 清空失败: " + err.Error()})
		} else {
			_, _ = reply(msgCtx, channel.ReplyMessage{Text: "✅ 已清空对话历史"})
		}
	} else {
		_, _ = reply(msgCtx, channel.ReplyMessage{Text: "✅ 已清空对话历史"})
	}
}

// parseModeArgs extracts the mode argument from "/mode <args>".
// Returns "" when no argument is present.
func parseModeArgs(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) >= 2 {
		return strings.ToLower(fields[1])
	}
	return ""
}

// streamReply drains the agent stream and sends every message as a card.
//
// Card patches are debounced — updates to the same card within 500ms
// are collapsed so the Feishu API sees at most 2 PATCH/s per card.
// This prevents the event loop from blocking on HTTP latency.
//
// When updater is non-nil, approval buttons are embedded in the run card
// via a callback + signal channel: Ask calls the callback (setting
// pendingApproval), which sends on approvalSig, which wakes this loop
// to flush the card with buttons. This avoids the approver reading
// streamReply's shared state from a different goroutine.
//
// When sizer is non-nil, the serialized card size is checked before each
// patch; if it exceeds maxCardBytes the current card is abandoned and a
// fresh one is created with only the latest tool call.
func streamReply(reply channel.ReplyFunc, stream <-chan openagent.StreamEvent, sessionID string, updater runCardUpdater, sizer CardSizer) {
	type tpend struct {
		name string
	}

	var (
		pq = newPatchQueue(reply)
		// Make sure final flush happens.
		_ = pq.stop // used via defer-like pattern below
	)

	// One card per agent run. Title tracks the stage; body interleaves
	// thinking (collapsed) → [text → tool call (collapsed)]* → text.
	var (
		runCardID   string
		thoughtBuf  strings.Builder
		pendingTool = map[string]*tpend{} // toolCallID → {name}
		blocks      []block

		// Stage only advances (thinking < toolcalling < answering < done),
		// so a second round of reasoning mid-turn doesn't flicker the title
		// back to "思考中".
		stage    = stageThinking
		lastErr  string
		lastTime = time.Now()

		// Approval integration: pendingApproval is set by the Ask callback
		// (runtime goroutine) and read by flushRunCard (this goroutine).
		// The mutex makes the cross-goroutine access safe; approvalSig
		// wakes the select loop to flush when buttons are added.
		approvalMu      sync.Mutex
		pendingApproval *channel.CardApproval
		approvalSig     = make(chan struct{}, 1)
	)

	// flushRunCard rebuilds the single run card from current state and
	// creates-or-patches it. The 500ms patch debounce bounds API rate.
	flushRunCard := func() {
		if thoughtBuf.Len() == 0 && len(blocks) == 0 && lastErr == "" {
			return
		}
		card := runCard(stage, thoughtBuf.String(), blocks, lastErr)
		approvalMu.Lock()
		card.Approval = pendingApproval
		approvalMu.Unlock()

		var cardSize int
		if sizer != nil {
			cardSize = sizer.CardSize(card)
		}
		slog.Info("flushRunCard",
			"stage", stage.title(),
			"thoughtLen", thoughtBuf.Len(),
			"blocks", len(blocks),
			"cardSize", cardSize,
			"runCardID", runCardID)

		// When the card exceeds the platform's size limit, fold the
		// old card to a collapsed "done" state, then start a fresh
		// one. Progressively trim blocks (last 1 → 0) until the
		// rebuilt card fits; runCard's internal thought cap guarantees
		// the card fits once blocks are empty.
		if runCardID != "" && cardTooLarge(sizer, card) {
			slog.Info("cardTooLarge triggered",
				"cardSize", cardSize,
				"thoughtLen", thoughtBuf.Len(),
				"blocks", len(blocks),
				"oldRunCardID", runCardID)
			pq.mark(runCardID, &channel.Card{
				Header: channel.CardHeader{Title: stageDone.title()},
				Color:  stageDone.color(),
				Panels: card.Panels,
				Fold:   channel.FoldCollapsed,
			})
			runCardID = ""
			for cardTooLarge(sizer, card) {
				if len(blocks) > 1 {
					blocks = blocks[len(blocks)-1:]
				} else if len(blocks) == 1 {
					blocks = nil
				} else {
					break
				}
				card = runCard(stage, thoughtBuf.String(), blocks, lastErr)
			}
			approvalMu.Lock()
			card.Approval = pendingApproval
			approvalMu.Unlock()
			if sizer != nil {
				cardSize = sizer.CardSize(card)
			}
			slog.Info("cardTooLarge rebuild",
				"newCardSize", cardSize,
				"thoughtLen", thoughtBuf.Len(),
				"blocks", len(blocks))
		}

		// When approval buttons are shown, the title reflects the wait
		// for a human decision instead of the tool-calling stage.
		if card.Approval != nil {
			card.Header.Title = "⏳ 等待审批"
		}

		if runCardID == "" {
			slog.Info("card create", "cardSize", cardSize)
			runCardID = pq.create(channel.ReplyMessage{Card: card})
		} else {
			pq.mark(runCardID, card)
		}
		lastTime = time.Now()
	}

	// maybeFlush throttles patches during streaming output.
	maybeFlush := func() {
		if time.Since(lastTime) >= 80*time.Millisecond {
			flushRunCard()
		}
	}

	// clearApproval removes the pending approval so the next flush omits
	// the button row. Called on stream events that indicate the approval
	// was resolved (the stream advanced past Ask).
	clearApproval := func() {
		approvalMu.Lock()
		pendingApproval = nil
		approvalMu.Unlock()
	}

	if updater != nil {
		updater.SetRunCardUpdater(sessionID, func(toolName, args, approvalID string) {
			approvalMu.Lock()
			pendingApproval = &channel.CardApproval{
				ToolName:   toolName,
				Args:       args,
				ApprovalID: approvalID,
			}
			approvalMu.Unlock()
			select {
			case approvalSig <- struct{}{}:
			default:
			}
		})
		defer updater.ClearRunCardUpdater(sessionID)
	}

loop:
	for {
		select {
		case evt, ok := <-stream:
			if !ok {
				break loop
			}
			slog.Debug("stream event",
				"type", evt.Type,
				"thoughtLen", thoughtBuf.Len(),
				"blocks", len(blocks))
			// Clear approval buttons only on events that indicate the
			// stream has advanced past the approval point (tool result,
			// text output, done/error). StreamThought and StreamToolCall
			// can be buffered in the stream channel while the event loop
			// is blocked in pq.create; processing them after the Ask
			// callback has set pendingApproval would wrongly clear the
			// approval buttons (race with pq.create blocking).
			switch evt.Type {
			case openagent.StreamToolResult, openagent.StreamTextDelta,
				openagent.StreamRetrying, openagent.StreamDone,
				openagent.StreamError, openagent.StreamAborted:
				clearApproval()
			}
			switch evt.Type {
			case openagent.StreamThought:
				stage = stageThinking
				thoughtBuf.WriteString(evt.Text)
				maybeFlush()

			case openagent.StreamTextDelta:
				stage = stageAnswering
				if len(blocks) == 0 || blocks[len(blocks)-1].tool != nil {
					blocks = append(blocks, block{text: evt.Text})
				} else {
					blocks[len(blocks)-1].text += evt.Text
				}
				maybeFlush()

			case openagent.StreamToolCall:
				stage = stageToolCalling
				for _, tc := range evt.Message.ToolCalls {
					switch tc.Function.Name {
					case "plan_create":
						goal, steps := parsePlanCreate(tc.Function.Arguments)
						if goal != "" {
							pq.create(channel.ReplyMessage{Card: mkCard("📋 "+goal, steps, channel.CardColorBlue)})
						}
						continue
					case "plan_update", "enter_plan_mode":
						pendingTool[tc.ID] = &tpend{name: tc.Function.Name}
						continue
					}
					pendingTool[tc.ID] = &tpend{name: tc.Function.Name}
					entry := toolCallEntry{
						name:       tc.Function.Name,
						args:       tc.Function.Arguments,
						status:     "in_progress",
						title:      opentool.ToolTitle(tc.Function.Name, tc.Function.Arguments),
						toolCallID: tc.ID,
					}
					blocks = append(blocks, block{tool: &entry})
				}
				flushRunCard()

			case openagent.StreamToolProgress:
				if _, ok := pendingTool[evt.ToolCallID]; !ok {
					continue
				}
				for i := len(blocks) - 1; i >= 0; i-- {
					if blocks[i].tool != nil && blocks[i].tool.toolCallID == evt.ToolCallID {
						blocks[i].tool.output += evt.Text
						break
					}
				}
				flushRunCard()

			case openagent.StreamToolResult:
				t, ok := pendingTool[evt.Message.ToolCallID]
				if !ok {
					continue
				}
				delete(pendingTool, evt.Message.ToolCallID)
				if t.name == "plan_update" || t.name == "enter_plan_mode" {
					continue
				}
				output := evt.Message.Content
				status := "completed"
				if strings.HasPrefix(output, "error: ") {
					status = "failed"
				}
				for i := len(blocks) - 1; i >= 0; i-- {
					if blocks[i].tool != nil && blocks[i].tool.toolCallID == evt.Message.ToolCallID {
						blocks[i].tool.status = status
						blocks[i].tool.output = output
						break
					}
				}
				flushRunCard()

			case openagent.StreamRetrying:
				if evt.Error != nil {
					lastErr = fmt.Sprintf("retrying: %v", evt.Error)
				} else {
					lastErr = "retrying..."
				}
				flushRunCard()

			case openagent.StreamDone:
				stage = stageDone
				flushRunCard()
				pq.flush()

			case openagent.StreamError:
				stage = stageDone
				if evt.Error != nil {
					lastErr = fmt.Sprintf("error: %v", evt.Error)
				}
				flushRunCard()
				pq.stop()
				return

			case openagent.StreamAborted:
				stage = stageDone
				lastErr = "aborted"
				flushRunCard()
				pq.stop()
				return
			}

		case <-approvalSig:
			// Flush the run card with the approval section. If the card
			// hasn't been created yet, skip — the next stream event will
			// create it with the approval section already set (avoids
			// flashing an empty card with just buttons).
			if runCardID != "" {
				flushRunCard()
			}
		}
	}

	// Stream closed without StreamDone/Error/Aborted — finalize the card so
	// the title doesn't freeze on "回答中". If a terminal event already
	// set stageDone this is a harmless re-mark.
	clearApproval()
	if runCardID != "" && stage != stageDone {
		stage = stageDone
		flushRunCard()
	}
	pq.stop()
}

// ── Run card ──

// cardTooLarge reports whether the card's serialized size exceeds maxCardBytes.
// Returns false when sizer is nil (no platform size limit).
func cardTooLarge(sizer CardSizer, card *channel.Card) bool {
	if sizer == nil {
		return false
	}
	return sizer.CardSize(card) > maxCardBytes
}

// mkCard is a plain card builder used for standalone cards (plan_create).
func mkCard(title, body string, color channel.CardColor) *channel.Card {
	return &channel.Card{Header: channel.CardHeader{Title: title}, Content: body, Color: color}
}

// stage tracks the agent run's progress for the run card title.
type stage int

const (
	stageThinking    stage = iota // 🤔 思考中
	stageToolCalling              // 🔧 调用工具中
	stageAnswering                // 💬 回答中
	stageDone                     // ✅ 已完成
)

func (s stage) title() string {
	switch s {
	case stageThinking:
		return "🤔 思考中"
	case stageToolCalling:
		return "🔧 调用工具中"
	case stageAnswering:
		return "💬 回答中"
	case stageDone:
		return "✅ 已完成"
	}
	return "🤔 思考中"
}

func (s stage) color() channel.CardColor {
	if s == stageDone {
		return channel.CardColorGrey
	}
	return channel.CardColorYellow
}

// runCard builds the single card for an agent run. The body interleaves
// in event-arrival order: thinking (collapsed) → [text → tool call
// (collapsed)]* → text. Empty segments are omitted. errMsg, when set,
// is appended as a final text segment.
//
// thought is capped to the last maxThoughtDisplay bytes so a reasoning
// model's long chain-of-thought can't blow past the card size limit — the
// buffer itself is untouched (session persistence keeps the full trace).
func runCard(s stage, thought string, blks []block, errMsg string) *channel.Card {
	var panels []channel.Card

	if thought != "" {
		if len(thought) > maxThoughtDisplay {
			thought = "...(已截断)\n" + thought[len(thought)-maxThoughtDisplay:]
		}
		panels = append(panels, channel.Card{
			Content: thought,
			Fold:    channel.FoldCollapsed,
		})
	}

	var lastText string
	for _, b := range blks {
		if b.tool != nil {
			if lastText != "" {
				panels = append(panels, channel.Card{Content: lastText, Fold: channel.FoldNone})
				lastText = ""
			}
			panels = append(panels, toolCallSubCard(*b.tool))
		} else {
			lastText += b.text
		}
	}
	if errMsg != "" {
		lastText += "\n\n" + errMsg
	}
	if lastText != "" {
		if len(lastText) > maxThoughtDisplay {
			lastText = lastText[len(lastText)-maxThoughtDisplay:] + "\n...(已截断)"
		}
		panels = append(panels, channel.Card{Content: lastText, Fold: channel.FoldNone})
	}

	fold := channel.FoldNone
	if s == stageDone {
		fold = channel.FoldExpanded
	}
	return &channel.Card{
		Header: channel.CardHeader{Title: s.title()},
		Color:  s.color(),
		Fold:   fold,
		Panels: panels,
	}
}

// ── Tool card ──

// toolCallEntry is the per-call state collected for the run card.
type toolCallEntry struct {
	name       string
	args       string
	status     string // "in_progress" | "completed" | "failed"
	output     string
	title      string // human-readable title from opentool.ToolTitle; empty = fall back to name
	toolCallID string // for lookup during StreamToolProgress/Result
}

// block is one segment of the run card body, preserving the interleaved
// order of agent text and tool calls (mimicking SSE-style rendering).
type block struct {
	text string         // agent output segment
	tool *toolCallEntry // tool call segment
}

// toolCallSubCard builds the inner collapsed Card for one tool call.
// Title is the tool title (with bold first word) + status marker; body is
// input + output.
func toolCallSubCard(e toolCallEntry) channel.Card {
	title := e.title
	if title == "" {
		title = e.name
	}
	title = boldFirstWord(title)
	switch e.status {
	case "completed":
		title += " ✓"
	case "failed":
		title += " ✗"
	}

	body := formatInput(e.name, e.args)
	if e.output != "" {
		out := e.output
		if r := []rune(out); len(r) > maxCardBytes/4 {
			out = string(r[:maxCardBytes/4-3]) + "..."
		}
		body += "\n" + channel.CodeBlock(out)
	}

	return channel.Card{
		Header:  channel.CardHeader{Title: title, TitleMarkdown: true},
		Content: body,
		Fold:    channel.FoldCollapsed,
	}
}

func formatInput(name, args string) string {
	m := jsonMap(args)
	switch name {
	case "shell", "terminal_create":
		cmd := jsonStr(m, "command")
		if cmd != "" {
			return channel.CodeBlock(trunc(cmd, 500))
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
	case "websearch":
		if q := jsonStr(m, "query"); q != "" {
			return "`" + q + "`"
		}
	case "webfetch":
		if u := jsonStr(m, "url"); u != "" {
			return "`" + u + "`"
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
	case "feishu_sendfile":
		if path := jsonStr(m, "path"); path != "" {
			return "`" + filepath.Base(path) + "`"
		}
	case "wecom_sendfile":
		if path := jsonStr(m, "path"); path != "" {
			return "`" + filepath.Base(path) + "`"
		}
	}
	return channel.CodeBlock(trunc(args, 200))
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
	case "websearch":
		return "🌐"
	case "webfetch":
		return "🔗"
	case "recall":
		return "🧠"
	case "load_skill":
		return "📦"
	case "feishu_sendfile":
		return "📎"
	case "wecom_sendfile":
		return "📎"
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
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "..."
	}
	return s
}

// boldFirstWord wraps the first whitespace-delimited word in ** for
// lark_md bold, leaving the rest of the string unchanged.
func boldFirstWord(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	i := strings.IndexByte(s, ' ')
	if i < 0 {
		return "**" + s + "**"
	}
	return "**" + s[:i] + "**" + s[i:]
}
