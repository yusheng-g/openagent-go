package chat

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/gausszhou/gruff/gruff"

	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/components"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/layout"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/theme"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/utils"
)

// This package implements the TUI chat page: welcome screen, input, ACP
// backend connection (in-process via io.Pipe), streaming agent responses,
// and transcript rendering.

type FocusArea int

const (
	FocusChat FocusArea = iota
)

const (
	defaultWidth  = 80
	defaultHeight = 30
)

const (
	PlaceholderPrefix = "Ask anything ... e.g. "
	PlaceholderSuffix = " (Tab to accept)"
)

// Model is the chat page model. It is deliberately render-only: no ACP client,
// no event loop, no input history. NewModel takes plain parameters so the TUI
// can boot standalone without a backend.
type Model struct {
	width  int
	height int

	ctx    context.Context
	cancel context.CancelFunc

	workDir string
	version string

	focus FocusArea

	activeSessionID string

	// inChat controls welcome vs chat view. Set true on first Enter so the
	// view switches immediately without waiting for the ACP session ID.
	inChat bool

	// mode is the current session mode ("auto" | "manual" | "plan"). Shown as
	// a badge in the input header; manual is the server default.
	mode string

	// logoColor / logoGradient drive the welcome-page logo coloring from
	// settings.json. Gradient (2+ stops) wins over single color.
	logoColor    string
	logoGradient []string

	messages []ChatMessage

	spinner components.Loading
	loading bool

	statusBar components.StatusBar

	turnId int64

	// ACP backend connection (injected by app.go's startACPInProcess goroutine)
	acpSession *openacp.Session
	program    *tea.Program

	// permission dialog: when non-nil, a tool call is awaiting approval.
	// The user's selection is sent back via permissionReplyCh.
	permissionReq         *openacp.RequestPermissionRequest
	permissionReplyCh     chan openacp.RequestPermissionResponse
	permissionSelectedIdx int

	chatViewport viewport.Model
	chatTextarea textarea.Model

	viewportDirty bool
	textareaDirty bool

	// renderPending marks that transcript content changed while the agent
	// was streaming and the throttled viewport flush has not run yet.
	// flushPending is true while a flush timer is in flight, so concurrent
	// chunks collapse into a single rebuild (9.2 render throttling).
	renderPending bool
	flushPending  bool

	statusText string

	// notifyMsg is a transient success toast (e.g. "Session created"). It
	// auto-clears after notifyDuration; statusText remains the persistent
	// status line.
	notifyMsg string

	// Token usage (usage_update) and prompt count, shown in the right
	// sidebar: usedTokens is the session's consumed context, contextSize
	// the model's window, promptCount the prompts sent in the current
	// session (live sends plus replayed history).
	usedTokens  int
	contextSize int
	promptCount int

	needAutoScroll bool

	// input cursor blink
	blinkCount int
	blink      bool

	tips       string
	suggestion string

	modelId    string
	providerId string

	// command panel (Ctrl+P or a "/" input). When panelOpen, key input is
	// routed to the panel: ↑/↓ select, enter executes, esc closes, printed
	// chars filter by slash prefix. panelMode picks which list is shown.
	panelOpen   bool
	panelFilter string
	panelIdx    int
	panelMode   int
	// panelFromSlash remembers whether the command palette was opened by
	// typing "/" (slash-command card) or by Ctrl+P (command-panel design).
	panelFromSlash bool

	// sessions/models panel backing data.
	sessionItems    []sessionItem
	sessionsLoading bool
	configOptions   []openacp.SessionConfigOption

	// pendingModelsPanel remembers that /models was requested before any
	// session existed (lazy boot): the picker opens when the session-less
	// config options arrive from session/list_config_options.
	pendingModelsPanel bool

	// pendingConfigSet holds a config option (model, mode) picked from a
	// picker while no session existed yet: the option applies to a session,
	// so it is applied via set_config_option once the on-demand session
	// creation lands (newSessionMsg).
	pendingConfigSet *configPick

	// configPickerID is the config option id ("mode", "thought_level", ...)
	// the open config picker edits; set by openConfigPanel.
	configPickerID string

	// replaying is true while LoadSession replays history into the
	// transcript; user input is held until it completes.
	replaying bool

	// planEntries is the agent's latest plan (sessionUpdate "plan"),
	// rendered as the TODO list in the right sidebar.
	planEntries []openacp.PlanEntry

	// searchResults holds the transcript message indices matching the
	// search panel's current query (panelFilter).
	searchResults []int

	// renderCache memoizes each message's styled block (增量渲染): streaming
	// re-styles only the chunk that changed, and toggles/width changes are
	// picked up by the fingerprint. Keyed by message index; entries are
	// dropped automatically once they exceed the message count.
	renderCache map[int]renderCacheEntry

	// fedOffset/fedHeight remember the window the viewport content was last
	// built for, so the virtual scroll refeeds (styling newly revealed
	// messages) whenever the user scrolls the window elsewhere.
	fedOffset int
	fedHeight int

	// geom caches render geometry computed during Update so View can render
	// without mutating model state (see layout.go).
	geom layoutSnap

	// input history (↑/↓ browses on single-line input).
	history    []string
	historyIdx int // == len(history) means fresh, un-browsed input

	// inputQueue holds user messages that arrived while a prompt was still
	// running (max maxInputQueue); they are sent in order when prompt_done
	// arrives.
	inputQueue []string

	// transcript display toggles, driven by /toggle_* commands.
	visibleConfig components.VisibleConfig

	// splitView toggles the two-pane transcript layout (/split).
	splitView bool

	// themeIdx is the active color preset index (/theme, see presets.go).
	themeIdx int

	// pluginItems is the installed-plugin list for the plugins panel.
	pluginItems []string

	// exportNotice is the /export outcome shown in the dismiss-only export
	// dialog (the written path, or the reason nothing was written).
	exportNotice string
}

// panelMode values for the floating overlay.
const (
	panelModeCommand = iota
	panelModeSessions
	panelModeModels
	panelModeHelp
	panelModeSearch
	panelModeEdit
	panelModePlugins
	panelModeConfig
	panelModeExport
)

// maxHistory caps the input history ring.
const maxHistory = 100

// maxInputQueue caps how many user messages may wait while the agent is
// still busy with an earlier prompt.
const maxInputQueue = 3

// notifyDuration is how long a transient toast stays visible.
const notifyDuration = 2500 * time.Millisecond

// Render-cost budgets (TUI_Features 9.2/9.3): the viewport flush cadence
// grows with transcript size, tool output folds after a few lines, and the
// rendered document is capped so a long session never rebuilds a huge
// string on every flush.
const (
	baseRenderInterval     = 100 * time.Millisecond
	baseHighlightInterval  = 50 * time.Millisecond // reserved for highlight paths
	contentThrottleStep100 = 100 * 1024            // > 100KB → 150ms
	contentThrottleStep500 = 500 * 1024            // > 500KB → 200ms
	maxRenderChars         = 1_000_000
	defaultToolOutputLines = 5

	// maxStoredChars bounds how much transcript is held in memory at all
	// (2x the render budget); older messages are dropped so a very long
	// session never accumulates unbounded history (17.2 lazy history).
	maxStoredChars = 2 * maxRenderChars
)

// sessionItem is one row of the sessions panel.
type sessionItem struct {
	id    string
	title string
	// updated is the session's last-update time as an RFC3339 string, as
	// carried by session/list; empty when the agent didn't provide one.
	updated string
}

// configPick is a config-option change (id + value) waiting for a session
// to apply it to.
type configPick struct {
	id    string
	value string
}

// cmdAction is the executable behaviour bound to a slash command. Commands
// that need panel UIs (sessions/models/skills) are wired in later phases and
// currently report "not implemented yet".
type cmdAction int

const (
	actionNone cmdAction = iota
	actionExit
	actionNew
	actionSessions
	actionModels
	actionUpdateSkills
	actionToggleThinking
	actionToggleSkill
	actionToggleShell
	actionToggleToolDetail
	actionToggleMode
	actionThoughtLevel
	actionToggleLineNumbers
	actionHelp
	actionSearch
	actionExport
	actionEdit
	actionTheme
	actionPlugins
	actionSplit
)

// panelCommand is a slash-command entry for the command panel.
type panelCommand struct {
	slash   string
	title   string
	action  cmdAction
	enabled bool
	space   bool // toggles with the space key when the filter is empty
}

// allPanelCommands is the static command registry. Commands whose
// functionality has not landed yet stay out of the list entirely (parked:
// /update_skills is still the "not implemented" stub, /plugins only listed
// a directory nothing consumes) — re-enable by adding a line here; the
// executeCommand handlers are kept.
func allPanelCommands() []panelCommand {
	return []panelCommand{
		{"/sessions", "Switch session", actionSessions, true, false},
		{"/new", "New session", actionNew, true, false},
		{"/models", "Switch model", actionModels, true, false},
		{"/toggle_mode", "Switch mode", actionToggleMode, true, false},
		{"/thought_level", "Switch thought level", actionThoughtLevel, true, false},
		{"/toggle_thinking", "Expand thinking content", actionToggleThinking, true, true},
		{"/toggle_skill", "Toggle skill tools", actionToggleSkill, true, true},
		{"/toggle_shell", "Toggle shell tools", actionToggleShell, true, true},
		{"/toggle_toolcall", "Toggle tool call detail", actionToggleToolDetail, true, true},
		{"/toggle_linenumbers", "Toggle input line numbers", actionToggleLineNumbers, true, true},
		{"/help", "Show help", actionHelp, true, false},
		{"/search", "Search transcript", actionSearch, true, false},
		{"/export", "Export transcript to Markdown", actionExport, true, false},
		{"/edit", "Edit a past user message", actionEdit, true, false},
		{"/theme", "Cycle color theme", actionTheme, true, true},
		{"/split", "Toggle split view", actionSplit, true, false},
		{"/exit", "Exit the app", actionExit, true, false},
	}
}

// toggleIcon reports the ○/● state for a toggle command.
func (m *Model) toggleIcon(slash string) string {
	on := false
	switch slash {
	case "/toggle_thinking":
		on = m.visibleConfig.ExpandThinking
	case "/toggle_skill":
		on = m.visibleConfig.ShowToolSkill
	case "/toggle_shell":
		on = m.visibleConfig.ShowToolShell
	case "/toggle_toolcall":
		on = m.visibleConfig.ShowToolDetail
	case "/toggle_linenumbers":
		on = m.chatTextarea.ShowLineNumbers
	case "/theme":
		return string(themePresets[m.themeIdx%len(themePresets)].name[0])
	}
	if on {
		return "●"
	}
	return "○"
}

