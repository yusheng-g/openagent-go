// Package feishu implements channel.Channel for Feishu (Lark) via
// WebSocket long connection using the official larksuite SDK v3.
//
// The SDK handles authentication, keep-alive, reconnection, and event
// deserialization automatically. This package only needs to register
// event handlers and normalize messages for the Agent.
package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkapplication "github.com/larksuite/oapi-sdk-go/v3/service/application/v6"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/yusheng-g/openagent-go/channel"
	"github.com/yusheng-g/openagent-go/governance"
)

// leadingMentionRe matches one or more consecutive Feishu @-mention
// placeholders at the start of a message (e.g. "@_user_1 @_user_2").
// In group chats, Feishu embeds these opaque tokens in the text field
// instead of human-readable "@name". They must be stripped so that
// slash commands like "/mode" are recognized and the agent doesn't
// receive rendering artifacts as user input.
var leadingMentionRe = regexp.MustCompile(`^(@_user_\d+\s*)+`)

// Channel implements channel.Channel for Feishu via WebSocket long connection.
type Channel struct {
	appID     string
	appSecret string

	client   *lark.Client
	ws       *larkws.Client
	once     sync.Once
	approver *feishuApprover

	// Per-chat mode state ("manual" | "auto"). defMode is the fallback
	// for chats that have never switched; it comes from config
	// DefaultMode (defaults to "manual").
	modeMu  sync.RWMutex
	modes   map[string]string
	defMode string

	// onReady is invoked by the SDK once the WebSocket is connected and
	// ready to receive messages (nil = ignore). Used by the connection
	// manager to flip its status from connecting → connected.
	onReady        func()
	onReconnecting func()
	// onError is invoked by the SDK on every connection failure (including
	// failed reconnects). Used by the manager to surface first-connect
	// failures (e.g. bad credentials) instead of a silent reconnect loop.
	onError func(err error)

	// connectedOnce is set when the SDK has connected at least once (the
	// Start goroutine has entered its permanent select{}). It gates the
	// leak counter: only a Start that actually reached the select{} leaks
	// goroutines — a cancel during the connect/reconnect phase lets the
	// SDK goroutine return normally, and counting it would be a false
	// positive.
	connectedOnce atomic.Bool
}

// New returns a Feishu Channel. defaultMode is the initial mode for chats
// that haven't explicitly switched ("manual" or "auto"; empty = "manual").
// The Channel must be started via Start() to begin receiving messages.
func New(appID, appSecret, defaultMode string) *Channel {
	if !channel.IsValidMode(defaultMode) {
		defaultMode = channel.ModeManual
	}
	return &Channel{
		appID:     appID,
		appSecret: appSecret,
		approver:  newFeishuApprover(),
		modes:     make(map[string]string),
		defMode:   defaultMode,
	}
}

// GetMode returns the current mode for a chat ("manual" | "auto").
// Chats that have never switched return the default mode.
func (c *Channel) GetMode(chatID string) string {
	c.modeMu.RLock()
	defer c.modeMu.RUnlock()
	if m, ok := c.modes[chatID]; ok {
		return m
	}
	return c.defMode
}

// SetMode updates the mode for a chat.
func (c *Channel) SetMode(chatID, mode string) {
	if !channel.IsValidMode(mode) {
		return
	}
	c.modeMu.Lock()
	defer c.modeMu.Unlock()
	c.modes[chatID] = mode
}

// BuildModeCard returns a platform-neutral Card for the mode-switching
// UI. The current mode for the chat is highlighted. Used by the channel
// handler when the user sends /mode.
func (c *Channel) BuildModeCard(chatID string) *channel.Card {
	return &channel.Card{
		Header:     channel.CardHeader{Title: "模式切换"},
		Color:      channel.CardColorBlue,
		Content:    modeCardContent(c.GetMode(chatID)),
		ModeSwitch: &channel.CardModeSwitch{CurrentMode: c.GetMode(chatID)},
	}
}

// modeCardContent returns the markdown body for the mode card.
func modeCardContent(currentMode string) string {
	return fmt.Sprintf(
		"选择 agent 的工作模式：\n\n"+
			modeDescription+"\n\n"+
			"当前模式：**%s**",
		modeLabel(currentMode),
	)
}

