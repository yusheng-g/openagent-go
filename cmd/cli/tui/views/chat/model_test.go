package chat

import (
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
	"context"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/lucasb-eyer/go-colorful"

	"github.com/mattn/go-runewidth"

	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/layout"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/theme"
	"github.com/yusheng-g/openagent-go/cmd/cli/tui/utils"
)

func newTestModel() *Model {
	ctx, cancel := context.WithCancel(context.Background())
	return NewModel(ctx, cancel, "/tmp", "test-version", "manual", "", nil)
}

func TestUpKeyFreezesAutoScroll(t *testing.T) {
	m := newTestModel()
	upd, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m2 := upd.(*Model)
	if m2.needAutoScroll {
		t.Error("up key on empty input should freeze auto-scroll")
	}
}

func TestWheelUpFreezesAutoScroll(t *testing.T) {
	m := newTestModel()
	m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: strings.Repeat("line\n", 50), TurnId: 0})
	m.chatViewport.SetContent(m.renderMessages())
	upd, _ := m.Update(tea.MouseWheelMsg{X: 5, Y: 0, Button: tea.MouseWheelUp})
	m2 := upd.(*Model)
	if m2.needAutoScroll {
		t.Error("wheel up should freeze auto-scroll")
	}
}

func TestWheelDownSyncsAutoScrollWithBottom(t *testing.T) {
	m := newTestModel()
	m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: strings.Repeat("line\n", 50), TurnId: 0})
	m.chatViewport.SetContent(m.renderMessages())
	m.chatViewport.GotoTop()
	upd, _ := m.Update(tea.MouseWheelMsg{X: 5, Y: 0, Button: tea.MouseWheelDown})
	m2 := upd.(*Model)
	if m2.needAutoScroll != m2.isNearBottom() {
		t.Error("wheel down should re-enable auto-scroll exactly when near bottom")
	}
}

func TestWheelIgnoredOutsideViewport(t *testing.T) {
	m := newTestModel()
	upd, _ := m.Update(tea.MouseWheelMsg{X: 5, Y: 999, Button: tea.MouseWheelUp})
	m2 := upd.(*Model)
	if !m2.needAutoScroll {
		t.Error("wheel below the viewport must not touch auto-scroll")
	}
	if m2.chatViewport.YOffset() != 0 {
		t.Error("wheel below the viewport must not scroll")
	}
}

func TestUsageUpdateSetsTokens(t *testing.T) {
	m := newTestModel()
	upd, _ := m.Update(usageUpdateMsg{used: 12300, total: 1000000})
	m2 := upd.(*Model)
	if m2.usedTokens != 12300 || m2.contextSize != 1000000 {
		t.Errorf("usage = %d/%d, want 12300/1000000", m2.usedTokens, m2.contextSize)
	}
	// Sidebar shows the consumed context against the window.
	if got := m2.contextValue(); got != "12.3k / 1M tokens" {
		t.Errorf("contextValue = %q, want %q", got, "12.3k / 1M tokens")
	}
	// A model that never learned the window shows the used count alone
	// (total 0 falls through — fresh model so no window is remembered).
	upd, _ = newTestModel().Update(usageUpdateMsg{used: 42, total: 0})
	if got := upd.(*Model).contextValue(); got != "42 tokens" {
		t.Errorf("contextValue = %q, want %q", got, "42 tokens")
	}
}

func TestFormatTokens(t *testing.T) {
	cases := map[int]string{
		0: "0", 834: "834", 12300: "12.3k", 1000000: "1M", 2500000: "2.5M",
	}
	for n, want := range cases {
		if got := formatTokens(n); got != want {
			t.Errorf("formatTokens(%d) = %q, want %q", n, got, want)
		}
	}
}

// TestPromptCountTracked guards the sidebar's turn counter: live sends,
// queued sends and replayed history each count; /new and a session switch
// reset it.
func TestPromptCountTracked(t *testing.T) {
	m := newTestModel()
	m.inChat = true
	m.activeSessionID = "sess-1"
	m.chatTextarea.SetValue("one")
	m2, _ := enterKey(m)
	if m2.promptCount != 1 {
		t.Errorf("promptCount = %d, want 1 after first send", m2.promptCount)
	}
	// A prompt while one is in flight queues (and counts) immediately.
	m2.chatTextarea.SetValue("two")
	m3, _ := enterKey(m2)
	if m3.promptCount != 2 {
		t.Errorf("promptCount = %d, want 2 after queued send", m3.promptCount)
	}
	// Replayed history counts too.
	m4, _ := m3.Update(userMessageMsg{text: "replayed"})
	if m4.(*Model).promptCount != 3 {
		t.Errorf("promptCount = %d, want 3 after replayed user message", m4.(*Model).promptCount)
	}
	// A fresh /new resets the counter with the transcript.
	m4.(*Model).chatTextarea.SetValue("/new")
	m5, _ := enterKey(m4.(*Model))
	if m5.promptCount != 0 {
		t.Errorf("promptCount = %d, want 0 after /new", m5.promptCount)
	}
}

// TestSidebarShowsContextTurnsAndPlanProgress checks the right sidebar's
// data section: context usage, turn count, and the plan progress title.
func TestSidebarShowsContextTurnsAndPlanProgress(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 36
	m.activeSessionID = "sess-1"
	m.usedTokens = 12300
	m.contextSize = 1000000
	m.promptCount = 7
	m.planEntries = []openacp.PlanEntry{
		{Content: "step one", Status: "completed"},
		{Content: "step two", Status: "in_progress"},
	}
	right := utils.StripANSI(m.renderRight())
	for _, want := range []string{"12.3k / 1M tokens", "Turns", "7", "Plans 1/2", "[▶] step two"} {
		if !strings.Contains(right, want) {
			t.Errorf("sidebar missing %q:\n%s", want, right)
		}
	}
}

func TestModeUpdateChangesBadge(t *testing.T) {
	m := newTestModel()
	if m.mode != "manual" {
		t.Fatalf("initial mode = %q, want manual", m.mode)
	}
	upd, _ := m.Update(modeUpdateMsg{mode: "auto"})
	if got := upd.(*Model).mode; got != "auto" {
		t.Errorf("mode = %q, want auto", got)
	}
}

func TestEscClearsInputFirst(t *testing.T) {
	m := newTestModel()
	m.chatTextarea.SetValue("hello")
	upd, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m2 := upd.(*Model)
	if m2.chatTextarea.Value() != "" {
		t.Error("first esc should clear the input")
	}
	if m2.statusText != "" {
		t.Error("clearing input must not touch status text")
	}
}

func TestEscCancelsInFlightPrompt(t *testing.T) {
	m := newTestModel()
	m.loading = true
	m.activeSessionID = "sess-1"
	// No ACP session wired (nil) — esc must stand down instead of panicking.
	m.statusText = "Running..."
	upd, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if got := upd.(*Model).statusText; got != "Running..." {
		t.Errorf("esc with nil acpSession must not change status, got %q", got)
	}
}

func TestCtrlCClearsInputFirst(t *testing.T) {
	m := newTestModel()
	m.chatTextarea.SetValue("hello")
	upd, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd != nil {
		t.Error("ctrl+c with pending input must not quit")
	}
	m2 := upd.(*Model)
	if m2.chatTextarea.Value() != "" {
		t.Error("ctrl+c should clear the input")
	}
}

func TestCtrlCCancelsWhenLoading(t *testing.T) {
	m := newTestModel()
	m.loading = true
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd != nil {
		t.Error("ctrl+c while loading must cancel, not quit")
	}
}

func TestCtrlCQuitsWhenIdle(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("ctrl+c while idle should quit")
	}
}

// ── slash commands & command panel ──

func enterKey(m *Model) (*Model, tea.Cmd) {
	upd, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return upd.(*Model), cmd
}

func TestSlashExitQuits(t *testing.T) {
	m := newTestModel()
	m.chatTextarea.SetValue("/exit")
	_, cmd := enterKey(m)
	if cmd == nil {
		t.Fatal("send /exit should quit")
	}
}

func TestSlashUnknownCommand(t *testing.T) {
	m := newTestModel()
	m.chatTextarea.SetValue("/nope")
	m2, _ := enterKey(m)
	if m2.statusText != "Unknown command: /nope" {
		t.Errorf("statusText = %q, want unknown-command hint", m2.statusText)
	}
	if m2.chatTextarea.Value() != "" {
		t.Error("slash input should be consumed")
	}
}

func TestSlashNewEmptySessionGuard(t *testing.T) {
	m := newTestModel()
	m.chatTextarea.SetValue("/new")
	m2, _ := enterKey(m)
	if m2.statusText != "No active session to reset" {
		t.Errorf("empty session guard: statusText = %q", m2.statusText)
	}
	if m2.loading {
		t.Error("guarded /new must not start loading")
	}
}

func TestToggleThinkingFlipsConfig(t *testing.T) {
	m := newTestModel()
	m.chatTextarea.SetValue("/toggle_thinking")
	m2, _ := enterKey(m)
	if !m2.visibleConfig.ExpandThinking {
		t.Error("/toggle_thinking should expand thinking (default collapsed)")
	}
}

func TestPanelCtrlPOpenAndFilter(t *testing.T) {
	m := newTestModel()
	m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	if !m.panelOpen {
		t.Fatal("ctrl+p should open the panel")
	}
	for _, r := range "thin" {
		m.Update(tea.KeyPressMsg{Code: r})
	}
	cmds := m.buildPanelCommands()
	if len(cmds) != 1 || cmds[0].slash != "/toggle_thinking" {
		t.Fatalf("filter 'thin' should leave one command, got %+v", cmds)
	}
}

func TestPanelExecuteTogglesAndCloses(t *testing.T) {
	m := newTestModel()
	m.panelOpen = true
	m.panelFilter = "thin"
	m.panelIdx = 0
	m2, _ := enterKey(m)
	if m2.panelOpen {
		t.Error("panel should close after executing a command")
	}
	if !m2.visibleConfig.ExpandThinking {
		t.Error("panel selection should expand thinking (default collapsed)")
	}
}

func TestPanelEscCloses(t *testing.T) {
	m := newTestModel()
	m.panelOpen = true
	m.panelFilter = "t"
	upd, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if upd.(*Model).panelOpen {
		t.Error("esc should close the panel")
	}
}

func TestToggleIconTracksConfig(t *testing.T) {
	m := newTestModel()
	if m.toggleIcon("/toggle_thinking") != "○" {
		t.Error("thinking toggle should start off (collapsed by default)")
	}
	m.chatTextarea.SetValue("/toggle_thinking")
	m2, _ := enterKey(m)
	if m2.toggleIcon("/toggle_thinking") != "●" {
		t.Error("toggle icon should reflect config after toggle")
	}
}

// TestSlashNewReturnsToWelcome guards /new's semantics: it returns to the
// welcome page (the state a fresh boot starts from) without eagerly
// creating a session — the next prompt lazily creates one, and an
// in-flight prompt is abandoned.
func TestSlashNewReturnsToWelcome(t *testing.T) {
	m := newTestModel()
	m.inChat = true
	m.activeSessionID = "sess-old"
	m.messages = append(m.messages, ChatMessage{Role: "user", Content: "hi", TurnId: 0})
	m.usedTokens, m.contextSize, m.promptCount = 123, 0, 2
	m.loading = true

	m.chatTextarea.SetValue("/new")
	m2, _ := enterKey(m)
	if m2.inChat {
		t.Error("/new should return to the welcome page")
	}
	if m2.activeSessionID != "" {
		t.Errorf("activeSessionID = %q, want empty (next prompt creates a fresh session)", m2.activeSessionID)
	}
	if len(m2.messages) != 0 {
		t.Error("/new should clear the transcript")
	}
	if m2.usedTokens != 0 || m2.contextSize != 0 || m2.promptCount != 0 {
		t.Errorf("sidebar counters = %d/%d/%d, want all 0", m2.usedTokens, m2.contextSize, m2.promptCount)
	}
	if m2.loading {
		t.Error("/new should drop the in-flight prompt state")
	}
}

// TestNewSessionMsgBareCreationKeepsTranscript guards the bare newSessionMsg
// fallback: it adopts the fresh session but never touches the transcript —
// transcript resets belong to /new, which does not go through here.
func TestNewSessionMsgBareCreationKeepsTranscript(t *testing.T) {
	m := newTestModel()
	m.messages = append(m.messages, ChatMessage{Role: "user", Content: "hi", TurnId: 0})
	m.usedTokens, m.contextSize, m.promptCount = 123, 0, 2
	upd, _ := m.Update(newSessionMsg{sessionID: "sess-new"})
	m2 := upd.(*Model)
	if m2.activeSessionID != "sess-new" {
		t.Errorf("activeSessionID = %q, want sess-new", m2.activeSessionID)
	}
	if len(m2.messages) != 1 {
		t.Error("bare creation must not clear the transcript")
	}
	if m2.loading {
		t.Error("new session should clear loading")
	}
}

// ── lazy first session ──

// TestFirstEnterCreatesSessionLazily guards the deferred-session boot: the
// ACP session is no longer created at program start, so the very first
// prompt (Enter with no active session) must queue the text, place it in
// the transcript, and fire newSessionCmd instead of prompting a
// nonexistent session.
func TestFirstEnterCreatesSessionLazily(t *testing.T) {
	m := newTestModel()
	if m.activeSessionID != "" {
		t.Fatalf("boot must start without a session, got %q", m.activeSessionID)
	}
	m.chatTextarea.SetValue("first question")
	m2, cmd := enterKey(m)
	if !m2.inChat {
		t.Error("first prompt must enter the chat view")
	}
	if !m2.loading {
		t.Error("first prompt should show loading while the session is created")
	}
	if len(m2.inputQueue) != 1 || m2.inputQueue[0] != "first question" {
		t.Errorf("first prompt must be queued, got %v", m2.inputQueue)
	}
	if n := len(m2.messages); n != 1 || m2.messages[n-1].Content != "first question" {
		t.Errorf("first prompt must appear in the transcript, got %+v", m2.messages)
	}
	if m2.statusText != "Creating new session..." {
		t.Errorf("statusText = %q, want %q", m2.statusText, "Creating new session...")
	}
	if m2.chatTextarea.Value() != "" {
		t.Error("input should be cleared after queuing the first prompt")
	}
	if cmd == nil {
		t.Fatal("first prompt must fire the new-session command")
	}
}

// TestNewSessionMsgDrainsDeferredFirstPrompt guards the newSessionMsg
// side of lazy creation: once the session lands, the queued first prompt
// is sent (the transcript already holds it — it must not be reset or
// duplicated), and loading stays on until prompt_done.
func TestNewSessionMsgDrainsDeferredFirstPrompt(t *testing.T) {
	m := newTestModel()
	m.inputQueue = []string{"first question"}
	m.messages = append(m.messages, ChatMessage{Role: "user", Content: "first question", TurnId: 0})
	m.loading = true
	upd, _ := m.Update(newSessionMsg{sessionID: "sess-boot"})
	m2 := upd.(*Model)
	if m2.activeSessionID != "sess-boot" {
		t.Errorf("activeSessionID = %q, want sess-boot", m2.activeSessionID)
	}
	if len(m2.inputQueue) != 0 {
		t.Errorf("inputQueue must drain, got %v", m2.inputQueue)
	}
	if len(m2.messages) != 1 || m2.messages[0].Role != "user" || m2.messages[0].Content != "first question" {
		t.Errorf("deferred prompt must stay in the transcript exactly once, got %+v", m2.messages)
	}
	if !m2.loading {
		t.Error("drained prompt must keep loading on until prompt_done")
	}
	if m2.statusText != "Running..." {
		t.Errorf("statusText = %q, want %q", m2.statusText, "Running...")
	}
}

// testAcpSession returns a non-nil *openacp.Session backed by in-memory
// pipes; the TUI tests only need the non-nil pointer for connection guards,
// never a live agent conversation.
func testAcpSession(t *testing.T) *openacp.Session {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	var stdin strings.Builder
	return openacp.NewClient("tui-test", "test").ConnectIO(ctx, &stdin, strings.NewReader(""))
}