// ChatMessage is a transcript message record.
type ChatMessage struct {
	Role    string
	Content string
	TurnId  int64

	// thought timing (Role == "thought"): ThoughtStart is stamped when the
	// first chunk arrives, ThoughtEnd when thinking gives way to another
	// message or the turn finishes. Zero values mean the span is unknown —
	// history replayed from a session carries no measurable times.
	ThoughtStart time.Time
	ThoughtEnd   time.Time

	// tool-call messages (Role == "tool")
	ToolCallID string
	ToolName   string
	ToolStatus string // "running" | "done" | "failed"
	ToolInput  string
	ToolOutput string
}

// tool status markers rendered in the transcript.
const (
	toolRunning = "running"
	toolDone    = "done"
	toolFailed  = "failed"
)

// isShellTool reports whether a tool name is a shell/execute tool (gated by
// the ShowToolShell toggle). Loose substring match on the rendered title.
func isShellTool(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "bash") || strings.Contains(n, "shell") ||
		strings.Contains(n, "exec") || strings.Contains(n, "powershell")
}

// isSkillTool reports whether a tool name is a skill tool (gated by the
// ShowToolSkill toggle).
func isSkillTool(name string) bool {
	return strings.Contains(strings.ToLower(name), "skill")
}

// NewModel builds a chat model. ver is shown in the footer/sidebar; name is
// the agent name (used for ACP client identity); mode is the initial session
// mode ("auto"|"manual"|"plan"); logoColor/logoGradient drive the welcome
// logo coloring.
func NewModel(ctx context.Context, cancel context.CancelFunc, workDir, ver, mode, logoColor string, logoGradient []string) *Model {
	if mode == "" {
		mode = "manual"
	}

	// viewport (transcript scroll area)
	vp := viewport.New()
	vp.SetWidth(layout.GetViewWidth(defaultWidth))
	vp.SetHeight(layout.GetViewHeight(defaultHeight))
	vp.FillHeight = true
	vp.Style = theme.BaseStyle()

	// textarea (input)
	ta := textarea.New()
	styles := textarea.Styles{}
	styles.Focused.Base = theme.BaseStyle().Background(theme.BgSurface)
	styles.Focused.Placeholder = theme.BaseStyle().Background(theme.BgSurface).Foreground(theme.TextAsh)
	styles.Blurred.Base = theme.BaseStyle().Background(theme.BgSurface)
	styles.Blurred.Placeholder = theme.BaseStyle().Background(theme.BgSurface).Foreground(theme.TextAsh)
	// The cursor's blink-off cell renders with the CursorLine style; leave
	// its background unset and the cell carries no explicit bg of its own —
	// the renderer may then repaint it with the terminal's default
	// background (a theme-colored block flashing over the input). Pin it to
	// the surface so both blink phases stay on the card color. EndOfBuffer
	// styles the filler rows the same way: its default foreground is ANSI
	// palette 0, which the terminal theme resolves (aubergine on GNOME).
	styles.Focused.CursorLine = lipgloss.NewStyle().Background(theme.BgSurface)
	styles.Blurred.CursorLine = lipgloss.NewStyle().Background(theme.BgSurface)
	styles.Focused.EndOfBuffer = theme.BaseStyle().Background(theme.BgSurface).Foreground(theme.TextAsh)
	styles.Blurred.EndOfBuffer = theme.BaseStyle().Background(theme.BgSurface).Foreground(theme.TextAsh)
	// Re-enable the cursor blink: replacing the default Styles{} zeroes the
	// Cursor struct, which would otherwise leave the cursor static.
	// White cursor block, matching opencode: the virtual cursor renders the
	// cell under it reversed, so the color fills the block.
	styles.Cursor = textarea.CursorStyle{
		Color:      theme.TextNormal,
		Shape:      tea.CursorBlock,
		Blink:      true,
		BlinkSpeed: 530 * time.Millisecond,
	}

	ta.SetStyles(styles)
	suggestion := components.NextSuggestion()
	ta.Placeholder = suggestion + PlaceholderSuffix
	ta.Prompt = ""
	ta.SetWidth(defaultWidth)
	ta.SetHeight(layout.InputHeight)
	ta.CharLimit = 4096
	ta.ShowLineNumbers = false
	ta.Focus()

	return &Model{
		ctx:    ctx,
		cancel: cancel,

		workDir: workDir,
		version: ver,

		width:  defaultWidth,
		height: defaultHeight,

		activeSessionID: "",

		mode:      mode,
		logoColor: logoColor,
		// Default gradient: blue → purple → pink. Used when the user hasn't
		// set tui.logo_gradient or tui.colors.logo_color in settings.json.
		logoGradient: defaultLogoGradient(logoColor, logoGradient),

		focus: FocusChat,

		chatTextarea: ta,
		spinner:      components.NewLoading([]string{"|", "/", "-", "\\"}),
		loading:      false,

		statusBar:    components.NewStatusBar(),
		chatViewport: vp,

		statusText: "",

		needAutoScroll: true,

		modelId:    "",
		providerId: "",

		tips:       components.NextHelpTip(),
		suggestion: suggestion,

		// Transcript visibility defaults: thought cards are collapsed to a
		// one-line summary (Thinking... / Thought for Ns) until
		// /toggle_thinking expands them; skill/shell rows and tool details
		// are shown until toggled off (the palette icons start ●).
		visibleConfig: components.VisibleConfig{
			ShowToolSkill:  true,
			ShowToolShell:  true,
			ShowToolDetail: true,
		},

		// Permission policy is persisted; the safe default is ask.
	}
}

// defaultLogoGradient returns the gradient to use for the logo when the user
// hasn't configured one. Defaults to a white-to-gray gradient so the square
// block-typeface reads as a subtle monochrome fade on the black page.
func defaultLogoGradient(logoColor string, logoGradient []string) []string {
	if len(logoGradient) > 0 {
		return logoGradient
	}
	if strings.TrimSpace(logoColor) != "" {
		return nil // single color mode
	}
	return []string{"#ffffff", "#e9e9e9", "#d0d0d0"}
}

// SetProgram injects the tea.Program so ACP goroutines can send tea.Msg via
// Program.Send. Called by app.go after NewProgram but before Run.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// SetACPSession injects the ACP session once the in-process backend
// connection is established. Called by startACPInProcess in app.go.
func (m *Model) SetACPSession(s *openacp.Session) {
	m.acpSession = s
}

// ── ACP tea.Msg types ──

type acpReadyMsg struct {
	sessionID     string
	configOptions []openacp.SessionConfigOption
}
type agentMessageMsg struct{ text string }
type agentThoughtMsg struct{ text string }
type promptDoneMsg struct{}
type notifyClearMsg struct{}
type flushViewportMsg struct{}
type acpErrorMsg struct{ err error }
type usageUpdateMsg struct{ used, total int }
type modeUpdateMsg struct{ mode string }
type newSessionMsg struct {
	sessionID     string
	configOptions []openacp.SessionConfigOption
	mode          string
	err           error
}
type userMessageMsg struct{ text string }
type planMsg struct{ entries []openacp.PlanEntry }
type loadSessionsMsg struct {
	items []sessionItem
	err   error
}
type sessionLoadedMsg struct {
	sessionID     string
	configOptions []openacp.SessionConfigOption
	mode          string
	err           error
}
type configSetMsg struct {
	id            string
	configOptions []openacp.SessionConfigOption
	err           error
}
type configOptionsMsg struct {
	opts []openacp.SessionConfigOption
	err  error
}
type toolCallMsg struct {
	id     string
	title  string
	status string
	input  string
	output string
}
type permissionRequestMsg struct {
	req     openacp.RequestPermissionRequest
	replyCh chan openacp.RequestPermissionResponse
}

// Exported constructors for app.go to send these msgs from the ACP goroutine.
func AcpReadyMsg(sessionID string) tea.Msg { return acpReadyMsg{sessionID: sessionID} }

// AcpSessionReadyMsg carries the initial boot session plus the config
// options the server returned with it (mode/model/thought_level), so the
// input header can show them right away instead of waiting for a
// config_option_update event.
func AcpSessionReadyMsg(sessionID string, configOptions []openacp.SessionConfigOption) tea.Msg {
	return acpReadyMsg{sessionID: sessionID, configOptions: configOptions}
}

func AcpErrorMsg(err error) tea.Msg { return acpErrorMsg{err: err} }

func (m *Model) Init() tea.Cmd {
	// The textarea's virtual cursor only blinks once its blink loop is
	// kicked off; the initial message is delivered through Update, which
	// forwards it (and every subsequent cursor blink tick) to the textarea.
	// blinkTick drives the placeholder's block cursor: with an empty input
	// the textarea View is bypassed (renderInputAt hand-renders the
	// placeholder), so its cursor never shows — our own block blinks here.
	return tea.Batch(spinnerTick(), blinkTick(), func() tea.Msg { return textarea.Blink() })
}

// Update handles window resize, spinner/blink ticks, quit, text input,
// and ACP streaming events (agent messages, prompt completion, errors).
// All model state changes happen here in the bubbletea event loop — ACP
// goroutines send tea.Msg via Program.Send (thread-safe), no locks needed.
// After each update it refeeds the transcript viewport, so View stays a pure
// renderer (it never mutates state, and a render can't swallow a flush).
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.update(msg)
	m.syncViewport()
	return model, cmd
}

