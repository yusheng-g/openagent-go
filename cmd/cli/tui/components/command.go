package components

// Command describes a slash-command entry shown in panels and tips.
type Command struct {
	Title  string
	Key    string
	Slash  string
	Alias  string
	Icon   string
	Desc   string
	Enable bool
	Space  bool // toggle with space key
}

// VisibleConfig holds the transcript visibility toggles. Only the fields the
// view layer reads are kept; the toggle interactions are out of scope.
type VisibleConfig struct {
	// ExpandThinking false (default) collapses thought cards to a one-line
	// summary ("Thinking..." / "Thought for 3s"); true renders the full
	// live content (/toggle_thinking).
	ExpandThinking bool
	ShowToolSkill  bool
	ShowToolShell  bool
	ShowToolDetail bool
}