// testModelOptions builds a "model" select option like a real agent would
// return in session/new's config options.
func testModelOptions() []openacp.SessionConfigOption {
	return []openacp.SessionConfigOption{{
		ID:           "model",
		Name:         "Model",
		Category:     "model",
		Type:         "select",
		CurrentValue: "m1",
		Options:      []openacp.SessionConfigOptValue{{Value: "m1"}, {Value: "m2"}},
	}}
}

// TestBootReadyMsgFetchesConfigOptions guards the welcome-page badges: the
// boot ready message carries no options (lazy boot, no session), so the
// model must fetch the session-less config options right away — the
// welcome header then renders the default model badge from the cache.
func TestBootReadyMsgFetchesConfigOptions(t *testing.T) {
	m := newTestModel()
	m.acpSession = testAcpSession(t)
	_, cmd := m.Update(acpReadyMsg{})
	if cmd == nil {
		t.Fatal("boot without options must fire the list-config-options command")
	}
	upd, _ := m.Update(configOptionsMsg{opts: testModelOptions()})
	m2 := upd.(*Model)
	if got := m2.currentModel(); got != "m1" {
		t.Errorf("currentModel = %q, want m1 (welcome badge source)", got)
	}
	if got := m2.currentThoughtLevel(); got != "" {
		t.Errorf("currentThoughtLevel = %q, want empty (fixture has none)", got)
	}
	// With options already cached, a repeated ready message must not refetch.
	_, cmd2 := m2.Update(acpReadyMsg{})
	if cmd2 != nil {
		t.Error("ready with cached options must not refetch")
	}
}

// ── lazy /models ──

// TestModelsPanelFetchesConfigOptionsLazily guards the cold-start model
// picker: with the backend connected but no session yet, /models must not
// bounce with "No model options" nor create a session just to browse — it
// fires session/list_config_options and the picker opens when the options
// arrive.
func TestModelsPanelFetchesConfigOptionsLazily(t *testing.T) {
	m := newTestModel()
	m.acpSession = testAcpSession(t)
	m.chatTextarea.SetValue("/models")
	m2, cmd := enterKey(m)
	if !m2.pendingModelsPanel {
		t.Error("/models before any session must arm pendingModelsPanel")
	}
	if m2.panelOpen {
		t.Error("no picker yet — the config options are still being fetched")
	}
	if m2.statusText != "Fetching model list..." {
		t.Errorf("statusText = %q, want %q", m2.statusText, "Fetching model list...")
	}
	if cmd == nil {
		t.Fatal("/models must fire the list-config-options command")
	}
	// A second /models while the list is in flight must not stack commands.
	_, cmd2 := m2.openModelsPanel()
	if cmd2 != nil {
		t.Error("in-flight list must not fire a second list-config-options command")
	}
}

// TestConfigOptionsMsgOpensPendingModelsPanel guards the completion side of
// the cold-start /models fetch: the picker opens with the delivered options
// and the pending flag clears.
func TestConfigOptionsMsgOpensPendingModelsPanel(t *testing.T) {
	m := newTestModel()
	m.pendingModelsPanel = true
	m.loading = true
	upd, _ := m.Update(configOptionsMsg{opts: testModelOptions()})
	m2 := upd.(*Model)
	if !m2.panelOpen || m2.panelMode != panelModeModels {
		t.Errorf("picker must open on models mode, open=%v mode=%v", m2.panelOpen, m2.panelMode)
	}
	if got := m2.modelOptions(); len(got) != 2 || got[0] != "m1" || got[1] != "m2" {
		t.Errorf("modelOptions = %v, want [m1 m2]", got)
	}
	if m2.pendingModelsPanel {
		t.Error("pendingModelsPanel must clear once the options arrive")
	}
	if m2.loading {
		t.Error("loading must clear once the options arrive")
	}
	if m2.statusText != "" {
		t.Errorf("statusText = %q, want empty", m2.statusText)
	}
}

// TestConfigOptionsMsgErrClearsPendingModels ensures a failed fetch
// surfaces the error instead of leaving the picker request armed.
func TestConfigOptionsMsgErrClearsPendingModels(t *testing.T) {
	m := newTestModel()
	m.pendingModelsPanel = true
	upd, _ := m.Update(configOptionsMsg{err: fmt.Errorf("boom")})
	m2 := upd.(*Model)
	if m2.pendingModelsPanel {
		t.Error("failed fetch must clear pendingModelsPanel")
	}
	if m2.panelOpen {
		t.Error("failed fetch must not open the picker")
	}
	if !strings.Contains(m2.statusText, "List models failed") {
		t.Errorf("statusText = %q, want failure notice", m2.statusText)
	}
}

// TestModelPickWithoutSessionCreatesThenSets guards the cold-start pick: a
// model chosen from the session-less picker creates a session lazily and
// applies the choice via set_config_option once the session lands.
func TestModelPickWithoutSessionCreatesThenSets(t *testing.T) {
	m := newTestModel()
	m.acpSession = testAcpSession(t)
	m.configOptions = testModelOptions() // as cached by the list fetch
	m.panelOpen = true
	m.panelMode = panelModeModels
	m.panelIdx = 1 // pick "m2" (index 0 is the current default)
	upd, cmd := m.execSelectedModel()
	m2 := upd.(*Model)
	if m2.panelOpen {
		t.Error("picking must close the picker")
	}
	if m2.pendingConfigSet == nil || m2.pendingConfigSet.id != "model" || m2.pendingConfigSet.value != "m2" {
		t.Errorf("pendingConfigSet = %+v, want model/m2", m2.pendingConfigSet)
	}
	if cmd == nil {
		t.Fatal("picking without a session must fire the new-session command")
	}
	// The session lands: the pending pick must turn into a set_config_option
	// command (non-nil), and the transcript stays untouched.
	m3, setCmd := m2.Update(newSessionMsg{sessionID: "sess-boot", configOptions: testModelOptions()})
	m4 := m3.(*Model)
	if m4.pendingConfigSet != nil {
		t.Error("pendingConfigSet must clear once applied")
	}
	if setCmd == nil {
		t.Fatal("session creation must fire the set_config_option command")
	}
	if len(m4.messages) != 0 {
		t.Error("model pick must not touch the transcript")
	}
}

// TestNewSessionMsgErrClearsPendingModels ensures a failed on-demand
// creation surfaces the error instead of leaving the picker request armed.
func TestNewSessionMsgErrClearsPendingModels(t *testing.T) {
	m := newTestModel()
	m.pendingModelsPanel = true
	upd, _ := m.Update(newSessionMsg{err: fmt.Errorf("boom")})
	m2 := upd.(*Model)
	if m2.pendingModelsPanel {
		t.Error("failed creation must clear pendingModelsPanel")
	}
	if m2.panelOpen {
		t.Error("failed creation must not open the picker")
	}
	if !strings.Contains(m2.statusText, "New session failed") {
		t.Errorf("statusText = %q, want failure notice", m2.statusText)
	}
}

// TestSecondEnterWhileCreatingSessionOnlyQueues guards the in-flight case:
// pressing Enter again while the first session is still being created must
// only queue the text — not fire a second new-session command.
func TestSecondEnterWhileCreatingSessionOnlyQueues(t *testing.T) {
	m := newTestModel()
	m.inputQueue = []string{"first question"}
	m.loading = true // first newSessionCmd in flight
	m.chatTextarea.SetValue("second question")
	m2, cmd := enterKey(m)
	if len(m2.inputQueue) != 2 || m2.inputQueue[1] != "second question" {
		t.Errorf("second prompt must queue behind the first, got %v", m2.inputQueue)
	}
	if cmd != nil {
		t.Error("second enter during session creation must not fire another new-session command")
	}
	if !m2.loading {
		t.Error("loading must persist while the session is being created")
	}
	if n := len(m2.messages); n != 1 || m2.messages[n-1].Content != "second question" {
		t.Errorf("second prompt must appear in the transcript, got %+v", m2.messages)
	}
}

// ── tab suggestion acceptance ──

func TestTabAcceptsSuggestionOnEmptyInput(t *testing.T) {
	m := newTestModel()
	m.suggestion = "帮我做一个 PPT"
	old := m.suggestion
	upd, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := upd.(*Model)
	if m2.chatTextarea.Value() != "帮我做一个 PPT" {
		t.Errorf("tab should fill the suggestion, got %q", m2.chatTextarea.Value())
	}
	if m2.suggestion == old {
		t.Error("suggestion should rotate to the next one after accept")
	}
}

func TestTabDoesNotOverwriteInput(t *testing.T) {
	m := newTestModel()
	m.suggestion = "should not replace this"
	m.chatTextarea.SetValue("user typed text")
	upd, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m2 := upd.(*Model)
	// With text present tab falls through to the textarea: the typed text
	// must not be replaced by the suggestion.
	if got := m2.chatTextarea.Value(); got == m.suggestion {
		t.Error("tab must not replace typed input with the suggestion")
	}
	if !strings.HasPrefix(m2.chatTextarea.Value(), "user typed text") {
		t.Errorf("typed text should be preserved, got %q", m2.chatTextarea.Value())
	}
}

// ── command panel rendering ──

// The panel must keep one command per line: every rendered line has the
// same character count (rows that would overflow get wrapped by lipgloss,
// adding lines and breaking alignment). runewidth cannot be used here — it
// reports box-drawing runes (━┃┏) as 2 columns, which is a measurement
// artefact, not the layout.
func TestPanelRenderRowsConsistent(t *testing.T) {
	m := newTestModel()
	m.width = 120
	m.height = 30
	m.panelOpen = true
	rendered := m.renderCommandPanel()

	lines := strings.Split(utils.StripANSI(rendered), "\n")
	// title + blank + rows + blank + footer; no top/bottom frame lines.
	wantLines := len(allPanelCommands()) + 4
	if got := len(lines); got != wantLines {
		t.Fatalf("panel has %d lines, want %d — row wrapping detected", got, wantLines)
	}
	exp := utf8.RuneCountInString(lines[0])
	for i, ln := range lines {
		if n := utf8.RuneCountInString(ln); n != exp && n != exp-1 {
			t.Errorf("panel line %d = %d runes, want %d±1: %q", i, n, exp, ln)
		}
	}
}

// Popup panels are borderless: the Ctrl+P command palette (like sessions,
// models, help, search, edit, plugins) draws no frame at all — no ┃ side
// borders and no ┏┓┗┛ edges.
func TestPopupPanelHasNoFrame(t *testing.T) {
	m := newTestModel()
	m.panelOpen = true
	rendered := utils.StripANSI(m.renderCommandPanel())
	for _, glyph := range []string{"┃", "┏", "┓", "┗", "┛"} {
		if strings.Contains(rendered, glyph) {
			t.Errorf("popup panel must not draw %q (borderless), got:\n%s", glyph, rendered)
		}
	}
}

// TestPopupPanelsBorderless guards the popup look end-to-end: every popup
// renders as a borderless centered block (no frame glyphs) carrying its
// title and an esc hint.
func TestPopupPanelsBorderless(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.sessionItems = []sessionItem{{id: "s1", title: "Session one"}}
	m.pluginItems = []string{"plugin-a", "plugin-b"}
	m.configOptions = sampleConfigOptions()
	cases := []struct {
		name, title string
		render      func() string
	}{
		{"sessions", "Sessions", func() string { return m.renderSessionPanel() }},
		{"models", "Models", func() string { return m.renderModelPanel() }},
		{"help", "Help", func() string { return m.renderHelpPanel() }},
		{"search", "Search", func() string { return m.renderSearchPanel() }},
		{"edit", "Edit message", func() string { return m.renderEditPanel() }},
		{"plugins", "Plugins", func() string { return m.renderPluginsPanel() }},
		{"commands", "Commands", func() string { return m.renderCommandPanel() }},
	}
	for _, tc := range cases {
		got := utils.StripANSI(tc.render())
		for _, glyph := range []string{"┃", "┏", "┓", "┗", "┛"} {
			if strings.Contains(got, glyph) {
				t.Errorf("%s popup must be borderless (no %q):\n%s", tc.name, glyph, got)
			}
		}
		if !strings.Contains(got, tc.title) {
			t.Errorf("%s popup missing title %q:\n%s", tc.name, tc.title, got)
		}
		if !strings.Contains(got, "esc") {
			t.Errorf("%s popup missing esc hint:\n%s", tc.name, got)
		}
	}
}

// TestPopupPanelTextLeftAligned guards the opencode-style popup layout: the
// title, the first option row, and the bottom key hints all start on the
// same left edge (rows carry no leading icon/marker column).
func TestPopupPanelTextLeftAligned(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.sessionItems = []sessionItem{{id: "s1", title: "Session one"}}
	m.pluginItems = []string{"plugin-a"}
	m.configOptions = sampleConfigOptions()
	m.messages = []ChatMessage{
		{Role: "user", Content: "hello world"},
		{Role: "assistant", Content: "hi there"},
	}
	cases := []struct {
		name   string
		render func() string
	}{
		{"commands", func() string { return m.renderCommandPanel() }},
		{"sessions", func() string { return m.renderSessionPanel() }},
		{"models", func() string { return m.renderModelPanel() }},
		{"help", func() string { return m.renderHelpPanel() }},
		{"search", func() string { return m.renderSearchPanel() }},
		{"edit", func() string { return m.renderEditPanel() }},
		{"plugins", func() string { return m.renderPluginsPanel() }},
	}
	for _, tc := range cases {
		got := utils.StripANSI(tc.render())
		var nonBlank []int
		for _, ln := range strings.Split(got, "\n") {
			trim := strings.TrimLeft(ln, " ")
			if trim == "" {
				continue
			}
			nonBlank = append(nonBlank, len(ln)-len(trim))
		}
		if len(nonBlank) < 3 {
			t.Fatalf("%s popup: expected title/rows/footer, got %d non-blank lines:\n%s", tc.name, len(nonBlank), got)
		}
		titleCol, rowCol, footerCol := nonBlank[0], nonBlank[1], nonBlank[len(nonBlank)-1]
		if titleCol != rowCol || rowCol != footerCol {
			t.Errorf("%s popup left edges misaligned: title=%d first row=%d footer=%d:\n%s",
				tc.name, titleCol, rowCol, footerCol, got)
		}
	}
}

// TestPopupSelectedRowFilled guards the opencode-style selection: a popup's
// selected row is filled edge-to-edge with the CommandActive peach (not a
// leading ▶ marker).
func TestPopupSelectedRowFilled(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.panelIdx = 1
	m.sessionItems = []sessionItem{{id: "s1", title: "Session one"}, {id: "s2", title: "Session two"}}
	m.pluginItems = []string{"plugin-a", "plugin-b"}
	m.configOptions = sampleConfigOptions()
	m.messages = []ChatMessage{
		{Role: "user", Content: "old ask"},
		{Role: "assistant", Content: "reply"},
		{Role: "user", Content: "new ask"},
	}
	for _, tc := range []struct {
		name   string
		render func() string
	}{
		{"commands", func() string { return m.renderCommandPanel() }},
		{"sessions", func() string { return m.renderSessionPanel() }},
		{"models", func() string { return m.renderModelPanel() }},
		{"plugins", func() string { return m.renderPluginsPanel() }},
		{"edit", func() string { return m.renderEditPanel() }},
	} {
		got := tc.render()
		if !strings.Contains(got, "48;2;250;178;131") {
			t.Errorf("%s popup selected row must carry the CommandActive fill:\n%s", tc.name, got)
		}
	}
}

// ── sessions & models panels ──

func sampleConfigOptions() []openacp.SessionConfigOption {
	return []openacp.SessionConfigOption{
		{
			ID: "mode", Category: "mode", Type: "select",
			Name:         "Session Mode",
			CurrentValue: "manual",
			Options: []openacp.SessionConfigOptValue{
				{Value: "auto", Name: "Auto", Description: "Fully automated processing (HIGH RISK), AI will NOT seek your approval"},
				{Value: "manual", Name: "Manual", Description: "Your approval is required for AI to perform NONE-READ-ONLY operations"},
				{Value: "plan", Name: "Plan", Description: "Present the plan first, AI will execute it according to the plan"},
			},
		},
		{
			ID: "model", Category: "model", Type: "select",
			CurrentValue: "deepseek-v4",
			Options: []openacp.SessionConfigOptValue{
				{Value: "deepseek-v4"},
				{Value: "gpt-4o"},
			},
		},
	}
}