// update applies a single message, returning the commands it produced. It is
// the body of Update without the post-frame viewport sync.
func (m *Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.updateWindowSize(msg)

	case tea.FocusMsg:
		if m.focus == FocusChat {
			m.chatTextarea.Focus()
		}
		return m, nil

	case tea.BlurMsg:
		// Keep the textarea focused even on terminal blur events so the
		// TUI always accepts input.
		if m.focus == FocusChat {
			m.chatTextarea.Focus()
		}
		return m, nil

	case tea.PasteMsg:
		if m.focus == FocusChat {
			var cmd tea.Cmd
			m.chatTextarea, cmd = m.chatTextarea.Update(msg)
			return m, cmd
		}
		return m, nil

	case permissionRequestMsg:
		m.permissionReq = &msg.req
		m.permissionReplyCh = msg.replyCh
		m.permissionSelectedIdx = 0
		m.viewportDirty = true
		return m, nil

	case tea.KeyMsg:
		// Panel open: route keys to the panel until closed (see
		// handlePanelKey). The help panel is dismiss-only; sessions/models
		// panels only navigate and select; the command palette also filters.
		if m.panelOpen {
			if k, ok := msg.(tea.KeyPressMsg); ok {
				if cmd, handled := m.handlePanelKey(k); handled {
					return m, cmd
				}
			}
			return m, nil
		}

		// If permission dialog is open, intercept keys for selection.
		if m.permissionReq != nil {
			if k, ok := msg.(tea.KeyPressMsg); ok {
				return m, m.handlePermissionKey(k)
			}
			return m, nil
		}
		if k, ok := msg.(tea.KeyPressMsg); ok {
			switch k.String() {
			case "ctrl+c":
				// Three-level semantics: with text pending ctrl+c only
				// clears the input (never discards what was typed); with
				// an in-flight prompt it cancels; only when idle does it
				// quit — so quitting takes a deliberate second press.
				if m.chatTextarea.Value() != "" {
					m.chatTextarea.SetValue("")
					return m, nil
				}
				if m.loading {
					m.cancelPrompt()
					return m, nil
				}
				return m, tea.Quit
			case "ctrl+p":
				m.panelOpen = true
				m.panelMode = panelModeCommand
				m.panelFromSlash = false
				m.panelIdx = 0
				m.panelFilter = ""
				return m, nil
			case "esc":
				return m.escPressed()
			case "tab":
				// Empty input: tab accepts the placeholder suggestion and
				// rotates to the next one. With text present it falls
				// through to the textarea below (tab indent).
				if m.chatTextarea.Value() == "" && m.suggestion != "" {
					m.chatTextarea.SetValue(m.suggestion)
					m.chatTextarea.CursorEnd()
					m.suggestion = components.NextSuggestion()
					m.chatTextarea.Placeholder = m.suggestion + PlaceholderSuffix
					return m, nil
				}
			case "enter":
				text := m.chatTextarea.Value()
				if strings.TrimSpace(text) == "" {
					return m, nil
				}
				// While history is replaying into the transcript, hold
				// user input until the session finishes loading.
				if m.replaying {
					return m, nil
				}
				// Slash commands intercept Enter. Unknown or unimplemented
				// commands surface in the status bar instead of chatting.
				if strings.HasPrefix(strings.TrimSpace(text), "/") {
					return m.runSlashCommand(text)
				}
				m.addToHistory(text)
				// First prompt with no session yet: session creation is
				// deferred until the user's first input (no boot session is
				// created anymore), so create it now — the text is queued and
				// newSessionMsg sends it once the session exists. The message
				// is appended to the transcript immediately, matching the
				// in-flight queue path so later drains never duplicate it.
				if m.activeSessionID == "" {
					if len(m.inputQueue) >= maxInputQueue {
						m.statusText = fmt.Sprintf("Input queue full (%d pending)", maxInputQueue)
						// Keep the input text so nothing is lost.
						return m, nil
					}
					m.loading = true
					m.inChat = true
					m.updateInputWidth() // welcome box is narrower than chat; re-fit on page switch
					m.inputQueue = append(m.inputQueue, text)
					m.promptCount++
					m.messages = append(m.messages, ChatMessage{Role: "user", Content: text, TurnId: m.turnId})
					m.chatTextarea.SetValue("")
					m.viewportDirty = true
					if len(m.inputQueue) == 1 {
						// newSessionCmd sets the "Creating new session..."
						// status itself.
						return m, m.newSessionCmd()
					}
					m.statusText = fmt.Sprintf("[Queued:%d] waiting for the agent...", len(m.inputQueue))
					return m, nil
				}
				// While a prompt is in flight, queue the message instead
				// of firing a second one; the queue drains on prompt_done.
				if m.loading {
					if len(m.inputQueue) >= maxInputQueue {
						m.statusText = fmt.Sprintf("Input queue full (%d pending)", maxInputQueue)
						// Keep the input text so nothing is lost.
						return m, nil
					}
				m.inputQueue = append(m.inputQueue, text)
				m.promptCount++
				m.closeTrailingThought()
				m.messages = append(m.messages, ChatMessage{Role: "user", Content: text, TurnId: m.turnId})
				m.chatTextarea.SetValue("")
				m.viewportDirty = true
				m.statusText = fmt.Sprintf("[Queued:%d] waiting for the agent...", len(m.inputQueue))
				return m, nil
				}
				// User message → transcript, then async Prompt to ACP.
				m.promptCount++
				m.messages = append(m.messages, ChatMessage{
					Role:    "user",
					Content: text,
					TurnId:  m.turnId,
				})
				m.chatTextarea.SetValue("")
				m.viewportDirty = true
				m.loading = true
				m.inChat = true
				m.updateInputWidth() // welcome box is narrower than chat; re-fit on page switch
				m.statusText = "Running..."
				m.sendPrompt(text)
				return m, nil
			case "pgup", "pgdown":
				// Page through the transcript. PageUp freezes auto-scroll;
				// PageDown restores it once near the bottom.
				if k.String() == "pgup" {
					m.chatViewport.PageUp()
					m.needAutoScroll = false
				} else {
					m.chatViewport.PageDown()
					m.needAutoScroll = m.isNearBottom()
				}
				return m, nil
			case "home":
				if m.chatTextarea.Value() == "" {
					m.chatViewport.GotoTop()
					m.needAutoScroll = false
					return m, nil
				}
			case "end":
				if m.chatTextarea.Value() == "" {
					m.chatViewport.GotoBottom()
					m.needAutoScroll = true
					return m, nil
				}
			case "up":
				// Empty input: ↑ scrolls the transcript (3 lines).
				// Single-line input: ↑ browses older history entries.
				// Multi-line input: falls through to the textarea cursor.
				if m.chatTextarea.Value() == "" {
					m.chatViewport.ScrollUp(3)
					m.needAutoScroll = false
					return m, nil
				}
				if !strings.Contains(m.chatTextarea.Value(), "\n") {
					m.historyUp()
					return m, nil
				}
			case "down":
				if m.chatTextarea.Value() == "" {
					m.chatViewport.ScrollDown(3)
					m.needAutoScroll = m.isNearBottom()
					return m, nil
				}
				if !strings.Contains(m.chatTextarea.Value(), "\n") {
					m.historyDown()
					return m, nil
				}
			default:
				// "/" on an empty input opens the slash-command picker live,
				// matching the documented "/" trigger — and falls through so
				// the "/" itself lands in the input box: while the sheet is
				// open the typed text stays in the box (the sheet only
				// mirrors the matching commands), and Enter runs the
				// selected command and clears the box.
				if k.String() == "/" && m.chatTextarea.Value() == "" {
					m.panelOpen = true
					m.panelMode = panelModeCommand
					m.panelFromSlash = true
					m.panelIdx = 0
					m.panelFilter = ""
				}
			}
		}
		// Forward all other keys to the textarea.
		if m.focus == FocusChat {
			// Typing while browsing history exits the browse position so
			// the next ↑ starts from the freshest entries again.
			if m.historyIdx != len(m.history) {
				m.historyIdx = len(m.history)
			}
			var cmd tea.Cmd
			m.chatTextarea, cmd = m.chatTextarea.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.MouseWheelMsg:
		// Wheel over the transcript scrolls it. Up breaks auto-scroll;
		// down restores it once near the bottom. Events outside the
		// transcript area (negative coords or below the viewport) are
		// ignored so scrolling over input/status never rewinds the chat.
		if m.permissionReq != nil || msg.Y < 0 || msg.Y >= m.chatViewport.Height() {
			return m, nil
		}
		switch msg.Button {
		case tea.MouseWheelUp:
			m.chatViewport.ScrollUp(3)
			m.needAutoScroll = false
		case tea.MouseWheelDown:
			m.chatViewport.ScrollDown(3)
			m.needAutoScroll = m.isNearBottom()
		}
		return m, nil

	case tea.MouseClickMsg:
		// Click on a permission option to select it.
		if m.permissionReq != nil && msg.Button == tea.MouseLeft {
			idx := m.permissionOptionAt(msg.Y)
			if idx >= 0 {
				m.respondPermission(idx)
			}
			return m, nil
		}

	// ── ACP streaming events ──
	case acpReadyMsg:
		m.activeSessionID = msg.sessionID
		if msg.configOptions != nil {
			m.configOptions = msg.configOptions
			// Adopt the mode option so the header badge matches the server's
			// session state at boot.
			if s := sessionConfigValue(msg.configOptions, "mode"); s != "" {
				m.mode = s
			}
			return m, nil
		}
		// Lazy boot carries no options (no session yet): fetch the
		// session-less config options so the welcome header shows the
		// default model/thought-level badges.
		if len(m.configOptions) == 0 {
			return m, m.listConfigOptionsCmd()
		}
		return m, nil
	case agentMessageMsg:
		m.closeTrailingThought()
		if n := len(m.messages); n > 0 && m.messages[n-1].Role == "assistant" && m.messages[n-1].TurnId == m.turnId {
			m.messages[n-1].Content += msg.text
		} else {
			m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: msg.text, TurnId: m.turnId})
		}
		m.trimMessageStore()
		return m.markContentDirty()
	case agentThoughtMsg:
		if n := len(m.messages); n > 0 && m.messages[n-1].Role == "thought" && m.messages[n-1].TurnId == m.turnId {
			m.messages[n-1].Content += msg.text
		} else {
			start := time.Now()
			if m.replaying {
				// Replayed history has no meaningful span: leave the
				// timestamps zero so the collapsed card stays undated.
				start = time.Time{}
			}
			m.messages = append(m.messages, ChatMessage{Role: "thought", Content: msg.text, TurnId: m.turnId, ThoughtStart: start})
		}
		m.trimMessageStore()
		return m.markContentDirty()
	case notifyClearMsg:
		m.notifyMsg = ""
		return m, nil

	case flushViewportMsg:
		m.flushPending = false
		if m.renderPending {
			m.renderPending = false
			m.viewportDirty = true
		}
		return m, nil

	case promptDoneMsg:
		m.closeTrailingThought()
		m.loading = false
		m.statusText = ""
		// Drain the input queue: send the oldest waiting message next.
		if len(m.inputQueue) > 0 {
			text := m.inputQueue[0]
			m.inputQueue = m.inputQueue[1:]
			m.loading = true
			m.statusText = "Running..."
			m.sendPrompt(text)
		}
		return m, nil
	case usageUpdateMsg:
		if msg.total > 0 {
			m.contextSize = msg.total
		}
		if msg.used > 0 {
			m.usedTokens = msg.used
		}
		return m, nil
	case modeUpdateMsg:
		if msg.mode != "" {
			m.mode = msg.mode
		}
		return m, nil
	case newSessionMsg:
		fromPending := m.pendingModelsPanel
		m.pendingModelsPanel = false
		if msg.err != nil {
			m.loading = false
			m.pendingConfigSet = nil
			m.statusText = "New session failed: " + msg.err.Error()
			return m, nil
		}
		m.activeSessionID = msg.sessionID
		if msg.configOptions != nil {
			m.configOptions = msg.configOptions
		}
		if msg.mode != "" {
			m.mode = msg.mode
		}
		// A model picked from the cold-start picker: the session now exists,
		// so apply the choice via set_config_option (configSetMsg notifies).
		var modelSet tea.Cmd
		if m.pendingConfigSet != nil {
			pick := m.pendingConfigSet
			m.pendingConfigSet = nil
			modelSet = m.setConfigOptionCmd(pick.id, pick.value)
		}
		// /models requested while another trigger's session creation was in
		// flight: the fresh session carries the config options, so surface
		// the model picker. The transcript is left untouched — this is not
		// ctrl+n's reset.
		if fromPending {
			m.loading = false
			m.statusText = ""
			m.openModelsPanel()
			m.viewportDirty = true
			if len(m.inputQueue) == 0 {
				return m, modelSet
			}
		}
		// A first prompt deferred until the session existed (lazy boot
		// creation): the text was already placed in the transcript at Enter
		// time — only send it now that the session exists. The transcript
		// must NOT be reset here (it belongs to this just-created session).
		if len(m.inputQueue) > 0 {
			text := m.inputQueue[0]
			m.inputQueue = m.inputQueue[1:]
			m.loading = true
			m.statusText = "Running..."
			m.viewportDirty = true
			m.sendPrompt(text)
			return m, modelSet
		}
		// Every new-session trigger carries work — a queued first prompt, a
		// pending config pick, or a pending /models panel — and all of the
		// paths above returned. A bare creation just surfaces itself; the
		// /new reset does not go through here anymore (it never creates a
		// session, it returns to the welcome page directly).
		m.loading = false
		m.viewportDirty = true
		if modelSet != nil {
			return m, modelSet
		}
		return m, m.notify("Session created")
	case userMessageMsg:
		// Replayed user message during session/load history replay. Each
		// replayed prompt counts toward the session's turn count.
		if strings.TrimSpace(msg.text) == "" {
			return m, nil
		}
		m.promptCount++
		m.messages = append(m.messages, ChatMessage{Role: "user", Content: msg.text, TurnId: m.turnId})
		m.trimMessageStore()
		m.viewportDirty = true
		return m, nil
	case planMsg:
		m.planEntries = msg.entries
		return m, nil
	case loadSessionsMsg:
		m.sessionsLoading = false
		if msg.err != nil {
			m.statusText = "List sessions failed: " + msg.err.Error()
			return m, nil
		}
		m.sessionItems = msg.items
		m.panelIdx = 0
		return m, nil
	case sessionLoadedMsg:
		m.loading = false
		m.replaying = false
		m.pendingModelsPanel = false
		if msg.err != nil {
			m.statusText = "Load session failed: " + msg.err.Error()
			return m, nil
		}
		m.activeSessionID = msg.sessionID
		if msg.configOptions != nil {
			m.configOptions = msg.configOptions
		}
		if msg.mode != "" {
			m.mode = msg.mode
		}
		m.statusText = ""
		m.needAutoScroll = true
		m.viewportDirty = true
		return m, m.notify("Session loaded")
	case configOptionsMsg:
		// Live sessionUpdate "config_option_update" and the cold-start
		// session/list_config_options reply: keep the cached options fresh
		// so the model/thinking badges and model panel track changes.
		if msg.err != nil {
			m.pendingModelsPanel = false
			m.statusText = "List models failed: " + msg.err.Error()
			return m, nil
		}
		m.configOptions = msg.opts
		// The list a pending cold-start /models was waiting for: open the
		// picker now that the options are cached.
		if m.pendingModelsPanel {
			m.pendingModelsPanel = false
			m.loading = false
			m.statusText = ""
			m.openModelsPanel()
			m.viewportDirty = true
		}
		return m, nil
	case configSetMsg:
		if msg.err != nil {
			m.statusText = "Set config failed: " + msg.err.Error()
			return m, nil
		}
		if msg.configOptions != nil {
			m.configOptions = msg.configOptions
		}
		switch msg.id {
		case "mode":
			return m, m.notify("Mode: " + m.mode)
		case "thought_level":
			return m, m.notify("Thought level: " + m.currentThoughtLevel())
		}
		return m, m.notify("Model: " + m.currentModel())
	case toolCallMsg:
		// A tool call streams multiple updates (start → done/failed) under
		// one ToolCallID; update the existing row instead of appending.
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Role != "tool" || m.messages[i].ToolCallID != msg.id {
				continue
			}
			m.messages[i].ToolStatus = msg.status
			if msg.input != "" {
				m.messages[i].ToolInput = msg.input
			}
			if msg.output != "" {
				m.messages[i].ToolOutput = msg.output
			}
			return m.markContentDirty()
		}
		m.closeTrailingThought()
		m.messages = append(m.messages, ChatMessage{
			Role:       "tool",
			TurnId:     m.turnId,
			ToolCallID: msg.id,
			ToolName:   msg.title,
			ToolStatus: msg.status,
			ToolInput:  msg.input,
			ToolOutput: msg.output,
		})
		m.trimMessageStore()
		return m.markContentDirty()
	case acpErrorMsg:
		m.closeTrailingThought()
		m.messages = append(m.messages, ChatMessage{Role: "error", Content: msg.err.Error()})
		m.loading = false
		m.viewportDirty = true
		return m, nil

	case spinnerTickMsg:
		m.spinner = m.spinner.Tick()
		return m, spinnerTick()

	case blinkTickMsg:
		m.blink = !m.blink
		m.blinkCount++
		return m, blinkTick()
	default:
		// Forward unhandled messages to the textarea. This carries the
		// cursor's blink kickoff and its periodic blink ticks, so the
		// input cursor blinks; the textarea ignores anything unrelated.
		if m.focus == FocusChat {
			var cmd tea.Cmd
			m.chatTextarea, cmd = m.chatTextarea.Update(msg)
			return m, cmd
		}
		return m, nil
	}
	return m, nil
}

