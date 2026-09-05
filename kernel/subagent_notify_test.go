package kernel

import (
	"strings"
	"testing"
)

func TestFormatSubAgentNote(t *testing.T) {
	cases := []struct {
		name           string
		agentID        string
		description    string
		result         string
		stopReason     string
		wantSubstrs    []string
		notWantSubstrs []string
	}{
		{
			name:        "with description and result",
			agentID:     "explorer-1",
			description: "find auth module",
			result:      "auth lives in /src/auth/",
			wantSubstrs: []string{
				"<system-reminder>",
				"[SUB-AGENT COMPLETED]",
				"explorer-1",
				"find auth module",
				"Result:",
				"auth lives in /src/auth/",
				"</system-reminder>",
			},
		},
		{
			name:        "empty description omits parenthetical",
			agentID:     "researcher-2",
			description: "",
			result:      "found 3 sources",
			wantSubstrs: []string{
				"Sub-agent researcher-2 has completed.",
				"found 3 sources",
			},
			notWantSubstrs: []string{"()"},
		},
		{
			name:        "max_turns marks incomplete",
			agentID:     "explorer-3",
			description: "survey configs",
			result:      "partial findings...",
			stopReason:  "max_turns",
			wantSubstrs: []string{
				"[INCOMPLETE: this sub-agent hit its turn limit",
				"partial, not a finished answer",
			},
		},
		{
			name:        "error stop reason",
			agentID:     "reviewer-1",
			description: "audit API layer",
			result:      "",
			stopReason:  "error",
			wantSubstrs: []string{
				"[ERROR: this sub-agent failed to complete.]",
			},
		},
		{
			name:           "empty result omits Result section",
			agentID:        "explorer-4",
			description:    "check logs",
			result:         "",
			stopReason:     "",
			wantSubstrs:    []string{"Sub-agent explorer-4"},
			notWantSubstrs: []string{"Result:"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatSubAgentNote(tc.agentID, tc.description, tc.result, tc.stopReason)
			for _, want := range tc.wantSubstrs {
				if !strings.Contains(got, want) {
					t.Errorf("FormatSubAgentNote output missing %q\ngot:\n%s", want, got)
				}
			}
			for _, notWant := range tc.notWantSubstrs {
				if strings.Contains(got, notWant) {
					t.Errorf("FormatSubAgentNote output should not contain %q\ngot:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestFormatSubAgentNote_TaskLabelNotIdentity(t *testing.T) {
	// The description parameter is a task label (e.g. "find auth module"),
	// NOT the sub-agent's identity description. The note must carry the
	// task label so the model can tell which delegation completed. The
	// agent identity is already in the agent_id prefix (explorer/researcher/
	// reviewer).
	note := FormatSubAgentNote("explorer-1", "find auth module", "result", "")
	if !strings.Contains(note, "(find auth module)") {
		t.Errorf("task label should appear in parenthetical; got:\n%s", note)
	}
	// The identity description (e.g. "Collect and organize LOCAL information")
	// must NOT appear — that's the tool Description, not the task label.
	if strings.Contains(note, "Collect and organize") {
		t.Errorf("identity description leaked into notification; got:\n%s", note)
	}
}