func TestOpenSessionsPanelSetsMode(t *testing.T) {
	m := newTestModel()
	m2, _ := m.openSessionsPanel()
	mm := m2.(*Model)
	if !mm.panelOpen || mm.panelMode != panelModeSessions {
		t.Errorf("panelOpen=%v panelMode=%d, want open sessions panel", mm.panelOpen, mm.panelMode)
	}
	if mm.sessionsLoading {
		t.Error("sessionsLoading must be false when backend is nil (guard)")
	}
}

func TestLoadSessionsMsgPopulatesItems(t *testing.T) {
	m := newTestModel()
	m.sessionsLoading = true
	upd, _ := m.Update(loadSessionsMsg{items: []sessionItem{{id: "a", title: "A"}, {id: "b", title: "B"}}})
	m2 := upd.(*Model)
	if m2.sessionsLoading {
		t.Error("sessionsLoading should clear on response")
	}
	if len(m2.sessionItems) != 2 || m2.sessionItems[1].id != "b" {
		t.Errorf("sessionItems = %+v", m2.sessionItems)
	}
}

func TestLoadSessionsMsgError(t *testing.T) {
	m := newTestModel()
	upd, _ := m.Update(loadSessionsMsg{err: fmt.Errorf("boom")})
	if got := upd.(*Model).statusText; got != "List sessions failed: boom" {
		t.Errorf("statusText = %q", got)
	}
}

func TestExecSelectedSessionClearsAndLoads(t *testing.T) {
	m := newTestModel()
	m.messages = append(m.messages, ChatMessage{Role: "user", Content: "old", TurnId: 0})
	m.sessionItems = []sessionItem{{id: "x", title: "X"}}
	m.panelOpen = true
	m.panelMode = panelModeSessions
	m.panelIdx = 0
	upd, cmd := m.panelExecute()
	m2 := upd.(*Model)
	if m2.panelOpen {
		t.Error("panel should close after selecting a session")
	}
	if len(m2.messages) != 0 {
		t.Error("transcript must be cleared before replay")
	}
	if !m2.replaying || !m2.loading {
		t.Error("session switch should set replaying+loading")
	}
	if cmd == nil {
		t.Fatal("expected a load cmd")
	}
}

// TestExecSelectedSessionEntersChatView guards the welcome-page switch: a
// historical session picked straight from the welcome screen must flip
// inChat, or the replayed transcript stays invisible behind the logo.
func TestExecSelectedSessionEntersChatView(t *testing.T) {
	m := newTestModel()
	m.inChat = false
	m.sessionItems = []sessionItem{{id: "current", title: "Current"}, {id: "past", title: "Past"}}
	m.activeSessionID = "current"
	m.panelOpen = true
	m.panelMode = panelModeSessions
	m.panelIdx = 1 // pick the historical session
	upd, _ := m.panelExecute()
	if !upd.(*Model).inChat {
		t.Error("switching to a historical session from the welcome page must enter the chat view")
	}
}

func TestSessionLoadedMsgSwitchesState(t *testing.T) {
	m := newTestModel()
	m.replaying = true
	m.loading = true
	upd, _ := m.Update(sessionLoadedMsg{sessionID: "sess-new", configOptions: sampleConfigOptions(), mode: "plan"})
	m2 := upd.(*Model)
	if m2.activeSessionID != "sess-new" {
		t.Errorf("activeSessionID = %q", m2.activeSessionID)
	}
	if m2.mode != "plan" {
		t.Errorf("mode = %q", m2.mode)
	}
	if m2.replaying || m2.loading {
		t.Error("replaying/loading must clear on session load")
	}
	if m2.currentModel() != "deepseek-v4" {
		t.Errorf("currentModel = %q", m2.currentModel())
	}
}

func TestModelOptionsAndCurrent(t *testing.T) {
	m := newTestModel()
	m.configOptions = sampleConfigOptions()
	opts := m.modelOptions()
	if len(opts) != 2 || opts[1] != "gpt-4o" {
		t.Errorf("modelOptions = %v", opts)
	}
	if m.currentModel() != "deepseek-v4" {
		t.Errorf("currentModel = %q", m.currentModel())
	}
}

func TestOpenModelsPanelEmptyCloses(t *testing.T) {
	m := newTestModel() // no config options cached
	upd, _ := m.executeCommand(panelCommand{slash: "/models", action: actionModels})
	m2 := upd.(*Model)
	if m2.panelOpen {
		t.Error("models panel with no options must close itself")
	}
}

func TestConfigSetMsgUpdatesAndStatus(t *testing.T) {
	m := newTestModel()
	upd, _ := m.Update(configSetMsg{configOptions: sampleConfigOptions()})
	m2 := upd.(*Model)
	if m2.currentModel() != "deepseek-v4" {
		t.Errorf("currentModel = %q", m2.currentModel())
	}
	if m2.notifyMsg != "Model: deepseek-v4" {
		t.Errorf("toast = %q, want model toast", m2.notifyMsg)
	}
}

func TestUserMessageMsgAppendsUser(t *testing.T) {
	m := newTestModel()
	upd, _ := m.Update(userMessageMsg{text: "replayed hi"})
	m2 := upd.(*Model)
	if n := len(m2.messages); n != 1 || m2.messages[n-1].Role != "user" || m2.messages[n-1].Content != "replayed hi" {
		t.Errorf("messages = %+v", m2.messages)
	}
}

func TestEnterHeldDuringReplay(t *testing.T) {
	m := newTestModel()
	m.replaying = true
	m.chatTextarea.SetValue("hello")
	upd, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := upd.(*Model)
	if cmd != nil {
		t.Error("enter during replay must not send")
	}
	if m2.chatTextarea.Value() != "hello" {
		t.Error("input must be preserved during replay")
	}
}

// ── tool call display ──

func TestToolCallMsgAppendsAndUpdates(t *testing.T) {
	m := newTestModel()
	upd, _ := m.Update(toolCallMsg{id: "tc1", title: "bash_exec", status: toolRunning, input: `{"command":"ls"}`})
	m2 := upd.(*Model)
	if n := len(m2.messages); n != 1 {
		t.Fatalf("messages = %d, want 1", n)
	}
	tm := m2.messages[0]
	if tm.Role != "tool" || tm.ToolName != "bash_exec" || tm.ToolStatus != toolRunning {
		t.Errorf("tool msg = %+v", tm)
	}
	// Same ID updates the same row: status → done, no new message.
	upd, _ = m2.Update(toolCallMsg{id: "tc1", title: "bash_exec", status: toolDone, output: "ok"})
	m3 := upd.(*Model)
	if len(m3.messages) != 1 {
		t.Fatalf("update must reuse the row, got %d messages", len(m3.messages))
	}
	if m3.messages[0].ToolStatus != toolDone || m3.messages[0].ToolOutput != "ok" {
		t.Errorf("updated tool msg = %+v", m3.messages[0])
	}
}

func TestToolCallStatusMappingFailed(t *testing.T) {
	m := newTestModel()
	upd, _ := m.Update(toolCallMsg{id: "x", title: "read_file", status: toolFailed})
	if got := upd.(*Model).messages[0].ToolStatus; got != toolFailed {
		t.Errorf("status = %q, want failed", got)
	}
}

func TestFoldOutput(t *testing.T) {
	if got := foldOutput("a\nb\nc", 5); got != "a\nb\nc" {
		t.Errorf("short text must pass through, got %q", got)
	}
	got := foldOutput("1\n2\n3\n4\n5\n6\n7", 3)
	want := "1\n2\n3\n… (4 more lines)"
	if got != want {
		t.Errorf("foldOutput = %q, want %q", got, want)
	}
	// CRLF normalized
	got = foldOutput("1\r\n2\r\n3\r\n4", 2)
	if got != "1\n2\n… (2 more lines)" {
		t.Errorf("foldOutput CRLF = %q", got)
	}
}

func TestToolNameClassifiers(t *testing.T) {
	if !isShellTool("bash_exec") || !isShellTool("powershell_exec") || isShellTool("read_file") {
		t.Error("isShellTool misclassifies")
	}
	if !isSkillTool("skill:deploy") || isSkillTool("read_file") {
		t.Error("isSkillTool misclassifies")
	}
}

func TestThoughtCollapsedByDefault(t *testing.T) {
	m := newTestModel()
	m.messages = append(m.messages, ChatMessage{Role: "thought", Content: "secret reasoning", TurnId: 0})
	rendered := m.renderMessages()
	if strings.Contains(rendered, "secret reasoning") {
		t.Error("collapsed thought must not show its content by default")
	}
	if !strings.Contains(rendered, "Thought") {
		t.Error("collapsed thought should show a summary line")
	}
	m.visibleConfig.ExpandThinking = true
	if got := m.renderMessages(); !strings.Contains(got, "secret reasoning") {
		t.Error("expanded thought should render its content")
	}
	m.visibleConfig.ExpandThinking = false
	m.visibleConfig.ShowToolDetail = false
	m.messages = append(m.messages, ChatMessage{Role: "tool", TurnId: 0, ToolCallID: "a", ToolName: "read_file", ToolStatus: toolDone, ToolOutput: "data"})
	rendered = m.renderMessages()
	if !strings.Contains(rendered, "✓ read_file") {
		t.Errorf("tool row should render, got:\n%s", rendered)
	}
	if strings.Contains(rendered, "data") {
		t.Error("tool output must be hidden when ShowToolDetail is off")
	}
}

// TestThoughtSummaryStates covers the collapsed card's three one-liners:
// streaming shows "Thinking...", a measured span shows the duration, and a
// replayed span of unknown length shows a bare "Thought". None of them may
// leak the thought text.
func TestThoughtSummaryStates(t *testing.T) {
	m := newTestModel()
	streaming := ChatMessage{Role: "thought", Content: "mid-flight", TurnId: 0, ThoughtStart: time.Now()}
	m.loading = true
	if got, _ := m.renderMessageBlock(0, streaming, layout.GetViewWidth(m.width)); !strings.Contains(got, "Thinking...") || strings.Contains(got, "mid-flight") {
		t.Errorf("streaming thought should summarize to Thinking...:\n%s", utils.StripANSI(got))
	}

	m.loading = false
	done := ChatMessage{Role: "thought", Content: "past", TurnId: 0,
		ThoughtStart: time.Now().Add(-1500 * time.Millisecond), ThoughtEnd: time.Now()}
	if got, _ := m.renderMessageBlock(0, done, layout.GetViewWidth(m.width)); !strings.Contains(got, "Thought: 1.5s") || strings.Contains(got, "past") {
		t.Errorf("finished thought should show its duration:\n%s", utils.StripANSI(got))
	}

	replayed := ChatMessage{Role: "thought", Content: "from history", TurnId: 0}
	if got, _ := m.renderMessageBlock(0, replayed, layout.GetViewWidth(m.width)); !strings.Contains(got, "Thought") || strings.Contains(got, "from history") {
		t.Errorf("replayed thought should show a bare summary:\n%s", utils.StripANSI(got))
	}
}

// TestCloseTrailingThought guards the duration measurement: a trailing
// open thought gets its end stamped when the next message arrives or the
// turn finishes; a replayed thought (zero start) stays undated.
func TestCloseTrailingThought(t *testing.T) {
	m := newTestModel()
	m.messages = []ChatMessage{{Role: "thought", Content: "hmm", TurnId: 0, ThoughtStart: time.Now().Add(-time.Second)}}

	m.Update(agentMessageMsg{text: "answer"})
	if m.messages[0].ThoughtEnd.IsZero() {
		t.Error("assistant arrival should stamp the thought's end")
	}
	if m.messages[0].ThoughtEnd.Sub(m.messages[0].ThoughtStart) <= 0 {
		t.Error("stamped end must come after the start")
	}

	// promptDoneMsg stamps a still-trailing open thought (turn ended
	// without an assistant message).
	m.messages = []ChatMessage{{Role: "thought", Content: "hmm", TurnId: 0, ThoughtStart: time.Now()}}
	m.Update(promptDoneMsg{})
	if m.messages[0].ThoughtEnd.IsZero() {
		t.Error("turn completion should stamp the thought's end")
	}

	// Replayed thought: no start, so no end may be invented.
	m.messages = []ChatMessage{{Role: "thought", Content: "hmm", TurnId: 0}}
	m.Update(promptDoneMsg{})
	if !m.messages[0].ThoughtEnd.IsZero() {
		t.Error("replayed thought must stay undated")
	}
}

// ── input history ──

func TestAddToHistoryAndBrowse(t *testing.T) {
	m := newTestModel()
	m.addToHistory("first")
	m.addToHistory("second")
	m.addToHistory("third")
	if len(m.history) != 3 || m.historyIdx != 3 {
		t.Fatalf("history=%v idx=%d", m.history, m.historyIdx)
	}
	m.historyUp()
	if m.chatTextarea.Value() != "third" {
		t.Errorf("first historyUp -> %q, want third (newest)", m.chatTextarea.Value())
	}
	m.historyUp()
	if m.chatTextarea.Value() != "second" {
		t.Errorf("historyUp -> %q, want second", m.chatTextarea.Value())
	}
	m.historyUp()
	if m.chatTextarea.Value() != "first" {
		t.Errorf("historyUp -> %q, want first", m.chatTextarea.Value())
	}
	m.historyUp() // at oldest: no-op
	if m.chatTextarea.Value() != "first" {
		t.Errorf("historyUp at oldest -> %q", m.chatTextarea.Value())
	}
}

func TestHistoryDownReturnsToFresh(t *testing.T) {
	m := newTestModel()
	m.addToHistory("a")
	m.addToHistory("b")
	m.historyUp() // lands on newest "b"
	if m.chatTextarea.Value() != "b" {
		t.Errorf("historyUp -> %q, want b", m.chatTextarea.Value())
	}
	m.historyDown() // back to fresh
	if m.chatTextarea.Value() != "" {
		t.Errorf("historyDown past newest -> %q, want empty", m.chatTextarea.Value())
	}
	m.historyDown() // at freshest: no-op
	if m.chatTextarea.Value() != "" {
		t.Errorf("historyDown at freshest -> %q", m.chatTextarea.Value())
	}
}

func TestHistoryCapTracksMax(t *testing.T) {
	m := newTestModel()
	for i := 0; i < maxHistory+10; i++ {
		m.addToHistory(strings.Repeat("x", 1))
		m.history[len(m.history)-1] = fmt.Sprintf("msg-%d", i)
	}
	if len(m.history) > maxHistory {
		t.Fatalf("history len=%d exceeds cap %d", len(m.history), maxHistory)
	}
	// Ring rotated: oldest entry evicted.
	if m.history[0] != "msg-10" {
		t.Errorf("ring head = %q, want msg-10", m.history[0])
	}
}

func TestUpDownKeysBrowseHistoryOnlyOnSingleLine(t *testing.T) {
	m := newTestModel()
	m.addToHistory("hello world")
	// Single-line input: ↑ browses history.
	m.chatTextarea.SetValue("typed")
	upd, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m2 := upd.(*Model)
	if cmd != nil {
		t.Error("↑ on single-line input must not return a cmd")
	}
	if m2.chatTextarea.Value() != "hello world" {
		t.Errorf("↑ should fill history entry, got %q", m2.chatTextarea.Value())
	}
	// Multi-line input: ↑ must not clobber the text.
	m.chatTextarea.SetValue("line1\nline2")
	m2.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m2.chatTextarea.Value() != "line1\nline2" {
		t.Errorf("↑ on multi-line must preserve text, got %q", m2.chatTextarea.Value())
	}
}

// ── markdown rendering ──

func TestRenderMarkdownText(t *testing.T) {
	if out := renderMarkdownText("**hi**", 40); !strings.Contains(out, "\x1b[") {
		t.Errorf("bold markdown should be styled with ANSI, got %q", out)
	}
	if out := renderMarkdownText("", 40); out != "" {
		t.Errorf("empty markdown should render empty, got %q", out)
	}
	if out := renderMarkdownText("hello world", 40); !strings.Contains(utils.StripANSI(out), "hello world") {
		t.Errorf("plain text should pass through, got %q", out)
	}
	if out := renderMarkdownText("```go\nfmt.Println(1)\n```", 40); !strings.Contains(utils.StripANSI(out), "fmt.Println") {
		t.Errorf("fenced code should render its content, got %q", out)
	}
}