// permissionOptionAt returns the option index for a terminal Y coordinate,
// or -1 if the click is not on an option row. Uses geom.permOptionY, which
// syncViewport populates while the permission dialog is open.
func (m *Model) permissionOptionAt(y int) int {
	for i, oy := range m.geom.permOptionY {
		if oy == y {
			return i
		}
	}
	return -1
}

// respondPermission sends the user's selection back to the ACP server via
// the reply channel. idx >= 0 selects option[idx]; idx < 0 cancels.
func (m *Model) respondPermission(idx int) {
	if m.permissionReq == nil || m.permissionReplyCh == nil {
		return
	}
	var resp openacp.RequestPermissionResponse
	if idx >= 0 && idx < len(m.permissionReq.Options) {
		optID := openacp.PermissionOptionId(m.permissionReq.Options[idx].OptionID)
		resp = openacp.RequestPermissionResponse{
			Outcome: openacp.RequestPermissionOutcome{
				Outcome:  "selected",
				OptionID: &optID,
			},
		}
	} else {
		resp = openacp.RequestPermissionResponse{
			Outcome: openacp.RequestPermissionOutcome{Outcome: "cancelled"},
		}
	}
	m.permissionReplyCh <- resp
	m.permissionReq = nil
	m.permissionReplyCh = nil
}

// escPressed implements Esc outside the permission dialog: it clears the
// input, then cancels the in-flight prompt. Esc never quits — ctrl+c does.
func (m *Model) escPressed() (tea.Model, tea.Cmd) {
	if m.chatTextarea.Value() != "" {
		m.chatTextarea.SetValue("")
		return m, nil
	}
	m.cancelPrompt()
	return m, nil
}

// cancelPrompt sends session/cancel for the active session (a no-op without
// a wired ACP session). The status text reflects the outcome; loading is
// cleared when the prompt_done event arrives.
func (m *Model) cancelPrompt() {
	if m.acpSession == nil || m.activeSessionID == "" {
		return
	}
	if err := m.acpSession.Cancel(m.ctx, m.activeSessionID); err != nil {
		m.statusText = "Cancel failed"
		return
	}
	m.statusText = "Cancelling..."
}

// ── slash commands & command panel ──

// buildPanelCommands returns the filtered command list for the panel,
// honouring the current filter and per-command enabled state.
func (m *Model) buildPanelCommands() []panelCommand {
	var out []panelCommand
	for _, pc := range allPanelCommands() {
		if m.panelFilter != "" &&
			!strings.HasPrefix(pc.slash, m.panelFilter) &&
			!strings.Contains(pc.slash, m.panelFilter) {
			continue
		}
		pc.enabled = m.commandEnabled(pc)
		out = append(out, pc)
	}
	return out
}

// commandEnabled reports whether a slash command can run right now. The
// session/model commands need a live ACP backend; toggles and exit always
// work. Disabled commands stay visible in the panel, rendered dimmed.
func (m *Model) commandEnabled(pc panelCommand) bool {
	if !pc.enabled {
		return false
	}
	switch pc.action {
	case actionSessions, actionModels, actionNew:
		return m.acpSession != nil
	}
	return true
}

// runSlashCommand executes a "/"-prefixed input. The input is always
// consumed (cleared); unknown commands report in the status bar.
func (m *Model) runSlashCommand(text string) (tea.Model, tea.Cmd) {
	slash := strings.TrimSpace(text)
	m.chatTextarea.SetValue("")
	for _, pc := range allPanelCommands() {
		if pc.slash != slash {
			continue
		}
		if !pc.enabled {
			m.statusText = slash + " is disabled"
			return m, nil
		}
		return m.executeCommand(pc)
	}
	m.statusText = "Unknown command: " + slash
	return m, nil
}

// notify shows a transient success toast; it auto-clears after
// notifyDuration. Callers should return the returned cmd from Update.
func (m *Model) notify(text string) tea.Cmd {
	m.notifyMsg = text
	return func() tea.Msg {
		time.Sleep(notifyDuration)
		return notifyClearMsg{}
	}
}

// trimMessageStore keeps the in-memory transcript within maxStoredChars by
// dropping the oldest messages (17.2 lazy history: only a bounded window is
// held; search/export cover that window). The newest message always stays.
func (m *Model) trimMessageStore() {
	total := 0
	for i := range m.messages {
		total += len(m.messages[i].Content)
	}
	drop := 0
	for drop < len(m.messages)-1 && total > maxStoredChars {
		total -= len(m.messages[drop].Content)
		drop++
	}
	if drop > 0 {
		m.messages = m.messages[drop:]
	}
}