// Approver returns the Feishu-card-based human approver for this channel.
// May be called before Start; the lark client is wired in Start.
func (c *Channel) Approver(mem governance.ApprovalMemory) governance.HumanApprover {
	c.approver.memory = mem
	return c.approver
}

// SetRunCardUpdater registers a callback that Ask invokes to embed approval
// buttons in the run card instead of sending a separate approval card.
// streamReply calls this before each agent run and clears it after.
func (c *Channel) SetRunCardUpdater(sessionID string, cb func(toolName, args, approvalID string)) {
	c.approver.setRunCardUpdater(sessionID, cb)
}

// ClearRunCardUpdater removes the run-card update callback for a session.
func (c *Channel) ClearRunCardUpdater(sessionID string) {
	c.approver.clearRunCardUpdater(sessionID)
}

// SetOnReady registers the connection-ready callback (nil clears it).
// Fired on every successful WebSocket connection, including reconnects.
func (c *Channel) SetOnReady(f func()) { c.onReady = f }

// SetOnReconnecting registers the reconnecting callback (nil clears it).
// Fired when the SDK loses the connection and starts auto-reconnecting.
func (c *Channel) SetOnReconnecting(f func()) { c.onReconnecting = f }

// SetOnError registers the connection-error callback (nil clears it).
// Fired on every failed connection attempt, including reconnects.
func (c *Channel) SetOnError(f func(err error)) { c.onError = f }

// Name implements channel.Channel.
func (c *Channel) Name() string { return "feishu" }

// Start implements channel.Channel. It opens a WebSocket connection to
// Feishu and blocks until ctx is cancelled.
func (c *Channel) Start(ctx context.Context, handler channel.MessageHandler) error {
	c.client = lark.NewClient(c.appID, c.appSecret)
	c.approver.client = c.client

	dh := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			msg := toIncoming(event)
			if msg == nil {
				return nil
			}
			handler(ctx, *msg, c.buildReply(ctx, event))
			return nil
		}).
		OnP2CardActionTrigger(func(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
			return c.handleCardAction(ctx, event)
		}).
		// Silently accept non-message events to avoid error spam.
		OnP2ChatAccessEventBotP2pChatEnteredV1(func(ctx context.Context, event *larkim.P2ChatAccessEventBotP2pChatEnteredV1) error {
			return nil
		}).
		OnP2BotMenuV6(func(ctx context.Context, event *larkapplication.P2BotMenuV6) error {
			return nil
		})

	c.ws = larkws.NewClient(c.appID, c.appSecret,
		larkws.WithEventHandler(dh),
		larkws.WithLogLevel(larkcore.LogLevelError),
	)
	c.ws.SetOnReady(func() {
		// Gates the leak counter (see connectedOnce) — the SDK Start
		// goroutine has entered its permanent select{} only after the
		// first successful connection.
		c.connectedOnce.Store(true)
		if c.onReady != nil {
			c.onReady()
		}
	})
	if c.onReconnecting != nil {
		c.ws.SetOnReconnecting(c.onReconnecting)
	}
	if c.onError != nil {
		c.ws.SetOnError(c.onError)
	}

	// The SDK's Start blocks in `select {}` once connected and NEVER
	// returns — not on ctx cancellation, not on Close(). Run it in a
	// goroutine and manage the lifecycle here: on ctx.Done we close the
	// SDK client (which at least tears down the connection) and return.
	//
	// KNOWN LEAK (upstream SDK design): once connected, each Start leaks
	// TWO permanent goroutines — the one parked in `select {}` and the
	// SDK's pingLoop (a `for { time.Sleep }` with no exit path, which
	// even restarts itself on panic). Both close over the ws.Client, so
	// the whole Channel object graph stays reachable. Disconnect+
	// reconnect (e.g. applying new credentials) therefore accumulates
	// leaks without bound; the counter below makes the growth observable.
	sdkDone := make(chan error, 1)
	go func() { sdkDone <- c.ws.Start(ctx) }()

	select {
	case err := <-sdkDone:
		// The SDK returned on its own: a first-connect ClientError (bad
		// credentials) fails fast; anything else surfaces here too.
		// If the context was cancelled at the same time (a disconnect
		// during the connect/reconnect phase — the only window where
		// both branches can be ready), the SDK's error is a byproduct of
		// our own cancel: report a clean shutdown, not a fake LastError.
		if ctx.Err() != nil {
			return nil
		}
		return err
	case <-ctx.Done():
		c.ws.Close()
		c.recordSdkLeak()
		return nil
	}
}