// visibleCells strips ANSI and returns the terminal cell width of a line,
// counting CJK fullwidth runes as two cells.
func visibleCells(s string) int {
	w := 0
	for _, r := range utils.StripANSI(s) {
		if r > 0x2E7F {
			w += 2
		} else {
			w++
		}
	}
	return w
}

func TestMarkdownRendererWidthNotPinned(t *testing.T) {
	// A renderer's word wrap is fixed at construction, so the per-width
	// cache must wrap a message freshly for its own width instead of
	// reusing the first width ever seen.
	long := strings.Repeat("这是一个很长的助手回复片段用于测试换行。", 6)
	narrow := strings.Count(renderMarkdownText(long, 40), "\n")
	wide := strings.Count(renderMarkdownText(long, 120), "\n")
	if narrow <= wide {
		t.Errorf("narrow wrap (%d newlines) should exceed wide wrap (%d)", narrow, wide)
	}
	for _, w := range []int{40, 120} {
		for _, line := range strings.Split(renderMarkdownText(long, w), "\n") {
			if vc := visibleCells(line); vc > w+2 {
				t.Errorf("markdown line %d cells exceeds wrap width %d: %q", vc, w, line)
			}
		}
	}
}

func TestMessageCardsPerRole(t *testing.T) {
	// opencode style: only the user's prompt is a card (blue rail on the
	// surface, border cell sharing the fill); agent-side roles are cardless
	// indented rows identified by text color — thoughts warning orange,
	// completed tools muted, errors red. Assistant prose carries gruff's
	// own markdown styling, so only its cardlessness is asserted here.
	m := newTestModel()
	m.width = 100
	m.height = 40
	m.inChat = true
	m.messages = []ChatMessage{
		{Role: "user", Content: "hi", TurnId: 1},
		{Role: "assistant", Content: "hello", TurnId: 1},
		{Role: "thought", Content: "thinking", TurnId: 1},
		{Role: "tool", ToolName: "bash", ToolStatus: toolDone, ToolInput: "ls", TurnId: 1},
		{Role: "error", Content: "boom", TurnId: 1},
	}
	vpW := layout.GetViewWidth(100)
	blocks := make([]string, len(m.messages))
	for i, msg := range m.messages {
		block, _ := m.renderMessageBlock(i, msg, vpW)
		blocks[i] = block
		for _, line := range strings.Split(utils.StripANSI(block), "\n") {
			if vc := visibleCells(line); vc > vpW {
				t.Errorf("%s block row %d cells exceeds viewport %d", msg.Role, i, vpW)
			}
		}
	}
	if !strings.Contains(blocks[0], "38;2;0;122;255;48;2;28;28;28m┃") {
		t.Errorf("user rail should be Primary blue on the surface card:\n%s", utils.StripANSI(blocks[0]))
	}
	for _, i := range []int{1, 2, 3, 4} {
		if strings.Contains(blocks[i], "┃") {
			t.Errorf("role %s block should be cardless (no rail)", m.messages[i].Role)
		}
	}
	if !strings.Contains(blocks[2], "38;2;255;159;10") {
		t.Errorf("thought header should be warning orange:\n%s", utils.StripANSI(blocks[2]))
	}
	if !strings.Contains(blocks[3], "38;2;100;98;98") {
		t.Errorf("completed tool row should be muted:\n%s", utils.StripANSI(blocks[3]))
	}
	if !strings.Contains(blocks[4], "38;2;255;59;48") {
		t.Errorf("error row should be red:\n%s", utils.StripANSI(blocks[4]))
	}
}

// TestThoughtAndToolTextOnPage verifies the cardless agent-side rows paint
// the page background: since the opencode-style rework they sit on the page
// (BgNormal), so their colored text must pair with the page fill, not the
// old card surface.
func TestThoughtAndToolTextOnPage(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 40
	m.inChat = true
	vpW := layout.GetViewWidth(100)

	thought := ChatMessage{Role: "thought", Content: "checking the request", TurnId: 1}
	m.visibleConfig.ExpandThinking = true // expanded: header + body both render
	block, _ := m.renderMessageBlock(0, thought, vpW)
	// The header (Warning orange) and the body (muted) must both pair
	// their foreground with the page background.
	if !strings.Contains(block, "38;2;255;159;10;48;2;0;0;0") {
		t.Errorf("thinking header should render on the page bg:\n%s", utils.StripANSI(block))
	}
	if !strings.Contains(block, "38;2;100;98;98;48;2;0;0;0") {
		t.Errorf("thinking body should render muted on the page bg:\n%s", utils.StripANSI(block))
	}

	tool := ChatMessage{Role: "tool", ToolName: "bash", ToolStatus: toolDone, ToolInput: "ls", TurnId: 1}
	block, _ = m.renderMessageBlock(1, tool, vpW)
	if !strings.Contains(block, "38;2;100;98;98;48;2;0;0;0") {
		t.Errorf("tool line should render muted on the page bg:\n%s", utils.StripANSI(block))
	}
}

// TestMessageCardPaddingSymmetric verifies the card's inner padding is
// symmetric: a rail-only pad row frames the text above and below (the old
// top-0/bottom-1 padding made text hug the card's top edge while the
// bottom got a surface strip).
func TestMessageCardPaddingSymmetric(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 40
	m.inChat = true
	vpW := layout.GetViewWidth(100)
	block, _ := m.renderMessageBlock(0, ChatMessage{Role: "user", Content: "hi", TurnId: 1}, vpW)
	rows := strings.Split(utils.StripANSI(block), "\n")
	// The block is pad + text + pad + margin for a one-line body: no top
	// margin, so the gap between adjacent cards is the single bottom-margin
	// row.
	if len(rows) != 4 {
		t.Fatalf("card should be pad + text + pad + margin, got %d rows:\n%q", len(rows), rows)
	}
	railOnly := func(row string) bool { return strings.Trim(row, " ") == "┃" }
	if !railOnly(rows[0]) || !railOnly(rows[2]) {
		t.Errorf("expected rail-only pad rows framing the text, got %q / %q", rows[0], rows[2])
	}
	if !strings.Contains(rows[1], "hi") {
		t.Errorf("middle row should carry the text, got %q", rows[1])
	}
	if strings.TrimSpace(rows[3]) != "" {
		t.Errorf("last row should be the blank bottom margin, got %q", rows[3])
	}
}

func TestLongAssistantRowsFitViewport(t *testing.T) {
	// Long CJK answers must wrap inside the card's content area; the
	// markdown renderer wraps at content width so no row overflows the
	// viewport.
	m := newTestModel()
	m.width = 100
	m.height = 40
	m.inChat = true
	m.messages = []ChatMessage{
		{Role: "assistant", Content: strings.Repeat("这是一个很长的助手回复片段用于测试换行。", 6), TurnId: 1},
	}
	vpW := layout.GetViewWidth(100)
	block, _ := m.renderMessageBlock(0, m.messages[0], vpW)
	for _, line := range strings.Split(utils.StripANSI(block), "\n") {
		if vc := visibleCells(line); vc > vpW {
			t.Errorf("row %d cells exceeds viewport %d: %q", vc, vpW, line)
		}
	}
}

// markdownCellOK walks rendered markdown and checks that no visible cell is
// ever bare: every text run must be preceded by a span that set a background
// (the truecolor surface or a code tint). Glamour resets between spans would
// otherwise leave cells on the terminal default/black.
func markdownCellOK(t *testing.T, out string) {
	t.Helper()
	rest := out
	bgActive := false
	check := func(s string) {
		if s == "" || bgActive {
			return
		}
		for _, line := range strings.Split(s, "\n") {
			if line == "" {
				continue
			}
			t.Errorf("bare cell without background: %q\nfull output:\n%s", line, out)
			return
		}
	}
	for len(rest) > 0 {
		i := strings.Index(rest, "\x1b[")
		if i < 0 {
			check(rest)
			break
		}
		check(rest[:i])
		rest = rest[i:]
		j := strings.Index(rest, "m")
		if j < 0 {
			break
		}
		esc := rest[:j+1]
		rest = rest[j+1:]
		params := esc[2 : len(esc)-1]
		if params == "" || params == "0" {
			bgActive = false
		} else if strings.Contains(params, "48;") {
			bgActive = true
		}
	}
}

func TestMarkdownBackgroundsClean(t *testing.T) {
	// Markdown renders cardless on the page background (opencode style):
	// gruff spans only set the foreground, so the page background is
	// injected into every span to keep resets from dropping cells. Code
	// elements keep gruff's foreground-only style (no ANSI-256 or tinted
	// background), and no surface fill remains.
	out := renderMarkdownText("## 标题\n\n正文带 `行内代码` 和 **粗体**。\n\n```go\nfunc main() {}\n```", 90)
	if strings.Contains(out, "48;5;") {
		t.Errorf("markdown must not use ANSI-256 backgrounds, got 48;5;:\n%s", out)
	}
	if strings.Contains(out, "48;2;28;28;28") {
		t.Errorf("markdown must not sit on the old card surface:\n%s", out)
	}
	if !strings.Contains(out, "48;2;0;0;0") {
		t.Errorf("markdown spans should sit on the page background (0;0;0):\n%s", out)
	}
	markdownCellOK(t, out)
}

func TestMarkdownBackgroundsSurviveView(t *testing.T) {
	// The full view pipeline (card render, viewport fitting, page background
	// pass) must preserve the background discipline: no ANSI-256 backgrounds
	// appear anywhere, and every markdown cell sits on the page background
	// (gruff paints foregrounds only).
	m := newTestModel()
	m.inChat = true
	m.needAutoScroll = false
	md := "## 标题\n\n正文带 `行内代码`。\n\n```go\nfunc main() {}\n```"
	m.messages = []ChatMessage{{Role: "assistant", Content: md, TurnId: 1}}
	// Drive an Update so the viewport is fed (syncViewport runs in Update,
	// not View), then render.
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	v := m.View().Content
	if strings.Contains(v, "48;5;") {
		t.Errorf("full view must not use ANSI-256 backgrounds:\n%s", v)
	} else if !strings.Contains(v, "48;2;28;28;28") {
		t.Errorf("full view should keep the surface background in cards")
	}
}

// ── auto-approve policy ──

// toggleMode drives /toggle_mode through the command registry (same path as
// the panel's Enter) and returns the model.
func toggleMode(t *testing.T, m *Model) *Model {
	t.Helper()
	return toggleSlash(t, m, "/toggle_mode")
}

// toggleThoughtLevel drives /thought_level through the command registry.
func toggleThoughtLevel(t *testing.T, m *Model) *Model {
	t.Helper()
	return toggleSlash(t, m, "/thought_level")
}

func toggleSlash(t *testing.T, m *Model, slash string) *Model {
	t.Helper()
	for _, pc := range m.buildPanelCommands() {
		if pc.slash == slash {
			m2, _ := m.executeCommand(pc)
			return m2.(*Model)
		}
	}
	t.Fatalf("%s missing from registry", slash)
	return nil
}

// TestToggleModeOpensPickerAdaptedToAgent guards the /toggle_mode flow: the
// command opens the session-mode picker whose rows mirror the agent's "mode"
// config option, preselected on the current mode.
func TestToggleModeOpensPickerAdaptedToAgent(t *testing.T) {
	m := newTestModel()
	m.configOptions = sampleConfigOptions() // agent modes: auto/manual/plan
	m2 := toggleMode(t, m)
	if !m2.panelOpen || m2.panelMode != panelModeConfig || m2.configPickerID != "mode" {
		t.Errorf("command must open the mode picker, open=%v mode=%v id=%q",
			m2.panelOpen, m2.panelMode, m2.configPickerID)
	}
	if got := m2.configPickerIndex("mode"); got != 1 { // CurrentValue manual
		t.Errorf("picker must preselect the current mode, idx=%d", got)
	}
	if len(m2.configPickerOptions("mode")) != 3 {
		t.Errorf("picker rows = %d, want 3 (auto/manual/plan)", len(m2.configPickerOptions("mode")))
	}

	// The picker adapts: an agent defining different modes gets those rows.
	m.configOptions = []openacp.SessionConfigOption{{
		ID: "mode", Category: "mode", Type: "select", CurrentValue: "fast",
		Options: []openacp.SessionConfigOptValue{{Value: "fast", Name: "Fast"}},
	}}
	if m3 := toggleMode(t, m); len(m3.configPickerOptions("mode")) != 1 || m3.configPickerOptions("mode")[0].value != "fast" {
		t.Error("mode picker must mirror the agent's own mode option values")
	}
}

// TestModePanelAppliesViaSetConfigOption guards the picker's Enter: the
// picked mode reaches set_config_option ("mode"), the badge updates, and
// the panel closes; picking the current mode is a no-op that still closes.
func TestModePanelAppliesViaSetConfigOption(t *testing.T) {
	modeOpts := func(current string) []openacp.SessionConfigOption {
		opts := sampleConfigOptions()
		for i := range opts {
			if opts[i].ID == "mode" {
				opts[i].CurrentValue = current
			}
		}
		return opts
	}
	m := newTestModel()
	m.configOptions = modeOpts("manual")
	m.activeSessionID = "sess-1"
	m.acpSession = testAcpSession(t)
	m.mode = "manual"

	m2 := toggleMode(t, m)
	m2.panelIdx = 2 // "plan"
	upd, cmd := m2.panelExecute()
	m3 := upd.(*Model)
	if m3.mode != "plan" {
		t.Errorf("mode after pick = %q, want plan", m3.mode)
	}
	if m3.panelOpen {
		t.Error("picking must close the picker")
	}
	if cmd == nil {
		t.Fatal("picking must fire the set_config_option command")
	}

	// The server confirms with fresh config options; re-open preselected on
	// the newly applied mode.
	m3.configOptions = modeOpts("plan")
	m4 := toggleMode(t, m3)
	if m4.configPickerIndex("mode") != 2 {
		t.Errorf("picker must preselect plan, idx=%d", m4.configPickerIndex("mode"))
	}
	upd2, cmd2 := m4.panelExecute() // current mode: no-op
	m5 := upd2.(*Model)
	if m5.mode != "plan" || m5.panelOpen {
		t.Errorf("re-pick must keep mode and close, mode=%q open=%v", m5.mode, m5.panelOpen)
	}
	if cmd2 != nil {
		t.Error("re-picking the current mode must not fire a config command")
	}
}