// markContentDirty schedules a viewport rebuild, throttled while the agent
// streams (9.2): chunks set renderPending and one flush fires per interval
// instead of re-rendering the whole viewport on every chunk. Discrete,
// non-streaming changes flush immediately.
func (m *Model) markContentDirty() (tea.Model, tea.Cmd) {
	if !m.loading {
		m.renderPending = false
		m.viewportDirty = true
		return m, nil
	}
	m.renderPending = true
	if m.flushPending {
		return m, nil
	}
	m.flushPending = true
	return m, m.flushViewportCmd()
}

// flushViewportCmd sleeps for the current render interval, then asks the
// event loop to flush pending transcript changes.
func (m *Model) flushViewportCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(m.renderInterval())
		return flushViewportMsg{}
	}
}

// renderInterval picks the viewport flush cadence from the transcript size:
// 200ms past 500KB, 150ms past 100KB, 100ms otherwise (9.2 动态调整).
func (m *Model) renderInterval() time.Duration {
	switch {
	case m.contentSize() > contentThrottleStep500:
		return 200 * time.Millisecond
	case m.contentSize() > contentThrottleStep100:
		return 150 * time.Millisecond
	default:
		return baseRenderInterval
	}
}

// contentSize returns the total content length of the transcript.
func (m *Model) contentSize() int {
	n := 0
	for i := range m.messages {
		n += len(m.messages[i].Content)
	}
	return n
}

// sendPrompt fires an async Prompt for a user message. Its completion (or
// error) travels back through the event loop; prompt_done also drains the
// input queue, so queued messages send in order, one at a time. The SDK
// session's internal ID is already set: sendPrompt is only ever reached
// after a session exists (boot no longer creates one — the first prompt
// creates it, and the deferred text is sent by newSessionMsg).
func (m *Model) sendPrompt(text string) {
	// Snapshot the backend handle and context: the goroutine below runs off
	// the event loop, so it must not read fields the loop can mutate. The
	// session handle is set once at connect; ctx lives for the program's life.
	sess, ctx, program := m.acpSession, m.ctx, m.program
	if sess == nil || ctx == nil {
		return
	}
	go func() {
		_, err := sess.Prompt(ctx, openacp.PromptRequest{
			Prompt: []openacp.ContentBlock{{Type: "text", Text: text}},
		})
		if err != nil {
			program.Send(acpErrorMsg{err: err})
			return
		}
		program.Send(promptDoneMsg{})
	}()
}

// panelItemCount returns the number of selectable rows for the current
// panel mode.
func (m *Model) panelItemCount() int {
	switch m.panelMode {
	case panelModeSessions:
		return len(m.sessionItems)
	case panelModeModels:
		return len(m.modelOptions())
	case panelModeConfig:
		return len(m.configPickerOptions(m.configPickerID))
	default:
		return len(m.buildPanelCommands())
	}
}

// panelExecute runs the currently selected panel entry and closes the
// panel. Pass-through text typed before opening is cleared.
func (m *Model) panelExecute() (tea.Model, tea.Cmd) {
	switch m.panelMode {
	case panelModeSessions:
		return m.execSelectedSession()
	case panelModeModels:
		return m.execSelectedModel()
	case panelModeConfig:
		return m.execSelectedConfig()
	default:
		cmds := m.buildPanelCommands()
		m.panelOpen = false
		m.panelMode = panelModeCommand
		m.panelFilter = ""
		if m.panelIdx < 0 || m.panelIdx >= len(cmds) {
			return m, nil
		}
		if m.chatTextarea.Value() != "" {
			m.chatTextarea.SetValue("")
		}
		return m.executeCommand(cmds[m.panelIdx])
	}
}

// panelToggleSelected space-toggles the selected toggle command.
func (m *Model) panelToggleSelected() (tea.Model, tea.Cmd) {
	cmds := m.buildPanelCommands()
	if m.panelIdx < 0 || m.panelIdx >= len(cmds) {
		return m, nil
	}
	pc := cmds[m.panelIdx]
	if !pc.space {
		return m, nil
	}
	return m.executeCommand(pc)
}

// executeCommand dispatches a slash command to its action.
func (m *Model) executeCommand(pc panelCommand) (tea.Model, tea.Cmd) {
	switch pc.action {
	case actionExit:
		return m, tea.Quit
	case actionNew:
		// Empty-session guard: no active session and nothing typed yet.
		if m.activeSessionID == "" && len(m.messages) == 0 {
			m.statusText = "No active session to reset"
			return m, nil
		}
		// /new returns to the welcome page — the state a fresh boot starts
		// from. No session is created eagerly: the next prompt lazily
		// creates one, and the old session stays resumable via /sessions.
		// An in-flight prompt is abandoned (cancelled) first.
		if m.loading {
			m.cancelPrompt()
		}
		m.activeSessionID = ""
		m.messages = nil
		m.inputQueue = nil
		m.pendingConfigSet = nil
		m.usedTokens, m.contextSize, m.promptCount = 0, 0, 0
		m.loading = false
		m.inChat = false
		m.updateInputWidth() // welcome box is narrower than chat; re-fit on page switch
		m.viewportDirty = true
		m.statusText = ""
		return m, nil
	case actionSessions:
		if m.acpSession == nil {
			m.statusText = "Backend not connected"
			return m, nil
		}
		return m.openSessionsPanel()
	case actionModels:
		if m.acpSession == nil {
			m.statusText = "Backend not connected"
			return m, nil
		}
		return m.openModelsPanel()
	case actionToggleThinking:
		m.visibleConfig.ExpandThinking = !m.visibleConfig.ExpandThinking
		m.viewportDirty = true
		return m, nil
	case actionToggleSkill:
		m.visibleConfig.ShowToolSkill = !m.visibleConfig.ShowToolSkill
		m.viewportDirty = true
		return m, nil
	case actionToggleShell:
		m.visibleConfig.ShowToolShell = !m.visibleConfig.ShowToolShell
		m.viewportDirty = true
		return m, nil
	case actionToggleToolDetail:
		m.visibleConfig.ShowToolDetail = !m.visibleConfig.ShowToolDetail
		m.viewportDirty = true
		return m, nil
	case actionToggleMode:
		return m.openConfigPanel("mode")
	case actionThoughtLevel:
		return m.openConfigPanel("thought_level")
	case actionToggleLineNumbers:
		m.chatTextarea.ShowLineNumbers = !m.chatTextarea.ShowLineNumbers
		if m.chatTextarea.ShowLineNumbers {
			m.statusText = "Input line numbers: on"
		} else {
			m.statusText = "Input line numbers: off"
		}
		return m, nil
	case actionHelp:
		m.panelOpen = true
		m.panelMode = panelModeHelp
		m.panelFromSlash = false // centered
		m.panelFilter = ""
		m.panelIdx = 0
		return m, nil
	case actionSearch:
		return m.openSearchPanel()
	case actionExport:
		return m.exportTranscript()
	case actionEdit:
		return m.openEditPanel()
	case actionTheme:
		return m.cycleTheme()
	case actionPlugins:
		return m.openPluginsPanel()
	case actionSplit:
		m.splitView = !m.splitView
		m.viewportDirty = true
		if m.splitView {
			m.statusText = "Split view: on"
		} else {
			m.statusText = "Split view: off"
		}
		return m, nil
	default:
		// actionSessions/actionModels/actionUpdateSkills land here until the
		// panels land in later phases.
		m.statusText = pc.slash + ": not implemented yet"
		return m, nil
	}
}

// newSessionCmd closes the current session (if any) and opens a fresh one
// with the same workdir, resetting the transcript on completion.
func (m *Model) newSessionCmd() tea.Cmd {
	m.loading = true
	m.statusText = "Creating new session..."
	// Snapshot backend state at command-creation time; the closure runs on a
	// bubbletea command goroutine, so it must not read fields the event loop
	// can mutate (activeSessionID changes as sessions swap).
	sess, ctx, workDir, activeID := m.acpSession, m.ctx, m.workDir, m.activeSessionID
	return func() tea.Msg {
		if sess == nil {
			return newSessionMsg{err: fmt.Errorf("backend not connected")}
		}
		if activeID != "" {
			_ = sess.CloseSession(ctx)
		}
		resp, err := sess.NewSession(ctx, openacp.NewSessionRequest{Cwd: workDir})
		if err != nil {
			return newSessionMsg{err: err}
		}
		msg := newSessionMsg{sessionID: resp.SessionID, configOptions: resp.ConfigOptions}
		if resp.Modes != nil {
			msg.mode = string(resp.Modes.CurrentModeID)
		}
		return msg
	}
}

// ── sessions & models panels ──

// openSessionsPanel opens the session list and kicks off ListSessions.
func (m *Model) openSessionsPanel() (tea.Model, tea.Cmd) {
	m.panelOpen = true
	m.panelMode = panelModeSessions
	m.panelFromSlash = false // centered, not docked to the slash sheet
	m.panelIdx = 0
	m.panelFilter = ""
	m.sessionsLoading = true
	m.sessionItems = nil
	if m.acpSession == nil {
		m.sessionsLoading = false
		return m, nil
	}
	return m, m.listSessionsCmd()
}

// openSearchPanel opens the transcript search overlay.
func (m *Model) openSearchPanel() (tea.Model, tea.Cmd) {
	m.panelOpen = true
	m.panelMode = panelModeSearch
	m.panelFromSlash = false // centered
	m.panelFilter = ""
	m.panelIdx = 0
	m.searchResults = m.searchResults[:0]
	return m, nil
}