// Stop implements channel.Channel. Closes the underlying SDK connection,
// which makes the SDK's receiveMessageLoop exit (the channel.Channel
// contract "no further handler calls after Stop" holds) — but it does
// NOT terminate the SDK's Start goroutine (see the leak note above); the
// Start context is the only thing that ends the Start lifecycle.
func (c *Channel) Stop() error {
	if c.ws != nil {
		c.ws.Close()
	}
	// Same leak accounting as the ctx path — Stop terminates the same
	// parked goroutines when the SDK had connected. Gated by
	// connectedOnce so a Stop before the first connect (nothing leaked)
	// or racing a connect-phase cancel is not double-counted.
	c.recordSdkLeak()
	return nil
}

// sdkLeakCount counts Start terminations that leak the SDK's select{}
// and pingLoop goroutines (gated by connectedOnce — a Start that never
// reached the permanent select{} returns normally and leaks nothing).
// Exposed via the warn in recordSdkLeak so unbounded growth is
// observable instead of silent.
var (
	sdkLeakCount  atomic.Int64
	sdkLeakMu     sync.Mutex
	sdkLeakWarned bool
)

func (c *Channel) recordSdkLeak() {
	if !c.connectedOnce.Load() {
		return // never reached the select{} — the SDK goroutine returned
	}
	n := sdkLeakCount.Add(1)
	// Warn once the growth is clearly beyond incidental (the manager's
	// documented disconnect+reconnect flows for applying credentials).
	sdkLeakMu.Lock()
	defer sdkLeakMu.Unlock()
	if n >= sdkLeakWarnThreshold && !sdkLeakWarned {
		sdkLeakWarned = true
		slog.Warn("feishu: SDK Start does not return on close — each disconnect leaks 2 goroutines "+
			"(select{} + pingLoop) and the client object graph; consider restarting the process periodically",
			"disconnects", n)
	}
}

// sdkLeakWarnThreshold is the disconnect count after which the leak is
// surfaced (incidental reconnects stay quiet; sustained churn is warned).
const sdkLeakWarnThreshold = 10

// ── Normalization ──

// toIncoming converts a Feishu message event to a channel.IncomingMessage.
// Returns nil if the message should be ignored.
func toIncoming(event *larkim.P2MessageReceiveV1) *channel.IncomingMessage {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}
	msg := event.Event.Message

	text := stripLeadingMentions(extractText(msg))
	if strings.TrimSpace(text) == "" {
		return nil
	}

	chatType := "private"
	if msg.ChatType != nil {
		chatType = *msg.ChatType
	}

	chatID := ""
	if msg.ChatId != nil {
		chatID = *msg.ChatId
	}

	userID, userName := extractSender(event)

	msgID := ""
	if msg.MessageId != nil {
		msgID = *msg.MessageId
	}

	var mentions []string
	if msg.Mentions != nil {
		for _, m := range msg.Mentions {
			if m.Id != nil {
				if m.Id.OpenId != nil {
					mentions = append(mentions, *m.Id.OpenId)
				} else if m.Id.UnionId != nil {
					mentions = append(mentions, *m.Id.UnionId)
				} else if m.Id.UserId != nil {
					mentions = append(mentions, *m.Id.UserId)
				}
			}
		}
	}

	return &channel.IncomingMessage{
		ID:       msgID,
		ChatID:   chatID,
		ChatType: chatType,
		UserID:   userID,
		UserName: userName,
		Text:     text,
		Mentions: mentions,
		Raw:      event,
	}
}

// extractSender pulls user ID and display name from the event sender.
func extractSender(event *larkim.P2MessageReceiveV1) (string, string) {
	if event.Event.Sender == nil || event.Event.Sender.SenderId == nil {
		return "", ""
	}
	sid := event.Event.Sender.SenderId
	switch {
	case sid.OpenId != nil:
		return *sid.OpenId, *sid.OpenId
	case sid.UnionId != nil:
		return *sid.UnionId, *sid.UnionId
	case sid.UserId != nil:
		return *sid.UserId, *sid.UserId
	}
	return "", ""
}