func TestPermissionArrowsSwitchSelection(t *testing.T) {
	m := newTestModel()
	m.permissionReq = &openacp.RequestPermissionRequest{
		Options: []openacp.PermissionOption{
			{OptionID: "a", Name: "A"},
			{OptionID: "b", Name: "B"},
		},
	}
	m.permissionReplyCh = make(chan openacp.RequestPermissionResponse, 1)
	m.permissionSelectedIdx = 0

	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m.permissionSelectedIdx != 1 {
		t.Errorf("right arrow should select option 1, got %d", m.permissionSelectedIdx)
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.permissionSelectedIdx != 0 {
		t.Errorf("left arrow should select option 0, got %d", m.permissionSelectedIdx)
	}
	if m.permissionReq == nil {
		t.Error("arrow keys must not close the dialog")
	}
}

func TestToggleModeOpensPanel(t *testing.T) {
	m := newTestModel()
	m.configOptions = sampleConfigOptions()
	m = toggleMode(t, m)
	if !m.panelOpen || m.panelMode != panelModeConfig || m.configPickerID != "mode" {
		t.Error("/toggle_mode must open the mode picker, not toggle inline")
	}
}

// TestThoughtLevelOpensPanel guards the thought-level picker: it edits the
// agent's "thought_level" config option through the same generic picker.
func TestThoughtLevelOpensPanel(t *testing.T) {
	m := newTestModel()
	m.configOptions = thoughtConfigOptions()
	m.activeSessionID = "sess-1"
	m.acpSession = testAcpSession(t)
	m2 := toggleThoughtLevel(t, m)
	if !m2.panelOpen || m2.panelMode != panelModeConfig || m2.configPickerID != "thought_level" {
		t.Fatalf("/thought_level must open the thought-level picker")
	}
	if got := m2.configPickerIndex("thought_level"); got != 3 { // CurrentValue high
		t.Errorf("picker must preselect high, idx=%d", got)
	}
	m2.panelIdx = 1 // "low"
	upd, cmd := m2.panelExecute()
	m3 := upd.(*Model)
	if m3.panelOpen {
		t.Error("picking must close the picker")
	}
	if cmd == nil {
		t.Fatal("picking must fire the set_config_option command")
	}
	if m3.currentThoughtLevel() != "high" {
		t.Errorf("thought level = %q, want high until the server confirms", m3.currentThoughtLevel())
	}
}

// ── sessions panel timestamps ──

// TestRelativeTime guards the sessions panel's relative-time labels with an
// injected clock: sub-minute, minutes, hours, days, then a plain date, and
// empty/unparseable input renders as no column at all.
func TestRelativeTime(t *testing.T) {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		rfc  string
		want string
	}{
		{"", ""},
		{"not-a-time", ""},
		{now.Add(-30 * time.Second).Format(time.RFC3339), "just now"},
		{now.Add(-5 * time.Minute).Format(time.RFC3339), "5m ago"},
		{now.Add(-2 * time.Hour).Format(time.RFC3339), "2h ago"},
		{now.Add(-3 * 24 * time.Hour).Format(time.RFC3339), "3d ago"},
		{now.Add(-30 * 24 * time.Hour).Format(time.RFC3339), "2026-08-05"},
	}
	for _, tc := range cases {
		if got := relativeTime(now, tc.rfc); got != tc.want {
			t.Errorf("relativeTime(%q) = %q, want %q", tc.rfc, got, tc.want)
		}
	}
}

// TestSessionPanelShowsUpdatedAt guards the sessions list's time column:
// each row carries the relative update time; rows without a parseable
// timestamp still render their title.
func TestSessionPanelShowsUpdatedAt(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.sessionItems = []sessionItem{
		{id: "s1", title: "Fresh session", updated: time.Now().Add(-5 * time.Minute).Format(time.RFC3339)},
		{id: "s2", title: "No timestamp"},
	}
	out := m.renderSessionPanel()
	if !strings.Contains(out, "5m ago") {
		t.Error("sessions panel must show the relative update time")
	}
	if !strings.Contains(out, "No timestamp") {
		t.Error("rows without a timestamp must still render their title")
	}
}

// ── disabled-command dimming ──

func TestBackendDependentCommandsDisabledWithoutSession(t *testing.T) {
	m := newTestModel()
	for _, pc := range m.buildPanelCommands() {
		switch pc.action {
		case actionSessions, actionModels, actionNew, actionUpdateSkills:
			if pc.enabled {
				t.Errorf("%s should be disabled without a backend", pc.slash)
			}
		case actionToggleMode, actionToggleThinking, actionToggleSkill,
			actionToggleShell, actionToggleToolDetail, actionExit:
			if !pc.enabled {
				t.Errorf("%s should always be enabled", pc.slash)
			}
		}
	}
}

func TestDisabledCommandReportsBackendDown(t *testing.T) {
	m := newTestModel()
	cmds := m.buildPanelCommands()
	var pc panelCommand
	for _, c := range cmds {
		if c.slash == "/sessions" {
			pc = c
			break
		}
	}
	if pc.slash == "" {
		t.Fatal("/sessions missing from registry")
	}
	m2, _ := m.executeCommand(pc)
	if m2.(*Model).panelOpen {
		t.Error("disabled /sessions must not open a panel")
	}
	if m2.(*Model).statusText != "Backend not connected" {
		t.Errorf("statusText = %q, want backend hint", m2.(*Model).statusText)
	}
}

// ── input queue & live panel trigger ──

func TestSlashTriggersPanelLive(t *testing.T) {
	m := newTestModel()
	m.Update(tea.KeyPressMsg{Code: '/'})
	if !m.panelOpen || m.panelMode != panelModeCommand {
		t.Errorf("typing / on an empty input should open the command panel, panelOpen=%v mode=%d", m.panelOpen, m.panelMode)
	}
}

func TestSlashDoesNotTriggerMidInput(t *testing.T) {
	m := newTestModel()
	m.chatTextarea.SetValue("a/b")
	m.chatTextarea.CursorEnd()
	m.Update(tea.KeyPressMsg{Code: '/'})
	if m.panelOpen {
		t.Error("/ mid-input must not open the panel")
	}
	if !strings.Contains(m.chatTextarea.Value(), "/") {
		t.Error("/ mid-input should be typed into the textarea")
	}
}

// TestSlashSheetTypesIntoInput verifies the sheet's filter lives in the
// input box: "/" opens the sheet and lands in the box, further keys keep
// typing there (the sheet never echoes them), and Enter runs the selected
// command and clears the box.
func TestSlashSheetTypesIntoInput(t *testing.T) {
	m := newTestModel()
	m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	if !m.panelOpen || !m.panelFromSlash {
		t.Fatal("typing / on an empty input should open the slash sheet")
	}
	if got := m.chatTextarea.Value(); got != "/" {
		t.Fatalf("input box = %q, want %q (the slash must land in the box)", got, "/")
	}
	m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	m.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	m.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
	m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if got := m.chatTextarea.Value(); got != "/thin" {
		t.Fatalf("input box = %q, want %q (typing must stay in the box)", got, "/thin")
	}
	if m.panelFilter != "thin" {
		t.Errorf("filter = %q, want %q (derived from the box)", m.panelFilter, "thin")
	}
	cmds := m.buildPanelCommands()
	if len(cmds) != 1 || cmds[0].slash != "/toggle_thinking" {
		t.Fatalf("filter /thin should leave /toggle_thinking, got %+v", cmds)
	}

	m2, _ := enterKey(m)
	mm := m2
	if mm.panelOpen {
		t.Error("enter should close the sheet")
	}
	if got := mm.chatTextarea.Value(); got != "" {
		t.Errorf("enter should clear the input, got %q", got)
	}
	if mm.panelFilter != "" {
		t.Errorf("filter = %q, want empty after execution", mm.panelFilter)
	}
	if !mm.visibleConfig.ExpandThinking {
		t.Error("enter should have run /toggle_thinking (expanded)")
	}
}

// TestSlashSheetBackspaceCloses verifies the sheet closes once the box
// stops looking like a command: backspacing the "/" away empties the box
// and closes the sheet.
func TestSlashSheetBackspaceCloses(t *testing.T) {
	m := newTestModel()
	m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if m.panelOpen {
		t.Error("backspacing the slash away should close the sheet")
	}
	if got := m.chatTextarea.Value(); got != "" {
		t.Errorf("input box = %q, want empty", got)
	}
}

// TestSlashSheetEscResetsBox verifies esc closes the sheet and empties the
// box: the sheet only opens on an empty input, so the "/"-filter draft is
// transient — keeping it would prefix the next prompt or double the slash
// on a retype.
func TestSlashSheetEscResetsBox(t *testing.T) {
	m := newTestModel()
	m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.panelOpen {
		t.Error("esc should close the sheet")
	}
	if got := m.chatTextarea.Value(); got != "" {
		t.Errorf("esc should reset the box, got %q", got)
	}
}

// TestSlashSheetEnterNoMatchKeepsSheet verifies enter with no matching
// command is a no-op: the sheet stays open and the draft stays editable
// (nothing was selected, so nothing runs and nothing is cleared).
func TestSlashSheetEnterNoMatchKeepsSheet(t *testing.T) {
	m := newTestModel()
	m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	m2, _ := enterKey(m)
	mm := m2
	if !mm.panelOpen {
		t.Error("enter with no matches should keep the sheet open")
	}
	if got := mm.chatTextarea.Value(); got != "/zz" {
		t.Errorf("input box = %q, want %q (kept for editing)", got, "/zz")
	}
}

// TestSlashSheetNoQueryEcho verifies the sheet never echoes the typed
// filter: no "/query" header row while filtering, and the no-match footer
// does not repeat the query either.
func TestSlashSheetNoQueryEcho(t *testing.T) {
	m := newTestModel()
	m.panelOpen = true
	m.panelFromSlash = true
	m.chatTextarea.SetValue("/mo")
	m.panelFilter = "mo"
	out := utils.StripANSI(m.renderSlashPanel())
	for _, ln := range strings.Split(out, "\n") {
		if ln == "/mo" || strings.HasPrefix(ln, "/mo ") {
			t.Errorf("sheet echoes the query in a header row:\n%s", out)
		}
	}
	if strings.Contains(out, "No matches for") {
		t.Errorf("sheet echoes the query in the no-match footer:\n%s", out)
	}

	// No-match footer stays useful without echoing the query.
	m.chatTextarea.SetValue("/zz")
	m.panelFilter = "zz"
	if out := utils.StripANSI(m.renderSlashPanel()); !strings.Contains(out, "No matching commands") {
		t.Errorf("no-match footer missing:\n%s", out)
	}
}

func TestInputQueueEnqueuesWhileLoading(t *testing.T) {
	m := newTestModel()
	m.loading = true
	m.activeSessionID = "sess-1" // a live session whose prompt is in flight
	m.chatTextarea.SetValue("second")
	m2, _ := enterKey(m)
	if len(m2.inputQueue) != 1 || m2.inputQueue[0] != "second" {
		t.Fatalf("queued = %v, want [second]", m2.inputQueue)
	}
	if !strings.Contains(m2.statusText, "[Queued:1]") {
		t.Errorf("status = %q, want queued hint", m2.statusText)
	}
	if m2.chatTextarea.Value() != "" {
		t.Error("accepted input should be consumed from the box")
	}
	if !m2.loading {
		t.Error("loading must stay true while queued")
	}
	if n := len(m2.messages); n < 1 || m2.messages[n-1].Content != "second" {
		t.Errorf("transcript should show the queued message, last = %+v", m2.messages)
	}
}

func TestInputQueueFullKeepsInput(t *testing.T) {
	m := newTestModel()
	m.loading = true
	m.activeSessionID = "sess-1" // a live session whose prompt is in flight
	m.inputQueue = []string{"a", "b", "c"}
	m.chatTextarea.SetValue("overflow")
	upd, _ := enterKey(m)
	m2 := upd
	if len(m2.inputQueue) != 3 {
		t.Fatalf("queue grew past max: %v", m2.inputQueue)
	}
	if m2.chatTextarea.Value() != "overflow" {
		t.Error("rejected input must stay in the box")
	}
	if !strings.Contains(m2.statusText, "full") {
		t.Errorf("status = %q, want full hint", m2.statusText)
	}
}

func TestInputQueueDrainsOnPromptDone(t *testing.T) {
	m := newTestModel()
	m.loading = false
	m.inputQueue = []string{"a", "b"}

	upd, _ := m.Update(promptDoneMsg{})
	m = upd.(*Model)
	if len(m.inputQueue) != 1 || !m.loading {
		t.Fatalf("after first done: queue=%v loading=%v, want [b] loading=true", m.inputQueue, m.loading)
	}

	upd, _ = m.Update(promptDoneMsg{})
	m = upd.(*Model)
	if len(m.inputQueue) != 0 || !m.loading {
		t.Fatalf("after second done: queue=%v loading=%v, want empty loading=true", m.inputQueue, m.loading)
	}

	upd, _ = m.Update(promptDoneMsg{})
	m = upd.(*Model)
	if m.loading || m.statusText != "" {
		t.Errorf("after drain: loading=%v status=%q, want idle", m.loading, m.statusText)
	}
}

func TestToggleLineNumbersFlipsTextarea(t *testing.T) {
	m := newTestModel()
	if m.chatTextarea.ShowLineNumbers {
		t.Fatal("line numbers should default off")
	}
	m2, _ := m.executeCommand(panelCommand{slash: "/toggle_linenumbers", action: actionToggleLineNumbers})
	mm := m2.(*Model)
	if !mm.chatTextarea.ShowLineNumbers {
		t.Error("/toggle_linenumbers should turn line numbers on")
	}
	if !strings.Contains(mm.statusText, "on") {
		t.Errorf("status = %q, want on-hint", mm.statusText)
	}
}

// ── help panel & notifications ──

func TestHelpCommandOpensPanel(t *testing.T) {
	m := newTestModel()
	m2, _ := m.executeCommand(panelCommand{slash: "/help", action: actionHelp})
	mm := m2.(*Model)
	if !mm.panelOpen || mm.panelMode != panelModeHelp {
		t.Errorf("panelOpen=%v panelMode=%d, want open help panel", mm.panelOpen, mm.panelMode)
	}
}

func TestHelpPanelClosesOnAnyKey(t *testing.T) {
	m := newTestModel()
	m.panelOpen = true
	m.panelMode = panelModeHelp
	upd, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if upd.(*Model).panelOpen {
		t.Error("any key should dismiss the help panel")
	}
}

func TestStaleHelpModeDoesNotSwallowKeys(t *testing.T) {
	// After dismissing help, panelMode stays Help while the panel is closed.
	// A subsequent key press must not be eaten by the help-dismiss branch.
	m := newTestModel()
	m.panelOpen = true
	m.panelMode = panelModeHelp
	upd, _ := m.Update(tea.KeyPressMsg{Code: 'x'})
	m = upd.(*Model)
	if m.panelOpen {
		t.Fatal("help should close")
	}
	m.chatTextarea.SetValue("hello")
	m.activeSessionID = "sess-1" // live session: the key must reach the send path
	m2, _ := enterKey(m)
	if n := len(m2.messages); n == 0 || m2.messages[n-1].Content != "hello" {
		t.Errorf("enter after closing help must send the message, transcript = %+v", m2.messages)
	}
}

func TestNotifyToastAutoClears(t *testing.T) {
	m := newTestModel()
	cmd := m.notify("Session created")
	if m.notifyMsg != "Session created" {
		t.Errorf("notifyMsg = %q, want toast text", m.notifyMsg)
	}
	if cmd == nil {
		t.Fatal("notify must return an auto-clear cmd")
	}
	upd, _ := m.Update(notifyClearMsg{})
	if upd.(*Model).notifyMsg != "" {
		t.Error("notifyClearMsg should clear the toast")
	}
}

func TestConfigSetMsgReturnsNotifyCmd(t *testing.T) {
	m := newTestModel()
	upd, cmd := m.Update(configSetMsg{configOptions: []openacp.SessionConfigOption{
		{ID: "model", Name: "Model", Type: "select", CurrentValue: "gpt-4o"},
	}})
	mm := upd.(*Model)
	if mm.notifyMsg == "" {
		t.Error("config change should raise a toast")
	}
	if cmd == nil {
		t.Error("toast should auto-clear (non-nil cmd)")
	}
}

func TestPlanMsgPopulatesEntries(t *testing.T) {
	m := newTestModel()
	upd, _ := m.Update(planMsg{entries: []openacp.PlanEntry{
		{Content: "inspect", Status: "in_progress"},
		{Content: "wire", Status: "completed"},
		{Content: "verify", Status: "pending"},
	}})
	mm := upd.(*Model)
	if len(mm.planEntries) != 3 || mm.planEntries[0].Status != "in_progress" {
		t.Fatalf("planEntries = %+v", mm.planEntries)
	}
	out := mm.renderPlanList(theme.BaseStyle().Background(theme.BgSecondary), 20)
	if !strings.Contains(out, "Plans") || !strings.Contains(out, "✓") {
		t.Errorf("plan list should render title and checkmarks, got %q", utils.StripANSI(out))
	}
}

// ── render throttling & budgets (9.2/9.3) ──

func TestRenderIntervalSteps(t *testing.T) {
	m := newTestModel()
	m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: strings.Repeat("a", 10*1024)})
	if got := m.renderInterval(); got != 100*time.Millisecond {
		t.Errorf("10KB interval = %v, want 100ms", got)
	}
	m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: strings.Repeat("b", 91*1024)})
	if got := m.renderInterval(); got != 150*time.Millisecond {
		t.Errorf("101KB interval = %v, want 150ms", got)
	}
	m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: strings.Repeat("c", 500*1024)})
	if got := m.renderInterval(); got != 200*time.Millisecond {
		t.Errorf(">500KB interval = %v, want 200ms", got)
	}
}