// editableMessages returns the transcript indices of user messages,
// newest first (a past user message can be edited and resent).
func (m *Model) editableMessages() []int {
	var idxs []int
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == "user" {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

// openEditPanel opens the message-edit picker.
func (m *Model) openEditPanel() (tea.Model, tea.Cmd) {
	m.panelOpen = true
	m.panelMode = panelModeEdit
	m.panelFromSlash = false // centered
	m.panelIdx = 0
	return m, nil
}

// execEditSelection copies the selected user message back into the input
// box for editing and closes the picker.
func (m *Model) execEditSelection() (tea.Model, tea.Cmd) {
	m.panelOpen = false
	m.panelMode = panelModeCommand
	idxs := m.editableMessages()
	if m.panelIdx < 0 || m.panelIdx >= len(idxs) {
		return m, nil
	}
	m.chatTextarea.SetValue(m.messages[idxs[m.panelIdx]].Content)
	m.chatTextarea.CursorEnd()
	m.viewportDirty = true
	return m, nil
}

// exportTranscript writes the transcript to a timestamped Markdown file
// next to the workdir and raises a toast with the path.
func (m *Model) exportTranscript() (tea.Model, tea.Cmd) {
	if len(m.messages) == 0 {
		return m.openExportPanel("Nothing to export yet — send a message first.")
	}
	dir := m.workDir
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return m.openExportPanel("Export failed: " + err.Error())
	}
	name := filepath.Join(dir, fmt.Sprintf("chat-%s.md", time.Now().Format("20060102-150405")))
	if err := os.WriteFile(name, []byte(m.exportMarkdown()), 0o600); err != nil {
		return m.openExportPanel("Export failed: " + err.Error())
	}
	return m.openExportPanel("Transcript exported to:\n" + name)
}

// openExportPanel surfaces the /export outcome in a dismiss-only dialog: a
// transient status-bar toast was too easy to miss.
func (m *Model) openExportPanel(msg string) (tea.Model, tea.Cmd) {
	m.exportNotice = msg
	m.panelOpen = true
	m.panelMode = panelModeExport
	m.panelFromSlash = false // centered
	m.panelIdx = 0
	m.panelFilter = ""
	return m, nil
}

// exportMarkdown renders the transcript as a raw Markdown document (no
// ANSI): roles become headings, tool payloads become code blocks.
func (m *Model) exportMarkdown() string {
	var b strings.Builder
	b.WriteString("# Chat transcript\n\n")
	if m.activeSessionID != "" {
		b.WriteString("Session: " + m.activeSessionID + "\n\n")
	}
	for i, msg := range m.messages {
		switch msg.Role {
		case "user":
			b.WriteString(fmt.Sprintf("## %d. User\n\n%s\n\n", i+1, msg.Content))
		case "assistant":
			b.WriteString(fmt.Sprintf("## %d. Assistant\n\n%s\n\n", i+1, msg.Content))
		case "thought":
			b.WriteString(fmt.Sprintf("### %d. Thought\n\n> %s\n\n", i+1, msg.Content))
		case "tool":
			b.WriteString(fmt.Sprintf("### %d. Tool: %s (%s)\n\n", i+1, msg.ToolName, msg.ToolStatus))
			if msg.ToolInput != "" {
				b.WriteString("```json\n" + msg.ToolInput + "\n```\n\n")
			}
			if msg.ToolOutput != "" {
				b.WriteString("```\n" + msg.ToolOutput + "\n```\n\n")
			}
		case "error":
			b.WriteString(fmt.Sprintf("### %d. Error\n\n**%s**\n\n", i+1, msg.Content))
		default:
			b.WriteString(fmt.Sprintf("### %d.\n\n%s\n\n", i+1, msg.Content))
		}
	}
	return b.String()
}

// refreshSearch recomputes the matched message indices for the current
// query (case-insensitive substring over message content).
func (m *Model) refreshSearch() {
	q := strings.ToLower(m.panelFilter)
	m.searchResults = m.searchResults[:0]
	if q == "" {
		return
	}
	for i := range m.messages {
		if strings.Contains(strings.ToLower(m.messages[i].Content), q) {
			m.searchResults = append(m.searchResults, i)
		}
	}
	if m.panelIdx >= len(m.searchResults) {
		m.panelIdx = 0
	}
}

// execSearchSelection scrolls the viewport to the selected match and
// closes the search panel.
func (m *Model) execSearchSelection() (tea.Model, tea.Cmd) {
	m.panelOpen = false
	m.panelFilter = ""
	if m.panelIdx < 0 || m.panelIdx >= len(m.searchResults) {
		return m, nil
	}
	idx := m.searchResults[m.panelIdx]
	// Jump via the virtual line mapping: feed a document whose styled
	// window is the destination (SetYOffset clamps against it), then place
	// the match's first estimated row at the top of the window.
	jump := m.virtualPrefixLines(idx)
	m.chatViewport.SetContent(m.renderVirtualDocAt(m.chatViewport.Height(), jump))
	m.chatViewport.SetYOffset(jump)
	m.fedOffset = jump
	m.fedHeight = m.chatViewport.Height()
	m.needAutoScroll = false
	m.viewportDirty = false
	return m, nil
}

// listSessionsCmd fetches the sessions available on the agent. Only
// ACP-channel sessions are shown — IM-channel rows (feishu/wechat/…)
// have no replayable chat history in the TUI.
func (m *Model) listSessionsCmd() tea.Cmd {
	// Snapshot backend state: the closure runs on a bubbletea command
	// goroutine, off the event loop.
	sess, ctx := m.acpSession, m.ctx
	return func() tea.Msg {
		if sess == nil {
			return loadSessionsMsg{err: fmt.Errorf("backend not connected")}
		}
		resp, err := sess.ListSessions(ctx, openacp.ListSessionsRequest{})
		if err != nil {
			return loadSessionsMsg{err: err}
		}
		var items []sessionItem
		for _, si := range resp.Sessions {
			if ch, ok := si.Meta["channel"].(string); ok && ch != "acp" {
				continue
			}
			title := si.Title
			if title == "" {
				title = string(si.SessionID)
			}
			items = append(items, sessionItem{id: string(si.SessionID), title: title, updated: si.UpdatedAt})
		}
		return loadSessionsMsg{items: items}
	}
}

// execSelectedSession loads the chosen session: the transcript is cleared,
// history replays in, and on completion the session state is refreshed.
func (m *Model) execSelectedSession() (tea.Model, tea.Cmd) {
	m.panelOpen = false
	m.panelMode = panelModeCommand
	if m.panelIdx < 0 || m.panelIdx >= len(m.sessionItems) {
		return m, nil
	}
	item := m.sessionItems[m.panelIdx]
	if item.id == m.activeSessionID {
		m.statusText = "Already on this session"
		return m, nil
	}
	// Enter the chat view: a session picked straight from the welcome page
	// must leave the welcome screen, or the replayed transcript would stay
	// invisible (View renders welcome until inChat is set).
	m.inChat = true
	m.messages = nil
	// The target session's own replay re-counts turns; usage waits for its
	// first usage_update.
	m.usedTokens, m.contextSize, m.promptCount = 0, 0, 0
	m.loading = true
	m.replaying = true
	m.statusText = "Loading session..."
	return m, m.loadSessionCmd(item.id)
}

// loadSessionCmd closes the current session (if different) and loads the
// target, replaying its history into the event stream.
func (m *Model) loadSessionCmd(id string) tea.Cmd {
	// Snapshot backend state: the closure runs on a bubbletea command
	// goroutine, off the event loop (activeSessionID can change mid-flight).
	sess, ctx, workDir, activeID := m.acpSession, m.ctx, m.workDir, m.activeSessionID
	return func() tea.Msg {
		if sess == nil {
			return sessionLoadedMsg{err: fmt.Errorf("backend not connected")}
		}
		if activeID != "" && activeID != id {
			_ = sess.CloseSession(ctx)
		}
		resp, err := sess.LoadSession(ctx, openacp.LoadSessionRequest{
			SessionID: id,
			Cwd:       workDir,
		})
		if err != nil {
			return sessionLoadedMsg{err: err}
		}
		msg := sessionLoadedMsg{sessionID: id, configOptions: resp.ConfigOptions}
		if resp.Modes != nil {
			msg.mode = string(resp.Modes.CurrentModeID)
		}
		return msg
	}
}

// openModelsPanel shows the model picker from the cached config options.
// Under lazy boot the options only arrive over the wire, so with no session
// yet the panel fetches them via session/list_config_options
// (pendingModelsPanel) instead of bouncing the user or creating a session
// just to browse.
func (m *Model) openModelsPanel() (tea.Model, tea.Cmd) {
	if m.activeSessionID == "" && len(m.modelOptions()) == 0 {
		if m.pendingModelsPanel {
			return m, nil // list already in flight
		}
		m.pendingModelsPanel = true
		m.panelOpen = false
		m.panelMode = panelModeCommand
		m.panelFromSlash = false
		m.statusText = "Fetching model list..."
		return m, m.listConfigOptionsCmd()
	}
	m.panelOpen = true
	m.panelMode = panelModeModels
	m.panelFromSlash = false // centered
	m.panelIdx = 0
	m.panelFilter = ""
	if len(m.modelOptions()) == 0 {
		m.statusText = "No model options available"
		m.panelOpen = false
		m.panelMode = panelModeCommand
	}
	return m, nil
}

// listConfigOptionsCmd fetches the session-less config options (mode,
// thought level, model list) via session/list_config_options — the
// cold-start model picker's data source.
func (m *Model) listConfigOptionsCmd() tea.Cmd {
	// Snapshot backend state: the closure runs on a bubbletea command
	// goroutine, off the event loop.
	sess, ctx := m.acpSession, m.ctx
	return func() tea.Msg {
		if sess == nil {
			return configOptionsMsg{err: fmt.Errorf("backend not connected")}
		}
		resp, err := sess.ListConfigOptions(ctx, openacp.ListConfigOptionsRequest{})
		if err != nil {
			return configOptionsMsg{err: err}
		}
		return configOptionsMsg{opts: resp.ConfigOptions}
	}
}

// configPickerRow is one entry of a config-option picker, mirrored from the
// agent's select option — the TUI adapts to whatever options the agent
// defines instead of hardcoding a list.
type configPickerRow struct {
	value string
	name  string
	desc  string
}

// configPickerOptions returns the picker rows for a config option id ("mode",
// "thought_level", ...) from the cached config options.
func (m *Model) configPickerOptions(id string) []configPickerRow {
	o := sessionConfigOption(m.configOptions, id)
	if o == nil {
		return nil
	}
	rows := make([]configPickerRow, 0, len(o.Options))
	for _, v := range o.Options {
		name := v.Name
		if name == "" {
			name = v.Value
		}
		rows = append(rows, configPickerRow{value: v.Value, name: name, desc: v.Description})
	}
	return rows
}

// configPickerIndex returns the picker row matching the option's current
// value (0 when it is unknown to the picker).
func (m *Model) configPickerIndex(id string) int {
	current := sessionConfigValue(m.configOptions, id)
	for i, r := range m.configPickerOptions(id) {
		if r.value == current {
			return i
		}
	}
	return 0
}

// openConfigPanel shows a select config option as a picker popup (the
// session mode, the thought level, ...) preselected on the current value.
func (m *Model) openConfigPanel(id string) (tea.Model, tea.Cmd) {
	if len(m.configPickerOptions(id)) == 0 {
		m.statusText = "No options available"
		return m, nil
	}
	m.configPickerID = id
	m.panelOpen = true
	m.panelMode = panelModeConfig
	m.panelFromSlash = false // centered
	m.panelFilter = ""
	m.panelIdx = m.configPickerIndex(id)
	return m, nil
}

// execSelectedConfig applies the picked value via SetSessionConfigOption.
// Picking from the cold-start picker (no session yet) creates one lazily:
// the choice rides in pendingConfigSet until newSessionMsg lands.
func (m *Model) execSelectedConfig() (tea.Model, tea.Cmd) {
	rows := m.configPickerOptions(m.configPickerID)
	m.panelOpen = false
	m.panelMode = panelModeCommand
	if m.panelIdx < 0 || m.panelIdx >= len(rows) {
		return m, nil
	}
	value := rows[m.panelIdx].value
	if value == sessionConfigValue(m.configOptions, m.configPickerID) {
		return m, nil
	}
	if m.configPickerID == "mode" {
		m.mode = value // badge feedback; the server state lands via configSetMsg
	}
	if m.activeSessionID == "" {
		if m.acpSession == nil {
			m.statusText = "Backend not connected"
			return m, nil
		}
		m.pendingConfigSet = &configPick{id: m.configPickerID, value: value}
		return m, m.newSessionCmd()
	}
	return m, m.setConfigOptionCmd(m.configPickerID, value)
}

// execSelectedModel switches the session model via SetSessionConfigOption.
// Picking from the cold-start picker (no session yet) creates one lazily:
// the choice rides in pendingModelSet until newSessionMsg lands.
func (m *Model) execSelectedModel() (tea.Model, tea.Cmd) {
	opts := m.modelOptions()
	m.panelOpen = false
	m.panelMode = panelModeCommand
	if m.panelIdx < 0 || m.panelIdx >= len(opts) {
		return m, nil
	}
	modelID := opts[m.panelIdx]
	if modelID == m.currentModel() {
		m.statusText = "Already on " + modelID
		return m, nil
	}
	if m.activeSessionID == "" {
		if m.acpSession == nil {
			m.statusText = "Backend not connected"
			return m, nil
		}
		m.pendingConfigSet = &configPick{id: "model", value: modelID}
		return m, m.newSessionCmd()
	}
	if m.acpSession == nil {
		m.statusText = "Backend not connected"
		return m, nil
	}
	return m, m.setConfigOptionCmd("model", modelID)
}

// modelOptions returns the selectable model ids from the cached config.
func (m *Model) modelOptions() []string {
	return sessionConfigValues(m.configOptions, "model")
}

// currentModel returns the active model id, or "" when unknown.
func (m *Model) currentModel() string {
	return sessionConfigValue(m.configOptions, "model")
}

// currentThoughtLevel returns the active thought-strength setting from the
// session config (category/option "thought_level"), or "" when the agent
// offers no such option.
func (m *Model) currentThoughtLevel() string {
	return sessionConfigValue(m.configOptions, "thought_level")
}

// setConfigOptionCmd changes a session config option on the server.
func (m *Model) setConfigOptionCmd(id, value string) tea.Cmd {
	// Snapshot backend state: the closure runs on a bubbletea command
	// goroutine, off the event loop (activeSessionID can change mid-flight).
	sess, ctx, activeID := m.acpSession, m.ctx, m.activeSessionID
	return func() tea.Msg {
		if sess == nil {
			return configSetMsg{err: fmt.Errorf("backend not connected")}
		}
		resp, err := sess.SetSessionConfigOption(ctx, openacp.SetSessionConfigOptionRequest{
			SessionID: activeID,
			ConfigID:  openacp.SessionConfigId(id),
			Type:      "select",
			Value:     value,
		})
		if err != nil {
			return configSetMsg{id: id, err: err}
		}
		return configSetMsg{id: id, configOptions: resp.ConfigOptions}
	}
}

// isNearBottom reports whether the transcript is scrolled within 2 lines of
// the bottom; used to decide whether auto-scroll resumes on wheel-down.
func (m *Model) isNearBottom() bool {
	vpH := m.chatViewport.Height()
	total := m.chatViewport.TotalLineCount()
	if total <= vpH {
		return true
	}
	return m.chatViewport.YOffset() >= total-vpH-2
}

// ── input history ──

// addToHistory appends a sent prompt to the history ring (capped at
// maxHistory) and resets the browse position to the freshest entry.
func (m *Model) addToHistory(text string) {
	if text == "" {
		return
	}
	m.history = append(m.history, text)
	if len(m.history) > maxHistory {
		m.history = m.history[1:]
	}
	m.historyIdx = len(m.history)
}

// historyUp moves the input to the previous history entry. The first ↑
// from fresh input lands on the newest entry (shell convention).
func (m *Model) historyUp() {
	if len(m.history) == 0 {
		return
	}
	switch {
	case m.historyIdx == len(m.history):
		m.historyIdx = len(m.history) - 1
	case m.historyIdx > 0:
		m.historyIdx--
	default:
		return
	}
	m.chatTextarea.SetValue(m.history[m.historyIdx])
	m.chatTextarea.CursorEnd()
}

// historyDown moves the input to the next history entry (or clears it at
// the freshest position).
func (m *Model) historyDown() {
	if m.historyIdx >= len(m.history) {
		return
	}
	m.historyIdx++
	if m.historyIdx == len(m.history) {
		m.chatTextarea.SetValue("")
	} else {
		m.chatTextarea.SetValue(m.history[m.historyIdx])
	}
	m.chatTextarea.CursorEnd()
}

func (m *Model) updateWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.chatViewport.SetWidth(layout.GetViewWidth(m.width))
	m.chatViewport.SetHeight(layout.GetViewHeight(m.height))
	m.updateInputWidth()
	m.viewportDirty = true
	m.textareaDirty = true
	return m, nil
}

