// ArtifactHook saves large tool results to disk so the model can
// inspect them with read/grep instead of consuming context window.
//
// Used as a built-in RunHooks: when a tool result exceeds Threshold bytes,
// the content is written to /tmp/openagent/<sessionID>/<tool>_<ts>.txt and
// the result is replaced with a short pointer message.
//
// Default threshold: 64 * 1024 (64 KB). Set to 0 to always save.
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
)

// ArtifactHookDefaultThreshold is the default size (bytes) above which a
// tool result is saved to disk rather than passed inline to the model.
const ArtifactHookDefaultThreshold = 64 * 1024 // 64 KB

// ArtifactHook saves large tool results to disk.
type ArtifactHook struct {
	Threshold int    // bytes; 0 = always save
	Prefix    string // optional prefix for the saved-result message
}

// OnAgentStart is a no-op — artifact hook only cares about tool results.
func (h *ArtifactHook) OnAgentStart(ctx context.Context, req openagent.ChatCompletionRequest) (any, error) {
	return nil, nil
}

// OnAgentEnd is a no-op.
func (h *ArtifactHook) OnAgentEnd(ctx context.Context, req openagent.ChatCompletionRequest, resp *openagent.ChatCompletionResponse, runErr error, startState any) {
}

// OnToolStart is a no-op — artifacts are handled after execution.
func (h *ArtifactHook) OnToolStart(ctx context.Context, tool openagent.FunctionDefinition, args json.RawMessage) (any, error) {
	return nil, nil
}

// OnToolEnd checks result size and saves to disk when it exceeds the threshold.
// Reads from the artifact directory are excluded to prevent artifact-of-artifact
// recursion.
func (h *ArtifactHook) OnToolEnd(ctx context.Context, tool openagent.FunctionDefinition, args json.RawMessage, result *string, err *error, startState any) {
	if result == nil || *result == "" {
		return
	}

	threshold := h.Threshold
	if threshold <= 0 {
		threshold = ArtifactHookDefaultThreshold
	}
	if len(*result) <= threshold {
		return
	}

	// Don't re-save reads from the artifact directory itself.
	if isReadingArtifact(args) {
		return
	}

	session, ok := openagent.SessionFromContext(ctx)
	if !ok {
		return // no session context — passthrough
	}

	dir := filepath.Join(ArtifactRoot(), session.ID)
	_ = os.MkdirAll(dir, 0755)

	name := fmt.Sprintf("%s_%d.txt", sanitizeArtifactName(tool.Name), time.Now().UnixNano())
	path := filepath.Join(dir, name)

	raw := *result
	_ = os.WriteFile(path, []byte(raw), 0644)

	// Replace the result with a terse pointer. The model is smart enough
	// to read/grep the file when it needs details.
	sizeKB := (len(raw) + 1023) / 1024
	prefix := h.Prefix
	if prefix != "" {
		prefix += ": "
	}
	*result = fmt.Sprintf("%sTool output saved to %s (%d KB, %d lines). Use read or grep to inspect.",
		prefix, path, sizeKB, strings.Count(raw, "\n")+1)
}

// isReadingArtifact checks whether args contain a path inside the artifact
// root. Used to prevent artifact-of-artifact recursion when read/grep
// inspect a previously saved artifact.
func isReadingArtifact(args json.RawMessage) bool {
	var params struct {
		Path string `json:"path"`
	}
	json.Unmarshal(args, &params)
	return params.Path != "" && strings.HasPrefix(params.Path, ArtifactRoot())
}

// sanitizeArtifactName replaces characters that are unsafe in filenames
// (slash, null) with underscores.
func sanitizeArtifactName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\x00", "_")
	return name
}