// extractText pulls plain text from a Feishu EventMessage.
// EventMessage.Content holds a JSON string with the actual message body.
// For text messages the JSON shape is {"text":"..."}.
func extractText(msg *larkim.EventMessage) string {
	if msg == nil || msg.Content == nil {
		return ""
	}
	raw := *msg.Content
	if raw == "" {
		return ""
	}

	// Try to extract the "text" key from the JSON content.
	var body struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(raw), &body); err == nil && body.Text != "" {
		return strings.TrimSpace(body.Text)
	}

	// Fallback: return the raw content (might be non-text message).
	return strings.TrimSpace(raw)
}

// stripLeadingMentions removes Feishu @-mention placeholders (@_user_N)
// from the beginning of text. Only leading mentions are stripped so that
// inline mentions (e.g. "tell @_user_1 to review") are preserved for the
// agent to see. Returns empty string when text contains only mentions.
func stripLeadingMentions(text string) string {
	t := strings.TrimSpace(text)
	t = leadingMentionRe.ReplaceAllString(t, "")
	return strings.TrimSpace(t)
}

// ── Reply ──

// buildReply returns a channel.ReplyFunc that sends a message back
// to the same chat. If UpdateID is set the channel patches the existing
// card; otherwise a new message is sent. Returns the platform message ID.
func (c *Channel) buildReply(ctx context.Context, event *larkim.P2MessageReceiveV1) channel.ReplyFunc {
	return func(replyCtx context.Context, msg channel.ReplyMessage) (string, error) {
		if c.client == nil {
			return "", fmt.Errorf("feishu: client not initialized")
		}

		received := event.Event.Message
		if received == nil {
			return "", fmt.Errorf("feishu: no message in event")
		}

		receiveIDType, receiveID := resolveReceive(received, event)
		if receiveID == "" {
			return "", fmt.Errorf("feishu: cannot determine receive ID")
		}

		// Card update: patch existing card.
		if msg.UpdateID != "" && msg.Card != nil {
			cardJSON, err := BuildCard(msg.Card)
			if err != nil {
				return "", err
			}
			return msg.UpdateID, c.patchCard(replyCtx, msg.UpdateID, cardJSON)
		}

		if msg.Card != nil {
			return c.sendCard(replyCtx, receiveIDType, receiveID, msg.Card)
		}
		return c.sendText(replyCtx, receiveIDType, receiveID, msg.Text)
	}
}

func resolveReceive(msg *larkim.EventMessage, event *larkim.P2MessageReceiveV1) (receiveIDType, receiveID string) {
	if msg.ChatType != nil && *msg.ChatType == "group" && msg.ChatId != nil {
		receiveIDType = "chat_id"
		receiveID = *msg.ChatId
	} else {
		receiveIDType = "open_id"
		userID, _ := extractSender(event)
		receiveID = userID
	}
	return
}

func (c *Channel) sendText(ctx context.Context, receiveIDType, receiveID, text string) (string, error) {
	if text == "" {
		return "", nil
	}
	content := fmt.Sprintf(`{"text":"%s"}`, escapeJSON(text))
	return c.sendMessage(ctx, receiveIDType, receiveID, "text", content)
}

// escapeJSON escapes a string for safe embedding in a JSON string.
func escapeJSON(s string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
		"\r", `\r`,
		"\t", `\t`,
	).Replace(s)
}

func (c *Channel) sendCard(ctx context.Context, receiveIDType, receiveID string, card *channel.Card) (string, error) {
	cardJSON, err := BuildCard(card)
	if err != nil {
		return "", fmt.Errorf("feishu: build card: %w", err)
	}
	if cardJSON == "" {
		return "", nil
	}
	return c.sendMessage(ctx, receiveIDType, receiveID, "interactive", cardJSON)
}

func (c *Channel) sendMessage(ctx context.Context, receiveIDType, receiveID, msgType, content string) (string, error) {
	return createMessage(ctx, c.client, receiveIDType, receiveID, msgType, content)
}

