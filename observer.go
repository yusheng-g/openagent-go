package openagent

import (
	"context"
	"time"
)

// ── Stage constants ──

const (
	StageMemoryFetch  = "memory.fetch"  // Recent() + Search(), turn 1 only
	StageGuardIn      = "guard.in"      // input guard, before loop
	StagePromptBuild  = "prompt.build"  // build messages for model
	StageModelCall    = "model.call"    // ChatCompletion + streaming
	StageGuardOut     = "guard.out"     // output guard (model + tool results)
	StageToolExecute  = "tool.execute"  // single tool execution
	StageMemoryAppend = "memory.append" // write message to storage

	// Team-level stages (Team observer, not Agent observer).
	StageTeamAgent = "team.agent" // agent enter/leave within a team run
	StageTeamRoute = "team.route" // router fallback event
)

// ── StageEvent ──

// StageEvent is emitted by the Runner at each stage boundary.
type StageEvent struct {
	Name     string         // stage constant
	Phase    string         // "enter" or "leave"
	Detail   map[string]any // optional metadata (tool name, model ID, etc.)
	Duration time.Duration  // wall-clock time of the stage, set on "leave"
	Err      error          // non-nil if the stage failed
}

// ── RunObserver ──

// RunObserver receives stage-level events from the Runner mainline loop.
// nil RunObserver = events are silently dropped.
//
// Unlike RunHooks which observes agent/tool lifecycles, RunObserver observes
// every stage inside the 8-node loop — memory fetch, prompt build, guard checks,
// model calls, tool execution, and memory append.
type RunObserver interface {
	ObserveStage(ctx context.Context, event StageEvent)
}

// MultiObserver combines multiple RunObservers into one.
// Each observer is called in order; one observer failing does not
// prevent subsequent observers from running. Nil observers are skipped.
func MultiObserver(observers ...RunObserver) RunObserver {
	var filtered []RunObserver
	for _, o := range observers {
		if o != nil {
			filtered = append(filtered, o)
		}
	}
	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		return &multiObserver{list: filtered}
	}
}

type multiObserver struct {
	list []RunObserver
}

func (m *multiObserver) ObserveStage(ctx context.Context, event StageEvent) {
	for _, o := range m.list {
		o.ObserveStage(ctx, event)
	}
}