// ── tick commands ──

type spinnerTickMsg struct {
	time time.Time
}

func spinnerTick() tea.Cmd {
	return tea.Tick(time.Millisecond*(1000/4), func(t time.Time) tea.Msg {
		return spinnerTickMsg{time: t}
	})
}

type blinkTickMsg struct {
	time time.Time
}

func blinkTick() tea.Cmd {
	return tea.Tick(time.Millisecond*530, func(t time.Time) tea.Msg {
		return blinkTickMsg{time: t}
	})
}

// renderMessages renders the message list as a styled transcript. Each role
// gets distinct visual treatment (background, border, color) so user input,
// agent thoughts, and agent replies are clearly separated.
// renderMessages renders the message list as a styled transcript. Each role
// gets distinct visual treatment (background, border, color) so user input,
// agent thoughts, and agent replies are clearly separated.
func (m *Model) renderMessages() string {
	return m.renderMessagesRange(0, len(m.messages))
}

// renderCacheEntry remembers one message's styled block together with the
// fingerprint it was styled under, so unchanged messages skip re-styling.
type renderCacheEntry struct {
	vpW                                                  int
	loading                                              bool
	expandThink, showSkill, showShell, showDetail        bool
	thoughtStart, thoughtEnd                             time.Time
	role, content, toolName, toolStatus, toolIn, toolOut string
	block                                                string
	skip                                                 bool
}

// renderCacheHits reports whether a cached entry was styled under exactly
// the current message and viewport settings. The thought timestamps are part
// of the fingerprint: closing a trailing thought (end stamp) must restyle
// its collapsed summary even though the content is unchanged.
func renderCacheHits(e renderCacheEntry, msg ChatMessage, vpW int, loading bool, vc components.VisibleConfig) bool {
	return e.vpW == vpW &&
		e.loading == loading &&
		e.expandThink == vc.ExpandThinking &&
		e.showSkill == vc.ShowToolSkill &&
		e.showShell == vc.ShowToolShell &&
		e.showDetail == vc.ShowToolDetail &&
		e.thoughtStart.Equal(msg.ThoughtStart) &&
		e.thoughtEnd.Equal(msg.ThoughtEnd) &&
		e.role == msg.Role &&
		e.content == msg.Content &&
		e.toolName == msg.ToolName &&
		e.toolStatus == msg.ToolStatus &&
		e.toolIn == msg.ToolInput &&
		e.toolOut == msg.ToolOutput
}

// renderMessageBlock returns the styled block for message i (skip=true when
// the message is gated out by visibility toggles). Untouched messages reuse
// their cached block instead of re-running glamour/lipgloss styling.
func (m *Model) renderMessageBlock(i int, msg ChatMessage, vpW int) (block string, skip bool) {
	if m.renderCache == nil {
		m.renderCache = make(map[int]renderCacheEntry)
	}
	if e, ok := m.renderCache[i]; ok &&
		renderCacheHits(e, msg, vpW, m.loading, m.visibleConfig) {
		return e.block, e.skip
	}
	e := renderCacheEntry{
		vpW: vpW, loading: m.loading,
		expandThink: m.visibleConfig.ExpandThinking,
		showSkill:   m.visibleConfig.ShowToolSkill,
		showShell:   m.visibleConfig.ShowToolShell,
		showDetail:  m.visibleConfig.ShowToolDetail,
		thoughtStart: msg.ThoughtStart,
		thoughtEnd:   msg.ThoughtEnd,
		role:         msg.Role, content: msg.Content,
		toolName: msg.ToolName, toolStatus: msg.ToolStatus,
		toolIn: msg.ToolInput, toolOut: msg.ToolOutput,
	}
	e.block, e.skip = m.styleMessageBlock(msg, vpW)
	m.renderCache[i] = e
	return e.block, e.skip
}

// messageRoleBorder returns the transcript rail color for a role. The rail
// card is the user's prompt alone (the opencode look); the color keeps the
// user's blue identity.
func messageRoleBorder(role string) color.Color {
	switch role {
	case "user":
		return theme.Primary
	default:
		return theme.BorderGray
	}
}

// messageCard wraps body in the transcript's only card: the user's prompt
// rail — a surface block with a role-colored left border. The border column
// shares the card background so the rail reads as part of the block instead
// of a black gutter. Inner padding is symmetric (top and bottom 1) so the
// text floats evenly inside the card; vertical rhythm comes from a single
// 1-row margin below each card (no top margin), so adjacent cards sit
// exactly one blank row apart. vpW is the total card width; a Width(vpW)
// style renders to the viewport width including border and padding.
func messageCard(vpW int, border color.Color, body string) string {
	return theme.BaseStyle().
		MarginBottom(1).MarginBackground(theme.BgNormal).
		Padding(1, 1, 1, 1).
		Width(vpW).
		Background(theme.BgSurface).
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderBackground(theme.BgSurface).
		BorderForeground(border).
		Render(body)
}