func TestMarkContentDirtyStreamingThrottles(t *testing.T) {
	m := newTestModel()
	m.loading = true
	m2, cmd := m.markContentDirty()
	if cmd == nil {
		t.Fatal("streaming dirty mark should schedule a flush")
	}
	if m2.(*Model).viewportDirty {
		t.Error("streaming mark must not flush immediately")
	}
	if !m2.(*Model).renderPending || !m2.(*Model).flushPending {
		t.Error("renderPending/flushPending should be set while streaming")
	}
	// A second chunk while the flush timer is in flight collapses into the
	// same flush (no new cmd).
	_, cmd2 := m2.(*Model).markContentDirty()
	if cmd2 != nil {
		t.Error("concurrent chunks must not schedule a second flush")
	}
	// The flush msg rebuilds the viewport once.
	upd, _ := m2.(*Model).Update(flushViewportMsg{})
	m3 := upd.(*Model)
	if !m3.viewportDirty || m3.renderPending || m3.flushPending {
		t.Errorf("after flush: dirty=%v renderPending=%v flushPending=%v, want only dirty", m3.viewportDirty, m3.renderPending, m3.flushPending)
	}
}

func TestMarkContentDirtyIdleFlushesImmediately(t *testing.T) {
	m := newTestModel()
	m.loading = false
	_, cmd := m.markContentDirty()
	if cmd != nil {
		t.Error("idle mark should not schedule a timer")
	}
	if !m.viewportDirty {
		t.Error("idle mark flushes immediately")
	}
}

func TestRenderKeepStartBudget(t *testing.T) {
	cases := []struct {
		sizes  []int
		budget int
		want   int
	}{
		{[]int{1, 2, 3}, 10, 0},  // all fit
		{[]int{5, 5, 5}, 12, 1},  // newest two fit
		{[]int{100, 1, 1}, 5, 1}, // drop the giant oldest
		{[]int{200}, 100, 0},     // newest alone over budget still renders
		{[]int{}, 100, 0},
	}
	for _, c := range cases {
		if got := renderKeepStart(c.sizes, c.budget); got != c.want {
			t.Errorf("renderKeepStart(%v, %d) = %d, want %d", c.sizes, c.budget, got, c.want)
		}
	}
}

func TestToolOutputDefaultFoldsAtFiveLines(t *testing.T) {
	m := newTestModel()
	var b strings.Builder
	for i := 0; i < 11; i++ {
		b.WriteString("line\n")
	}
	b.WriteString("line")
	m.messages = append(m.messages, ChatMessage{
		Role: "tool", TurnId: 0, ToolCallID: "t1",
		ToolName: "view_file", ToolStatus: toolDone, ToolOutput: b.String(),
	})
	rendered := m.renderMessages()
	if !strings.Contains(rendered, "… (7 more lines)") {
		t.Errorf("tool output should fold to %d lines, got:\n%s", defaultToolOutputLines, utils.StripANSI(rendered))
	}
}

// TestMessageBlocksSeparatedByNewline guards the block join: messageCard's
// Render output does not end with a newline, so blocks were joined without
// one — the first card's bottom margin merged with the next card's top pad
// into one line twice the viewport width (background leaks). Each 4-line
// card (top pad, body, bottom pad, bottom margin) must keep exactly one
// newline between blocks.
func TestMessageBlocksSeparatedByNewline(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.messages = []ChatMessage{
		{Role: "user", Content: "first"},
		{Role: "user", Content: "second"},
	}
	out := m.renderMessages()
	lines := strings.Split(out, "\n")
	if len(lines) != 8 {
		t.Fatalf("two 4-line cards should render 8 lines, got %d:\n%s", len(lines), utils.StripANSI(out))
	}
	for i, ln := range lines {
		if w := lipgloss.Width(ln); w > 97 {
			t.Errorf("line %d width %d exceeds viewport width 97 — card margin rows merged", i+1, w)
		}
	}
}

// TestAssistantCardNoLeadingBlank guards the markdown body: glamour emits a
// leading blank row, which would re-add the top padding that messageCard
// removed (a stray empty first row on every assistant card).
func TestAssistantCardNoLeadingBlank(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.messages = []ChatMessage{{Role: "assistant", Content: "# Hi\n\nbody text"}}
	lines := strings.Split(m.renderMessages(), "\n")
	// Cardless assistant: line 0 is the first markdown row (3-column indent
	// then text) — it must carry text. A blank there means glamour's own
	// leading blank leaked through.
	if len(lines) < 1 || strings.TrimSpace(utils.StripANSI(lines[0])) == "" {
		t.Errorf("assistant card content row is blank — leading markdown padding leaked:\n%s",
			utils.StripANSI(strings.Join(lines, "\n")))
	}
}

// missingBG returns the columns of line whose cells carry no explicit
// background at render time (they would be painted page-black by the final
// paint pass). Handles SGR runs and OSC 8 hyperlink sequences.
func missingBG(line string) []int {
	var holes []int
	bg := false
	col := 0
	for i := 0; i < len(line); {
		if line[i] == 0x1b {
			if i+1 < len(line) && line[i+1] == '[' {
				if end := strings.IndexByte(line[i+2:], 'm'); end >= 0 {
					params := line[i+2 : i+2+end]
					switch {
					case params == "" || params == "0":
						bg = false
					case strings.Contains(params, "48;"):
						bg = true
					}
					i += 2 + end + 1
				} else {
					i += 2
				}
				continue
			}
			if i+1 < len(line) && line[i+1] == ']' {
				if j := strings.IndexByte(line[i+2:], 0x07); j >= 0 {
					i += 2 + j + 1
				} else if j := strings.Index(line[i+2:], "\x1b\\"); j >= 0 {
					i += 2 + j + 2
				} else {
					i = len(line)
				}
				continue
			}
			i++
			continue
		}
		if !bg {
			holes = append(holes, col)
		}
		col++
		i++
	}
	return holes
}

// TestMarkdownTableCellsCarryPageBG guards markdownSurfaceBG: alignment
// spaces emitted after a bare reset (e.g. inside tables) must still sit on
// the page background. Every non-margin line of the transcript has to paint
// an explicit background; only the all-blank margin rows may fall through.
func TestMarkdownTableCellsCarryPageBG(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.messages = []ChatMessage{{
		Role:    "assistant",
		Content: "| col A | col B |\n| ----- | ----- |\n| 1     | 2     |\n\nafter table paragraph",
	}}
	out := m.renderMessages()
	if !strings.Contains(utils.StripANSI(out), "col A") {
		t.Fatalf("table did not render:\n%s", utils.StripANSI(out))
	}
	for i, ln := range strings.Split(out, "\n") {
		if strings.TrimSpace(utils.StripANSI(ln)) == "" {
			continue // margin rows are intentionally page-black
		}
		if h := missingBG(ln); len(h) > 0 {
			t.Errorf("line %d has %d no-bg cells (first at %v):\n%s",
				i+1, len(h), h[:min(6, len(h))], utils.StripANSI(ln))
		}
	}
}

// TestPopupLinesPaintFullWidth guards the popup surface: every text line of
// a borderless popup (header, option rows, footer) must paint an explicit
// background out to the full popup width. The footer used MaxWidth, which
// only clips — its tail past the text stayed unpainted and showed the
// terminal's own background through the popup.
func TestPopupLinesPaintFullWidth(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.panelOpen = true
	m.panelMode = panelModeModels
	out := m.renderModelPanel()
	if !strings.Contains(utils.StripANSI(out), "esc") {
		t.Fatalf("models popup did not render:\n%s", utils.StripANSI(out))
	}
	for i, ln := range strings.Split(out, "\n") {
		if strings.TrimSpace(utils.StripANSI(ln)) == "" {
			continue // blank separator rows carry no text
		}
		if h := missingBG(ln); len(h) > 0 {
			t.Errorf("popup line %d has %d unpainted cells (first at %v):\n%s",
				i+1, len(h), h[:min(6, len(h))], utils.StripANSI(ln))
		}
	}
}

// ── transcript search (/search) ──

func TestSearchPanelOpensAndFilters(t *testing.T) {
	m := newTestModel()
	m.messages = []ChatMessage{
		{Role: "assistant", Content: "hello world"},
		{Role: "user", Content: "foobar only"},
		{Role: "assistant", Content: "world again"},
	}
	upd, _ := m.executeCommand(panelCommand{slash: "/search", action: actionSearch})
	m = upd.(*Model)
	if !m.panelOpen || m.panelMode != panelModeSearch {
		t.Fatalf("panelOpen=%v mode=%d, want search panel", m.panelOpen, m.panelMode)
	}
	for _, r := range "world" {
		m.Update(tea.KeyPressMsg{Code: r})
	}
	if m.panelFilter != "world" {
		t.Errorf("panelFilter = %q", m.panelFilter)
	}
	if len(m.searchResults) != 2 || m.searchResults[0] != 0 || m.searchResults[1] != 2 {
		t.Errorf("searchResults = %v, want [0 2]", m.searchResults)
	}
}

func TestSearchNoMatches(t *testing.T) {
	m := newTestModel()
	m.messages = []ChatMessage{{Role: "user", Content: "hello"}}
	upd, _ := m.executeCommand(panelCommand{slash: "/search", action: actionSearch})
	m = upd.(*Model)
	for _, r := range "zzz" {
		m.Update(tea.KeyPressMsg{Code: r})
	}
	if len(m.searchResults) != 0 {
		t.Errorf("searchResults = %v, want empty", m.searchResults)
	}
}

func TestSearchEnterJumpsToMatch(t *testing.T) {
	m := newTestModel()
	m.messages = []ChatMessage{
		{Role: "assistant", Content: "first world\nwith two lines"},
		{Role: "user", Content: "irrelevant"},
		{Role: "assistant", Content: "second world"},
	}
	upd, _ := m.executeCommand(panelCommand{slash: "/search", action: actionSearch})
	m = upd.(*Model)
	for _, r := range "world" {
		m.Update(tea.KeyPressMsg{Code: r})
	}
	// A short viewport forces a real scroll offset (content exceeds it).
	m.chatViewport.SetHeight(4)
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // select result 1 (message 2)
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := m2.(*Model)
	if mm.panelOpen {
		t.Error("enter should close the search panel")
	}
	if got := mm.chatViewport.YOffset(); got <= 0 {
		t.Errorf("viewport should scroll to the match, offset = %d", got)
	}
	if v := mm.chatViewport.View(); !strings.Contains(utils.StripANSI(v), "second world") {
		t.Error("viewport content should contain the matched message")
	}
}

func TestSearchEscCloses(t *testing.T) {
	m := newTestModel()
	upd, _ := m.executeCommand(panelCommand{slash: "/search", action: actionSearch})
	m = upd.(*Model)
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m2.(*Model).panelOpen {
		t.Error("esc should close the search panel")
	}
}

// ── transcript export (/export) ──

func TestExportTranscriptWritesMarkdown(t *testing.T) {
	m := newTestModel()
	m.workDir = t.TempDir()
	m.activeSessionID = "sess-123"
	m.messages = []ChatMessage{
		{Role: "user", Content: "hi there"},
		{Role: "assistant", Content: "**bold** reply"},
		{Role: "tool", TurnId: 0, ToolCallID: "c1", ToolName: "bash", ToolStatus: toolDone, ToolInput: `{"cmd":"ls"}`, ToolOutput: "a.txt"},
	}
	upd, _ := m.executeCommand(panelCommand{slash: "/export", action: actionExport})
	m2 := upd.(*Model)
	if !m2.panelOpen || m2.panelMode != panelModeExport {
		t.Fatal("export should open the dismiss-only export dialog")
	}
	if !strings.Contains(m2.exportNotice, "Transcript exported to:") || !strings.Contains(m2.exportNotice, ".md") {
		t.Errorf("dialog should name the written file, got %q", m2.exportNotice)
	}
	entries, err := os.ReadDir(m.workDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one exported file, got %v (err %v)", entries, err)
	}
	b, _ := os.ReadFile(filepath.Join(m.workDir, entries[0].Name()))
	content := string(b)
	for _, want := range []string{"# Chat transcript", "sess-123", "hi there", "**bold** reply", "bash", "```json"} {
		if !strings.Contains(content, want) {
			t.Errorf("exported markdown missing %q:\n%s", want, content)
		}
	}
}

// TestExportPanelDismissesOnAnyKey guards the export dialog's dismiss-only
// interaction: any key closes it (matching the help panel).
func TestExportPanelDismissesOnAnyKey(t *testing.T) {
	m := newTestModel()
	m.panelOpen = true
	m.panelMode = panelModeExport
	upd, _ := m.Update(tea.KeyPressMsg{Code: 'x'})
	if upd.(*Model).panelOpen {
		t.Error("any key should close the export dialog")
	}
}

func TestExportEmptyTranscriptGuard(t *testing.T) {
	m := newTestModel()
	m.workDir = t.TempDir()
	upd, _ := m.executeCommand(panelCommand{slash: "/export", action: actionExport})
	m2 := upd.(*Model)
	if !m2.panelOpen || m2.panelMode != panelModeExport {
		t.Fatal("empty export should still surface the dialog")
	}
	if !strings.Contains(m2.exportNotice, "Nothing to export") {
		t.Errorf("exportNotice = %q, want the empty-transcript note", m2.exportNotice)
	}
	entries, err := os.ReadDir(m.workDir)
	if err != nil || len(entries) != 0 {
		t.Errorf("empty export must not write files, got %v (err %v)", entries, err)
	}
}

// ── incremental render cache ──

func TestRenderCacheReusesStableBlocks(t *testing.T) {
	m := newTestModel()
	m.messages = []ChatMessage{
		{Role: "user", Content: "hello world"},
		{Role: "assistant", Content: "reply one"},
	}
	m.renderMessages()
	if len(m.renderCache) != 2 {
		t.Fatalf("cache entries = %d, want 2", len(m.renderCache))
	}
	first := m.renderCache[0].block
	if first == "" {
		t.Fatal("user block should be cached")
	}
	// Streaming appends to the assistant message: only that entry restyles;
	// the untouched user message keeps its cached block.
	m.messages[1].Content = "reply one plus more"
	m.renderMessages()
	if m.renderCache[0].block != first {
		t.Error("unchanged message must reuse its cached block")
	}
	if m.renderCache[1].block == "" || m.renderCache[1].block == m.renderCache[1].content {
		t.Error("changed message should be restyled")
	}
	if len(m.renderCache) != 2 {
		t.Errorf("cache entries = %d, want 2 (no growth)", len(m.renderCache))
	}
}

func TestRenderCacheVisibilityChangeInvalidates(t *testing.T) {
	m := newTestModel()
	m.messages = []ChatMessage{
		{Role: "thought", Content: "secret"},
		{Role: "assistant", Content: "hi"},
	}
	m.renderMessages()
	before := m.renderCache[0].block
	if before == "" {
		t.Fatal("thought block should render by default")
	}
	m.visibleConfig.ExpandThinking = true
	out := m.renderMessages()
	if !strings.Contains(utils.StripANSI(out), "secret") {
		t.Error("expanded thought must render its content")
	}
	if m.renderCache[0].skip || m.renderCache[0].block == before {
		t.Error("thought entry should restyle when expansion flips")
	}
}

func TestRenderCacheBoundedByMessageCount(t *testing.T) {
	m := newTestModel()
	m.messages = []ChatMessage{{Role: "user", Content: "a"}, {Role: "user", Content: "b"}, {Role: "user", Content: "c"}}
	m.renderMessages()
	if len(m.renderCache) != 3 {
		t.Fatalf("cache entries = %d, want 3", len(m.renderCache))
	}
	// Simulate the render window advancing past the oldest message.
	m.messages = m.messages[1:]
	m.renderMessages()
	if len(m.renderCache) > len(m.messages) {
		t.Errorf("cache (%d) must not exceed message count (%d)", len(m.renderCache), len(m.messages))
	}
}

// ── edit picker (/edit) ──