// createMessage sends a message via the Feishu IM API and returns the
// platform-assigned message ID. Shared by Channel.sendMessage and
// feishuApprover.sendInteractive.
func createMessage(ctx context.Context, client *lark.Client, receiveIDType, receiveID, msgType, content string) (string, error) {
	resp, err := client.Im.Message.Create(ctx,
		larkim.NewCreateMessageReqBuilder().
			ReceiveIdType(receiveIDType).
			Body(larkim.NewCreateMessageReqBodyBuilder().
				MsgType(msgType).
				ReceiveId(receiveID).
				Content(content).
				Build()).
			Build())
	if err != nil {
		return "", fmt.Errorf("feishu: send message: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("feishu: send message failed: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data != nil && resp.Data.MessageId != nil {
		return *resp.Data.MessageId, nil
	}
	return "", nil
}

// patchCard updates an existing interactive card message by message ID.
// https://open.feishu.cn/document/server-docs/im-v1/message/patch
func (c *Channel) patchCard(ctx context.Context, messageID, cardJSON string) error {
	return patchMessageCard(ctx, c.client, messageID, cardJSON)
}

// patchMessageCard patches an existing interactive card message.
// Shared by Channel.patchCard and feishuApprover.patchResolved.
func patchMessageCard(ctx context.Context, client *lark.Client, messageID, cardJSON string) error {
	resp, err := client.Im.Message.Patch(ctx,
		larkim.NewPatchMessageReqBuilder().
			MessageId(messageID).
			Body(larkim.NewPatchMessageReqBodyBuilder().
				Content(cardJSON).
				Build()).
			Build())
	if err != nil {
		return fmt.Errorf("feishu: patch card: %w", err)
	}
	if !resp.Success() {
		slog.Warn("feishu: patch card failed", "code", resp.Code, "msg", resp.Msg, "cardSize", len(cardJSON))
		return fmt.Errorf("feishu: patch card failed: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

// CardSize returns the serialized JSON size of a card for size-limit
// checks. Implements the server.CardSizer interface.
func (c *Channel) CardSize(card *channel.Card) int {
	b, err := BuildCard(card)
	if err != nil {
		return 0
	}
	return len(b)
}

// ── Card action routing ──

// handleCardAction routes a Feishu card action trigger event by the
// "type" field in the action value. "mode_switch" is handled inline;
// all other actions (approval buttons) are delegated to
// handleApprovalAction.
func (c *Channel) handleCardAction(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
	if event == nil || event.Event == nil || event.Event.Action == nil {
		return c.approver.handleCardAction(ctx, event)
	}

	actionType, _ := event.Event.Action.Value["type"].(string)
	switch actionType {
	case "mode_switch":
		return c.handleModeSwitch(ctx, event)
	default:
		return c.handleApprovalAction(ctx, event)
	}
}

// handleApprovalAction resolves the pending approval via the approver.
// The approver handles allow_always remembering internally.
func (c *Channel) handleApprovalAction(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
	return c.approver.handleCardAction(ctx, event)
}

// handleModeSwitch processes a mode-switch button click. It extracts the
// target mode and chat ID from the event, updates the per-chat mode
// state, patches the card to show the result, and returns a toast.
func (c *Channel) handleModeSwitch(_ context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
	action := event.Event.Action
	mode, _ := action.Value["mode"].(string)
	if !channel.IsValidMode(mode) {
		return toastResponse("error", "未知模式: "+mode), nil
	}

	chatID := ""
	if event.Event.Context != nil {
		chatID = event.Event.Context.OpenChatID
	}
	if chatID == "" {
		return toastResponse("error", "无法识别聊天ID"), nil
	}

	c.SetMode(chatID, mode)

	// Patch the card to show the resolved state (fire-and-forget).
	go c.patchModeCardResolved(event, mode)

	return toastResponse("success", "已切换到 "+modeLabel(mode)+" 模式"), nil
}

// patchModeCardResolved updates the mode-switch card in-place to reflect
// the new current mode. Failures are non-critical (the toast already
// confirmed the switch).
func (c *Channel) patchModeCardResolved(event *callback.CardActionTriggerEvent, mode string) {
	if c.client == nil || event.Event.Context == nil {
		return
	}
	msgID := event.Event.Context.OpenMessageID
	if msgID == "" {
		return
	}
	cardJSON, err := buildModeCardResolved(mode)
	if err != nil {
		slog.Warn("feishu: build mode resolved card", "error", err)
		return
	}
	if err := c.patchCard(context.Background(), msgID, cardJSON); err != nil {
		slog.Warn("feishu: patch mode card", "error", err)
	}
}