// transcriptIndent is the left indent of the cardless agent-side blocks —
// the opencode look: only the user's prompt is a card; assistant prose,
// thoughts and tool rows sit straight on the page, indented 3.
const transcriptIndent = 3

// indentedBlock renders a cardless transcript block: the body's lines are
// indented by transcriptIndent page-background columns, with one blank
// margin row at the bottom (the inter-message gap). Lines arrive styled by
// the caller; the pad cells carry the page background so no cell falls to
// an unpainted default.
func indentedBlock(body string) string {
	pad := theme.BaseStyle().Render(strings.Repeat(" ", transcriptIndent))
	lines := strings.Split(body, "\n")
	for i, ln := range lines {
		lines[i] = pad + ln
	}
	return strings.Join(append(lines, ""), "\n")
}

// formatThoughtDuration renders a thinking span as a compact duration:
// "3.2s" below ten seconds, whole seconds below a minute, then "1m05s".
func formatThoughtDuration(d time.Duration) string {
	s := d.Seconds()
	switch {
	case s < 10:
		return fmt.Sprintf("%.1fs", s)
	case s < 60:
		return fmt.Sprintf("%.0fs", s)
	default:
		return fmt.Sprintf("%dm%02ds", int(s)/60, int(s)%60)
	}
}

// closeTrailingThought stamps the end time on a still-open trailing thought
// message: thinking is over once any other message follows or the turn
// finishes. Replayed thoughts (zero start) stay untouched — their duration
// is unknowable.
func (m *Model) closeTrailingThought() {
	n := len(m.messages)
	if n == 0 || m.messages[n-1].Role != "thought" {
		return
	}
	if m.messages[n-1].ThoughtStart.IsZero() || !m.messages[n-1].ThoughtEnd.IsZero() {
		return
	}
	m.messages[n-1].ThoughtEnd = time.Now()
}

// styleMessageBlock renders one message as its transcript block. skip means
// the message is hidden by the current visibility toggles.
func (m *Model) styleMessageBlock(msg ChatMessage, vpW int) (string, bool) {
	content := strings.TrimSpace(msg.Content)
	switch msg.Role {
	case "user":
		return messageCard(vpW, messageRoleBorder("user"), content), false
	case "assistant":
		// Cardless (opencode style): markdown straight on the page,
		// wrapped to the indent-adjusted width so no line soft-wraps.
		body := renderMarkdownText(content, vpW-transcriptIndent)
		return indentedBlock(body), false
	case "thought":
		return m.thoughtBlock(msg, content)
	case "tool":
		// Skill/shell rows are gated by their toggles; the detail toggle
		// hides input/output (the status line stays).
		if isSkillTool(msg.ToolName) && !m.visibleConfig.ShowToolSkill {
			return "", true
		}
		if isShellTool(msg.ToolName) && !m.visibleConfig.ShowToolShell {
			return "", true
		}
		return indentedBlock(m.toolBody(msg)), false
	case "error":
		return indentedBlock(theme.BaseStyle().Foreground(theme.Danger).Render(content)), false
	default:
		return indentedBlock(theme.BaseStyle().Render(content)), false
	}
}

// thoughtBlock renders the thought as opencode does — a warning-colored
// one-liner, cardless: "Thinking..." while the round-trip streams,
// "Thought: <duration>" once done, a bare "Thought" when the span is
// unknown (replayed history). /toggle_thinking expands it to the full
// content in muted text under the same header.
func (m *Model) thoughtBlock(msg ChatMessage, content string) (string, bool) {
	var header string
	switch {
	case msg.ThoughtEnd.IsZero() && m.loading:
		header = theme.BaseStyle().Foreground(theme.Warning).Render("Thinking...")
	case !msg.ThoughtStart.IsZero() && !msg.ThoughtEnd.IsZero():
		header = theme.BaseStyle().Foreground(theme.Warning).
			Render("Thought: " + formatThoughtDuration(msg.ThoughtEnd.Sub(msg.ThoughtStart)))
	case content != "":
		header = theme.BaseStyle().Foreground(theme.Warning).Render("Thought")
	default:
		return "", true
	}
	if m.visibleConfig.ExpandThinking && content != "" {
		body := theme.BaseStyle().Foreground(theme.TextMute).Render(content)
		return indentedBlock(header + "\n" + body), false
	}
	return indentedBlock(header), false
}

// toolBody renders the tool row(s): a status icon + tool name (with the
// folded input when details are on). Colors track status like opencode —
// running bright, completed muted, failed red — and the output renders
// muted below the status line.
func (m *Model) toolBody(msg ChatMessage) string {
	icon, fg := "⏳", theme.TextNormal
	switch msg.ToolStatus {
	case toolDone:
		icon, fg = "✓", theme.TextMute
	case toolFailed:
		icon, fg = "✗", theme.Danger
	}
	line := theme.BaseStyle().Foreground(fg).Render(icon + " " + msg.ToolName)
	if m.visibleConfig.ShowToolDetail && msg.ToolInput != "" {
		line += theme.BaseStyle().Foreground(fg).Render(" (" + foldOutput(msg.ToolInput, 1) + ")")
	}
	if m.visibleConfig.ShowToolDetail && msg.ToolOutput != "" {
		line += "\n" + theme.BaseStyle().Foreground(theme.TextMute).Render(foldOutput(msg.ToolOutput, defaultToolOutputLines))
	}
	return line
}

// renderMessagesRange renders messages[start:end) with the standard block
// styles. The start is clamped to the maxRenderChars budget start, so a
// match inside the rendered window can be measured and scrolled to; the
// whole document rendered here is what the viewport displays.
func (m *Model) renderMessagesRange(start, end int) string {
	if len(m.messages) == 0 {
		return ""
	}
	// Render budget (9.3): keep as many of the NEWEST messages as fit
	// within maxRenderChars, dropping the oldest that overflow so a long
	// session never rebuilds a huge document. The newest message always
	// renders, even when it alone exceeds the budget.
	sizes := make([]int, len(m.messages))
	for i := range m.messages {
		sizes[i] = len(m.messages[i].Content)
	}
	if keepStart := renderKeepStart(sizes, maxRenderChars); start < keepStart {
		start = keepStart
	}
	if end < start {
		end = start
	}
	vpW := layout.GetViewWidth(m.width)
	var doc strings.Builder
	// The style cache can never legitimately hold more entries than there
	// are messages (indices), so capping it against the message count bounds
	// memory after the render window advances past old messages.
	if len(m.renderCache) > len(m.messages) {
		m.renderCache = nil
	}
	wrote := false
	for i := start; i < end; i++ {
		block, skip := m.renderMessageBlock(i, m.messages[i], vpW)
		if !skip {
			// messageCard's block does not end with a newline, so joining
			// blocks back-to-back would collapse the boundary line: the
			// previous card's bottom margin row merges with the next card's
			// top pad row on one wrapped line (background leaks). Keep
			// exactly one newline between blocks.
			if wrote {
				doc.WriteByte('\n')
			}
			doc.WriteString(block)
			wrote = true
		}
	}
	return doc.String()
}

// renderKeepStart returns the index of the first message to render so the
// newest sizes fit within budget (at least one message always survives).
func renderKeepStart(sizes []int, budget int) int {
	total, kept := 0, 0
	for i := len(sizes) - 1; i >= 0; i-- {
		if total+sizes[i] > budget && kept > 0 {
			break
		}
		total += sizes[i]
		kept++
	}
	return len(sizes) - kept
}

// foldOutput keeps the first maxLines lines of a multi-line string (tool
// output, long parameters) and appends a "… (N more lines)" hint.
func foldOutput(s string, maxLines int) string {
	lines := strings.Split(utils.UnifiedEndOfLine(s), "\n")
	if len(lines) <= maxLines {
		return s
	}
	head := strings.Join(lines[:maxLines], "\n")
	return head + fmt.Sprintf("\n… (%d more lines)", len(lines)-maxLines)
}

// ── markdown rendering (gruff) ──
//
// Assistant replies are rendered with gruff, a lightweight ANSI markdown
// engine (goldmark AST + custom renderer with its own ANSI-aware word wrap)
// — an order of magnitude faster and leaner than glamour on large messages
// and free of glamour's global style registry. gruff's dark theme paints
// backgrounds as empty, so the surface color still has to be injected: every
// span that sets no background gets the card surface appended to its SGR and
// each line starts on the surface, so no markdown cell ever falls through to
// the page black.

// mdSurfaceSeq is the SGR background parameter sequence the markdown sits
// on, e.g. "48;2;28;28;28". Since the transcript went cardless (opencode
// style), that surface is the page background itself.
func mdSurfaceSeq() string {
	rgb := color.RGBAModel.Convert(theme.BgNormal).(color.RGBA)
	return fmt.Sprintf("48;2;%d;%d;%d", rgb.R, rgb.G, rgb.B)
}

// mdSgrRe matches an SGR sequence like "\x1b[38;2;255;255;255m".
var mdSgrRe = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// markdownSurfaceBG makes every markdown cell sit on the card surface.
// gruff's spans set only the foreground (its dark theme has no backgrounds);
// appending the surface background to every span that sets none keeps
// resets from dropping cells to the terminal default (which the page
// PaintBackground pass would turn black). Lines also start on the surface so
// any bare cells (indentation, glue spaces) never fall through either. A
// bare reset is rewritten to "reset + surface bg" so the very next cell —
// e.g. table alignment spaces emitted after a styled run — keeps the
// surface too.
func markdownSurfaceBG(out string) string {
	bgSeq := mdSurfaceSeq()
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if line != "" {
			line = "\x1b[" + bgSeq + "m" + line
		}
		line = mdSgrRe.ReplaceAllStringFunc(line, func(m string) string {
			params := m[2 : len(m)-1]
			if strings.Contains(params, "48;") {
				return m
			}
			if params == "" || params == "0" {
				return "\x1b[0;" + bgSeq + "m"
			}
			return "\x1b[" + params + ";" + bgSeq + "m"
		})
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

// renderMarkdownText renders GFM markdown to a styled ANSI string wrapped
// to width columns, sitting on the card surface. Falls back to the raw text
// on any render error so the chat never blanks out.
func renderMarkdownText(text string, width int) string {
	if text == "" {
		return ""
	}
	out, err := gruff.Render(text, gruff.WithDark(), gruff.WithWordWrap(width))
	if err != nil {
		return text
	}
	// gruff already trims its own leading/trailing blank rows, so the block
	// can go straight onto the card surface.
	return markdownSurfaceBG(out)
}