func TestEditPanelListsUserMessagesNewestFirst(t *testing.T) {
	m := newTestModel()
	m.messages = []ChatMessage{
		{Role: "user", Content: "old ask"},
		{Role: "assistant", Content: "reply"},
		{Role: "user", Content: "new ask"},
	}
	upd, _ := m.executeCommand(panelCommand{slash: "/edit", action: actionEdit})
	m = upd.(*Model)
	if !m.panelOpen || m.panelMode != panelModeEdit {
		t.Fatalf("panelOpen=%v mode=%d, want edit picker", m.panelOpen, m.panelMode)
	}
	idxs := m.editableMessages()
	if len(idxs) != 2 || idxs[0] != 2 || idxs[1] != 0 {
		t.Errorf("editableMessages = %v, want [2 0] (newest first)", idxs)
	}
}

func TestEditSelectionCopiesToInput(t *testing.T) {
	m := newTestModel()
	m.messages = []ChatMessage{
		{Role: "user", Content: "first ask\nwith a second line"},
		{Role: "assistant", Content: "reply"},
	}
	upd, _ := m.executeCommand(panelCommand{slash: "/edit", action: actionEdit})
	m = upd.(*Model)
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown}) // select the older message
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := m2.(*Model)
	if mm.panelOpen {
		t.Error("enter should close the edit picker")
	}
	if got := mm.chatTextarea.Value(); got != "first ask\nwith a second line" {
		t.Errorf("textarea = %q, want copied message", got)
	}
}

func TestEditEmptyTranscriptGuard(t *testing.T) {
	m := newTestModel()
	upd, _ := m.executeCommand(panelCommand{slash: "/edit", action: actionEdit})
	m = upd.(*Model)
	if len(m.editableMessages()) != 0 {
		t.Fatal("no user messages expected")
	}
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m2.(*Model).panelOpen {
		t.Error("enter with nothing to edit should close the picker")
	}
}

// ── theme cycling (/theme) ──

// themeHex renders a palette color back to its "#rrggbb" form for tests.
func themeHex(c color.Color) string {
	col, ok := colorful.MakeColor(c)
	if !ok {
		return ""
	}
	return col.Hex()
}

func TestThemeCyclesAndAppliesPresets(t *testing.T) {
	t.Cleanup(theme.Reset)
	m := newTestModel()
	var run = func() *Model {
		m2, cmd := m.executeCommand(panelCommand{slash: "/theme", action: actionTheme})
		if cmd == nil {
			t.Fatal("/theme should raise a toast")
		}
		return m2.(*Model)
	}
	m = run() // light
	if m.themeIdx != 1 {
		t.Fatalf("themeIdx = %d, want 1 (light)", m.themeIdx)
	}
	if got := themeHex(theme.BgNormal); got != "#f2f2f7" {
		t.Errorf("light bg = %v, want #f2f2f7", got)
	}
	m = run() // high-contrast
	if m.themeIdx != 2 {
		t.Fatalf("themeIdx = %d, want 2", m.themeIdx)
	}
	m = run() // wraps to default (built-in palette)
	if m.themeIdx != 0 {
		t.Fatalf("themeIdx = %d, want 0 after wrap", m.themeIdx)
	}
	if got := themeHex(theme.BgNormal); got != "#000000" {
		t.Errorf("default bg = %v, want built-in #000000", got)
	}
	if got := themeHex(theme.BgPanel); got != "#141414" {
		t.Errorf("default panel bg = %v, want built-in #141414", got)
	}
}

func TestThemeToggleIconTracksPreset(t *testing.T) {
	t.Cleanup(theme.Reset)
	m := newTestModel()
	if m.toggleIcon("/theme") != "d" {
		t.Error("icon should show preset initial 'd'")
	}
	m.themeIdx = 1
	if m.toggleIcon("/theme") != "l" {
		t.Error("icon should show 'l' for light")
	}
}

// ── plugins panel (/plugins) ──

func TestPluginsPanelListsInstalled(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "skill-a"), 0o755)
	os.MkdirAll(filepath.Join(dir, "skill-b"), 0o755)
	os.WriteFile(filepath.Join(dir, "plugin.json"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644)
	setPluginsDir(dir)
	t.Cleanup(func() { setPluginsDir("") })

	m := newTestModel()
	upd, _ := m.executeCommand(panelCommand{slash: "/plugins", action: actionPlugins})
	m = upd.(*Model)
	if !m.panelOpen || m.panelMode != panelModePlugins {
		t.Fatalf("panelOpen=%v mode=%d, want plugins panel", m.panelOpen, m.panelMode)
	}
	want := []string{"plugin.json", "skill-a", "skill-b"} // sorted by name
	if len(m.pluginItems) != len(want) {
		t.Fatalf("pluginItems = %v, want %v", m.pluginItems, want)
	}
	for i := range want {
		if m.pluginItems[i] != want[i] {
			t.Errorf("pluginItems[%d] = %q, want %q", i, m.pluginItems[i], want[i])
		}
	}
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m2.(*Model).panelOpen {
		t.Error("esc should close the plugins panel")
	}
}

// ── split view (/split) ──

func TestSplitTogglesTwoPaneLayout(t *testing.T) {
	m := newTestModel()
	m.messages = []ChatMessage{{Role: "user", Content: "hi"}}
	m.planEntries = []openacp.PlanEntry{{Content: "step", Status: "pending"}}

	upd, _ := m.executeCommand(panelCommand{slash: "/split", action: actionSplit})
	m = upd.(*Model)
	if !m.splitView {
		t.Fatal("splitView should turn on")
	}
	if !strings.Contains(utils.StripANSI(m.renderLeft(&viewGeom{})), "Context") {
		t.Error("split view should render the context pane")
	}
	m2, _ := m.executeCommand(panelCommand{slash: "/split", action: actionSplit})
	if m2.(*Model).splitView {
		t.Error("second /split should turn split off")
	}
}

// ── lazy history store (17.2) ──

func TestTrimMessageStoreDropsOldestBeyondBudget(t *testing.T) {
	m := newTestModel()
	big := strings.Repeat("a", 2_100_000) // over maxStoredChars (2M)
	m.messages = []ChatMessage{{Role: "assistant", Content: big}, {Role: "user", Content: "keep me"}}
	m.trimMessageStore()
	if len(m.messages) != 1 || m.messages[0].Content != "keep me" {
		t.Fatalf("messages = %d rows, want newest only", len(m.messages))
	}
}

func TestTrimMessageStoreKeepsNewestWhenAloneOverBudget(t *testing.T) {
	m := newTestModel()
	m.messages = []ChatMessage{{Role: "assistant", Content: strings.Repeat("z", 3_000_000)}}
	m.trimMessageStore()
	if len(m.messages) != 1 {
		t.Errorf("single huge message must never be dropped, got %d rows", len(m.messages))
	}
}

// ── line-level virtual scrolling (17.2) ──

func TestVirtualLineHeightsMatchRenderedBlocks(t *testing.T) {
	m := newTestModel()
	m.messages = []ChatMessage{
		{Role: "user", Content: "hi"},
		{Role: "user", Content: strings.Repeat("x", 66)}, // wraps at the card's inner width
		{Role: "thought"},                                // empty thought, idle: hidden in both modes
	}

	vpW := layout.GetViewWidth(m.width)
	heights := m.virtualLineHeights(vpW)

	// Heights must equal the real rendered block heights: the virtual
	// window cuts (or pads) a message's rows whenever they drift apart,
	// which is how card bottom padding used to vanish.
	for i, msg := range m.messages {
		block, skip := m.renderMessageBlock(i, msg, vpW)
		if skip {
			if heights[i] != 0 {
				t.Errorf("hidden message %d height = %d, want 0", i, heights[i])
			}
			continue
		}
		if want := strings.Count(block, "\n") + 1; heights[i] != want {
			t.Errorf("message %d height = %d, want rendered %d", i, heights[i], want)
		}
	}
	if heights[0] != 4 {
		t.Errorf("short user msg height = %d, want 4 (pad, text, pad, margin)", heights[0])
	}
}

func TestVirtualDocStylesOnlyVisibleWindow(t *testing.T) {
	m := newTestModel()
	for i := 0; i < 6; i++ {
		m.messages = append(m.messages, ChatMessage{Role: "user", Content: "m" + strconv.Itoa(i)})
	}
	m.chatViewport.SetHeight(4) // window rows [0,4)
	doc := m.renderVirtualDoc(4)

	// Measuring the exact heights styles every message once (through the
	// render cache); the windowing now lives in the ROWS: inside the window
	// the doc carries the real block rows, everywhere else placeholders.
	if len(m.renderCache) != 6 {
		t.Fatalf("cache entries = %d, want 6 (height measurement styles all)", len(m.renderCache))
	}
	lines := strings.Split(doc, "\n")
	heights := m.virtualLineHeights(layout.GetViewWidth(m.width))
	total := 0
	for _, h := range heights {
		total += h
	}
	if len(lines) != total {
		t.Fatalf("doc rows = %d, want %d (sum of exact heights)", len(lines), total)
	}
	// Message 0's height is 4 ([pad, text, pad, margin]) and the window is
	// 4 rows: the whole block renders inside the window — rows 0-2 carry
	// the rail (row 0 is the top pad), row 3 is the blank bottom margin —
	// and everything below is placeholder filler.
	for i := 0; i < 3; i++ {
		if !strings.Contains(lines[i], "┃") {
			t.Errorf("in-window row %d should be a real card row:\n%q", i, lines[i])
		}
	}
	if strings.TrimSpace(utils.StripANSI(lines[3])) != "" {
		t.Errorf("row 3 should be the blank bottom margin:\n%q", lines[3])
	}
	for i := 4; i < len(lines); i++ {
		if !strings.Contains(lines[i], "┆") {
			t.Errorf("out-of-window row %d should be a placeholder:\n%q", i, lines[i])
		}
	}
}

func TestVirtualScrollSupplementsMissingRows(t *testing.T) {
	m := newTestModel()
	for i := 0; i < 6; i++ {
		m.messages = append(m.messages, ChatMessage{Role: "user", Content: "m" + strconv.Itoa(i)})
	}
	m.chatViewport.SetHeight(4)

	m.renderVirtualDoc(4) // height measurement styles every message once
	first := m.renderCache[0].block
	if first == "" {
		t.Fatal("message 0 should be styled at the top")
	}

	// Feed the viewport (as renderLeft does) so ScrollDown has a non-empty
	// document to scroll within; the virtual feed then refeeds at the new
	// offset. The measurement pass styles everything up front, so the
	// property left to guard is that the exited message reuses its cached
	// block instead of restyling, and the refeed at the new offset keeps
	// the document stable. Auto-scroll is off so the window stays anchored.
	m.needAutoScroll = false
	m.feedViewport(4)
	m.chatViewport.ScrollDown(6)
	if m.chatViewport.YOffset() != 6 {
		t.Fatalf("offset = %d, want 6", m.chatViewport.YOffset())
	}
	m.feedViewport(4)
	if m.renderCache[0].block != first {
		t.Error("exited message must reuse its cached block, not restyle")
	}
	// The refeed at offset 6 must show real rows for the newly revealed
	// message 1 (4-row blocks put rows [4,8) inside it) rather than
	// placeholders.
	doc := m.renderVirtualDoc(4)
	lines := strings.Split(doc, "\n")
	if len(lines) < 7 || !strings.Contains(lines[6], "┃") {
		t.Errorf("row 6 after refeed should be a real card row:\n%q", lines[min(6, len(lines)-1)])
	}
}

func TestVirtualDocUniformRowWidthAndTotal(t *testing.T) {
	m := newTestModel()
	for i := 0; i < 4; i++ {
		m.messages = append(m.messages, ChatMessage{Role: "user", Content: "message " + strconv.Itoa(i)})
	}
	vpW := layout.GetViewWidth(m.width)
	m.chatViewport.SetHeight(5)
	doc := m.renderVirtualDoc(5)
	for i, l := range strings.Split(doc, "\n") {
		if w := utils.DisplayWidth(l); w != vpW {
			t.Errorf("doc row %d width = %d, want %d (uniform, no soft-wrap)", i, w, vpW)
		}
	}
	m.chatViewport.SetContent(doc)
	want := 0
	for _, h := range m.virtualLineHeights(vpW) {
		want += h
	}
	if got := m.chatViewport.TotalLineCount(); got != want {
		t.Errorf("viewport total = %d, want %d (exact rendered rows)", got, want)
	}
}

func TestMessageAtLineMapping(t *testing.T) {
	h := []int{3, 4, 2}
	cases := []struct {
		line, idx, within int
	}{
		{0, 0, 0}, {2, 0, 2}, {3, 1, 0}, {6, 1, 3}, {7, 2, 0}, {8, 2, 1}, {99, 2, 1},
	}
	for _, c := range cases {
		idx, within := messageAtLine(h, c.line)
		if idx != c.idx || within != c.within {
			t.Errorf("messageAtLine(%v, %d) = (%d,%d), want (%d,%d)", h, c.line, idx, within, c.idx, c.within)
		}
	}
	if idx, _ := messageAtLine(nil, 5); idx != -1 {
		t.Errorf("empty heights should map to -1, got %d", idx)
	}
}

// ── input header badges: model + thinking strength ──

func thoughtConfigOptions() []openacp.SessionConfigOption {
	return append(sampleConfigOptions(), openacp.SessionConfigOption{
		ID: "thought_level", Category: "thought_level", Type: "select",
		CurrentValue: "high",
		Options: []openacp.SessionConfigOptValue{
			{Value: "off"}, {Value: "low"}, {Value: "medium"}, {Value: "high"},
		},
	})
}

func TestCurrentThoughtLevelReadsConfig(t *testing.T) {
	m := newTestModel()
	if got := m.currentThoughtLevel(); got != "" {
		t.Errorf("absent option should be empty, got %q", got)
	}
	m.configOptions = thoughtConfigOptions()
	if got := m.currentThoughtLevel(); got != "high" {
		t.Errorf("currentThoughtLevel = %q, want high", got)
	}
}

func TestInputHeaderShowsModeModelAndThinking(t *testing.T) {
	m := newTestModel() // boots in the agent's "manual" mode
	m.configOptions = thoughtConfigOptions()
	out := utils.StripANSI(m.renderInput())
	for _, want := range []string{"Manual", "deepseek-v4", "high"} {
		if !strings.Contains(out, want) {
			t.Errorf("input header missing %q, got:\n%s", want, out)
		}
	}
	// Badges are joined with a spaced dot (" · ") in the
	// mode·model·thinking order and the thinking strength has no leading
	// icon. No policy badge: approval behavior follows the session mode.
	if strings.Contains(out, "💭") {
		t.Error("input header should not show the thinking icon")
	}
	if !strings.Contains(out, "Manual · deepseek-v4 · high") {
		t.Errorf("input header not formatted as 'mode · model · thinking', got:\n%s", out)
	}

	// The mode badge adapts to the agent's mode values.
	m.mode = "auto"
	if out2 := utils.StripANSI(m.renderInput()); !strings.Contains(out2, "Auto · deepseek-v4 · high") {
		t.Errorf("mode badge should render Auto, got:\n%s", out2)
	}
	m.mode = "plan"
	if out3 := utils.StripANSI(m.renderInput()); !strings.Contains(out3, "Plan · deepseek-v4 · high") {
		t.Errorf("mode badge should render Plan, got:\n%s", out3)
	}
}

func TestThinkingBadgeHiddenWithoutOption(t *testing.T) {
	m := newTestModel()
	if got := m.renderThinkingBadge(); got != "" {
		t.Errorf("thinking badge should be empty without the option, got %q", got)
	}
	if got := m.renderModelBadge(); got != "" {
		t.Errorf("model badge should be empty without config, got %q", got)
	}
}

func TestConfigOptionsMsgPopulatesOptions(t *testing.T) {
	m := newTestModel()
	opts := thoughtConfigOptions()
	upd, _ := m.Update(configOptionsMsg{opts: opts})
	mm := upd.(*Model)
	if mm.currentModel() != "deepseek-v4" || mm.currentThoughtLevel() != "high" {
		t.Errorf("configOptionsMsg should refresh model/thought, model=%q thought=%q", mm.currentModel(), mm.currentThoughtLevel())
	}
}

func TestAcpSessionReadyMsgStoresOptions(t *testing.T) {
	m := newTestModel()
	opts := thoughtConfigOptions()
	upd, _ := m.Update(AcpSessionReadyMsg("sess-boot", opts))
	mm := upd.(*Model)
	if mm.activeSessionID != "sess-boot" {
		t.Errorf("sessionID = %q", mm.activeSessionID)
	}
	if mm.currentModel() != "deepseek-v4" || mm.currentThoughtLevel() != "high" {
		t.Errorf("boot options not stored: model=%q thought=%q", mm.currentModel(), mm.currentThoughtLevel())
	}
}

func TestWelcomeInputNarrowerThanColumn(t *testing.T) {
	m := newTestModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
	// Empty value → the placeholder path, which is what the welcome page
	// shows; textarea content would carry its own (chat-sized) width.
	m.chatTextarea.SetValue("")

	// The welcome box renders at welcomeInputWidth (54 at 100 cols).
	// Measure with terminal-accurate widths: utils.DisplayWidth runs on
	// runewidth with EastAsianWidth=true, which over-counts the box-drawing
	// glyphs (┃/▀ render at 1 column on a real terminal, 2 under that ruler).
	col := m.getContentWidth() // welcome column width (72 at 100 cols)
	want := welcomeInputWidth(col)
	box := utils.StripANSI(m.renderInputAt(want))
	maxW, minW := 0, int(^uint(0)>>1)
	for _, l := range strings.Split(box, "\n") {
		w := terminalWidth(l)
		maxW = max(maxW, w)
		minW = min(minW, w)
	}
	if maxW != want {
		t.Errorf("input box width = %d, want %d", maxW, want)
	}
	// Every box row must be the same width so the frame stays rectangular.
	if maxW != minW {
		t.Errorf("input box rows not uniform: min=%d max=%d", minW, maxW)
	}
	if want > col {
		t.Errorf("input width %d should not exceed the welcome column %d", want, col)
	}
}

// terminalWidth measures the display width the way a real terminal renders
// it: CJK runes count as 2 columns, box-drawing/block glyphs (U+2500–U+259F)
// as 1. utils.DisplayWidth uses runewidth with EastAsianWidth=true, which
// over-counts those glyphs.
func terminalWidth(s string) int {
	w := 0
	for _, r := range utils.StripANSI(s) {
		if r >= 0x2500 && r <= 0x259F {
			w++
			continue
		}
		w += runewidth.RuneWidth(r)
	}
	return w
}

// TestWelcomeInputCenteredLive verifies the welcome input box under the
// boot-session condition (activeSessionID set): the welcome layout must
// still use the welcome column and center the box under the logo. Prior to
// the inChat keying, an existing boot session widened the whole welcome
// page to chat width and pinned the box to the left edge.
func TestWelcomeInputCenteredLive(t *testing.T) {
	m := newTestModel()
	m.activeSessionID = "sess-live"
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})

	out := utils.StripANSI(m.renderWelcome(&viewGeom{}))
	var boxCol, boxWidth int
	for _, l := range strings.Split(out, "\n") {
		leftTrimmed := strings.TrimLeft(l, " ")
		col := terminalWidth(l) - terminalWidth(leftTrimmed)
		switch {
		case strings.HasPrefix(leftTrimmed, "╹"):
			// The footer keeps its block glyphs, so Trim does not eat it.
			boxWidth = terminalWidth(strings.Trim(l, " "))
		case strings.HasPrefix(leftTrimmed, "┃"):
			// Blank rows are all interior spaces; check only the left edge.
			if boxCol == 0 {
				boxCol = col
			} else if col != boxCol {
				t.Errorf("input box rows not aligned: col=%d want %d", col, boxCol)
			}
		}
	}
	if boxWidth == 0 {
		t.Fatal("no input box found in welcome render")
	}
	want := welcomeInputWidth(layout.GetWelcomeWidth(m.width))
	if boxWidth != want {
		t.Errorf("input box width = %d, want %d", boxWidth, want)
	}
	// Centered in the window: left margin equals right margin.
	right := m.width - boxCol - boxWidth
	if boxCol != right {
		t.Errorf("input box not centered: left margin %d, right margin %d", boxCol, right)
	}
}

// TestWelcomeDropsBelowCenter verifies the welcome block hangs below exact
// vertical center: the input box top (geom.inputTopY, the row the slash
// sheet docks to) must sit welcomeDrop rows lower than strict centering,
// and the block must still fit the height-2 working area on short
// terminals (the drop is capped by the available slack there).
func TestWelcomeDropsBelowCenter(t *testing.T) {
	m := newTestModel()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})

	var geom viewGeom
	m.renderWelcome(&geom)

	contentH := lipgloss.Height(m.welcomeInner())
	logoH := lipgloss.Height(m.welcomeLogo(m.getContentWidth()))
	centered := (36-2-contentH)/2 + logoH
	if geom.inputTopY != centered+welcomeDrop {
		t.Errorf("welcome input top = %d, want centered %d + drop %d = %d",
			geom.inputTopY, centered, welcomeDrop, centered+welcomeDrop)
	}

	m2 := newTestModel()
	m2.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	var geom2 viewGeom
	m2.renderWelcome(&geom2)
	blockTop := geom2.inputTopY - lipgloss.Height(m2.welcomeLogo(m2.getContentWidth()))
	if blockTop+lipgloss.Height(m2.welcomeInner()) > 20-2 {
		t.Errorf("welcome block overflows the working area: top=%d height=%d",
			blockTop, lipgloss.Height(m2.welcomeInner()))
	}
}

// TestInputBoxStaysUniformWithLongContent guards the inner-width clamp:
// the shared textarea is sized for the chat column, so on the welcome page
// a long draft must wrap at the box's inner width instead of spilling past
// the frame.
func TestInputBoxStaysUniformWithLongContent(t *testing.T) {
	m := newTestModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
	long := strings.Repeat("abcde ", 20) // wraps across several lines
	m.chatTextarea.SetValue(long)

	want := welcomeInputWidth(m.getContentWidth())
	out := utils.StripANSI(m.renderInputAt(want))
	widths := map[int]int{}
	for _, l := range strings.Split(out, "\n") {
		widths[terminalWidth(l)]++
	}
	if len(widths) != 1 {
		t.Errorf("input box rows not uniform after long content: %v", widths)
	}
	for w := range widths {
		if w != want {
			t.Errorf("input box width = %d, want %d", w, want)
		}
	}
}

// TestChatTextareaFitsChatBox checks that on the chat page the textarea is
// sized to the box's inner text width (frame minus border and padding), so
// typed content cannot overflow the chat input box either.
func TestChatTextareaFitsChatBox(t *testing.T) {
	m := newTestModel()
	m.inChat = true
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})

	if got, want := m.inputTextWidth(), m.getContentWidth()-1-2; got != want {
		t.Errorf("inputTextWidth = %d, want %d", got, want)
	}
	long := strings.Repeat("abcde ", 20)
	m.chatTextarea.SetValue(long)
	out := utils.StripANSI(m.renderInput())
	widths := map[int]int{}
	for _, l := range strings.Split(out, "\n") {
		widths[terminalWidth(l)]++
	}
	if len(widths) != 1 {
		t.Errorf("chat input box rows not uniform: %v", widths)
	}
}

// TestNoUnpaintedBackgroundCells walks every rendered frame (welcome page,
// chat page, floating panels) with an ANSI state machine and asserts that no
// cell is left without a background. Unpainted cells would show the
// terminal's default background through the UI — lipgloss alignment padding
// is the usual culprit.
func TestNoUnpaintedBackgroundCells(t *testing.T) {
	m := newTestModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
	// user message so the chat page has real content
	m.messages = append(m.messages, ChatMessage{Role: "user", Content: "hello"})
	m.needAutoScroll = false
	scenarios := map[string]func(){
		"welcome": func() {},
		"chat":    func() { m.inChat = true },
		"panel":   func() { m.panelOpen = true; m.panelMode = panelModeCommand },
	}
	for name, setup := range scenarios {
		mm := newTestModel()
		mm.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
		if name == "chat" {
			mm.messages = append(mm.messages, ChatMessage{Role: "user", Content: "hello"})
			mm.needAutoScroll = false
			mm.inChat = true
		} else if name == "panel" {
			mm.panelOpen = true
			mm.panelMode = panelModeCommand
		}
		setup()
		_ = m
		total := 0
		for rowIdx, l := range strings.Split(mm.View().Content, "\n") {
			re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
			loc := re.FindAllStringIndex(l, -1)
			parts := re.Split(l, -1)
			bgSet := false
			for j, p := range parts {
				if j > 0 {
					code := l[loc[j-1][0]:loc[j-1][1]]
					switch {
					case code == "\x1b[0m" || code == "\x1b[m":
						bgSet = false
					case strings.Contains(code, "48;") || strings.Contains(code, "40") || strings.Contains(code, "47"):
						bgSet = true
					}
				}
				if p == "\n" {
					continue
				}
				if len(p) > 0 && !bgSet {
					total += len([]rune(p))
				}
			}
			if total > 0 {
				t.Logf("%s: first unpainted cells on row %d", name, rowIdx)
			}
		}
		if total != 0 {
			t.Errorf("%s view has %d cells without a background (terminal default leaks)", name, total)
		}
	}
}

// TestInputCursorRendersBlock verifies the textarea draws its cursor as a
// reverse-video block in the rendered input, and that the blink loop is
// wired: feeding the exported kickoff message through Update returns a
// re-arm command (the periodic blink ticks keep flowing to the textarea).

// sgrReverseRe matches SGR sequences that enable reverse video (code 7),
// possibly combined with other codes like "\x1b[7;38;2;...m".
var sgrReverseRe = regexp.MustCompile(`\x1b\[[0-9;]*7[0-9;]*m`)

// TestInputCursorRendersBlock verifies the textarea draws its cursor as a
// reverse-video block in the rendered input (the reversed cell carries the
// cursor color), and that the blink loop is wired: feeding the exported
// kickoff message through Update returns a re-arm command, which is only
// produced while the cursor is in blink mode and focused — i.e. it will
// blink on the periodic tick the runtime drives.
func TestInputCursorRendersBlock(t *testing.T) {
	m := newTestModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
	m.chatTextarea.SetValue("ab")
	m.chatTextarea.SetCursorColumn(1)

	// After Focus() the cursor block is visible: the char under the cursor
	// renders in reverse video.
	if !sgrReverseRe.MatchString(m.renderInput()) {
		t.Error("rendered input has no reverse-video cursor block")
	}

	// The exported kickoff message must be forwarded to the textarea, which
	// accepts it (blink mode + focused) and returns the periodic blink
	// command, keeping the loop alive.
	_, cmd := m.Update(textarea.Blink())
	if cmd == nil {
		t.Error("blink kickoff was not accepted: no re-arm command returned")
	}
}

func TestSlashPanelDistinct(t *testing.T) {
	m := newTestModel()
	m.panelOpen = true
	m.panelFromSlash = true

	slash := m.renderSlashPanel()
	// The "/"-picker is a pull-up sheet: no title row, slim key hints, side
	// borders (no full frame), and the selected row filled with the warning
	// orange that matches the borders.
	for _, want := range []string{"↑↓ move · enter run · esc close"} {
		if !strings.Contains(slash, want) {
			t.Errorf("slash panel missing %q:\n%s", want, utils.StripANSI(slash))
		}
	}
	if strings.Contains(utils.StripANSI(slash), "Slash commands") {
		t.Errorf("slash panel should not show a title row:\n%s", utils.StripANSI(slash))
	}
	for _, frame := range []string{"┏", "┗", "┛", "┓"} {
		if strings.Contains(slash, frame) {
			t.Errorf("slash panel must not use an outer frame %q:\n%s", frame, utils.StripANSI(slash))
		}
	}
	if !strings.Contains(slash, "48;2;250;178;131") {
		t.Errorf("selected row should be filled with the command-active peach:\n%s", utils.StripANSI(slash))
	}
	palette := utils.StripANSI(m.renderCommandPanel())
	if palette == utils.StripANSI(slash) {
		t.Error("slash panel and command palette render identically")
	}
	for _, frame := range []string{"┃", "┏", "┗", "┛", "┓"} {
		if strings.Contains(palette, frame) {
			t.Errorf("command palette popup must be borderless (no %q)", frame)
		}
	}
}

func TestSlashSheetWindowAndSelection(t *testing.T) {
	m := newTestModel()
	m.panelOpen = true
	m.panelFromSlash = true
	// 18 commands exist, but the sheet shows at most maxSheetRows; the
	// window slides so the selection stays visible.
	m.panelIdx = len(allPanelCommands()) - 1
	slash := utils.StripANSI(m.renderSlashPanel())
	rows := 0
	for _, ln := range strings.Split(slash, "\n") {
		if strings.Contains(ln, "/") {
			rows++
		}
	}
	if rows != maxSheetRows {
		t.Errorf("windowed sheet shows %d rows, want %d:\n%s", rows, maxSheetRows, slash)
	}
	last := allPanelCommands()[len(allPanelCommands())-1].slash
	if !strings.Contains(slash, last) {
		t.Errorf("selected (last) command %q must be visible in the window:\n%s", last, slash)
	}
	// The selected row carries the command-active-peach fill; verify it contains
	// the selected slash right after an orange-background sequence.
	raw := m.renderSlashPanel()
	if !strings.Contains(raw, "48;2;250;178;131") {
		t.Errorf("selection must be highlighted with the command-active-peach fill")
	}
}

func TestSlashSheetDocksToInput(t *testing.T) {
	m := newTestModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.messages = append(m.messages, ChatMessage{Role: "user", Content: "hello"})
	m.needAutoScroll = false
	m.inChat = true
	m.panelOpen = true
	m.panelMode = panelModeCommand
	m.panelFromSlash = true

	lines := strings.Split(utils.StripANSI(m.View().Content), "\n")
	sheetTop, sheetBottom, inputBottom := -1, -1, -1
	for i, ln := range lines {
		switch {
		// The sheet has no title row; its first content row is the first
		// visible command, so use that as the sheet-top marker.
		case strings.Contains(ln, "/sessions") && sheetTop < 0:
			sheetTop = i
		case strings.Contains(ln, "esc close") && sheetBottom < 0:
			sheetBottom = i
		case strings.Contains(ln, "╹"):
			inputBottom = i
		}
	}
	if sheetTop < 0 || sheetBottom < 0 {
		t.Fatalf("slash sheet not found in view:\n%s", strings.Join(lines, "\n"))
	}
	if inputBottom < 0 {
		t.Fatalf("input box not found in view:\n%s", strings.Join(lines, "\n"))
	}
	if sheetBottom >= inputBottom {
		t.Errorf("sheet bottom row %d must be above the input box bottom %d", sheetBottom, inputBottom)
	}
	// Docked: the input box is 5 rows tall (pad, textarea, pad, badges,
	// footer), so a flush sheet's last row sits exactly inputBottom-5.
	if sheetBottom != inputBottom-5 {
		t.Errorf("sheet must dock flush to the input top (sheet bottom %d, input bottom %d)", sheetBottom, inputBottom)
	}
	if sheetTop == 0 || sheetTop > inputBottom {
		t.Errorf("sheet top at row %d, input box bottom at %d", sheetTop, inputBottom)
	}
}

func TestWindowTitle(t *testing.T) {
	for _, inChat := range []bool{false, true} {
		m := newTestModel()
		m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
		m.inChat = inChat
		if got := m.View().WindowTitle; got != windowTitle {
			t.Errorf("inChat=%v WindowTitle = %q, want %q", inChat, got, windowTitle)
		}
	}
}

// TestViewMouseModeNone guards the text-selection fix: the TUI must not
// enable bubbletea's CellMotion mouse mode. CellMotion emits \x1b[?1002h,
// which forwards every button drag to the app and disables the terminal
// emulator's native text selection. Mouse tracking is driven manually in
// app.go as plain 1000h tracking (wheel/click still arrive, drags stay free
// for box selection).
func TestViewMouseModeNone(t *testing.T) {
	if got := createView("hello").MouseMode; got != tea.MouseModeNone {
		t.Errorf("MouseMode = %v, want MouseModeNone (manual 1000h tracking)", got)
	}
}
