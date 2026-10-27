package file

import (
	"context"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
)

// TestForkCompressedCopy drives Runtime.Fork over the file backend,
// covering CopyCompressed (the compressed-summary file duplicates under
// the fork's ID).
func TestForkCompressedCopy(t *testing.T) {
	ms, err := NewMessageStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewMessageStore: %v", err)
	}
	ctx := context.Background()

	cc := &openagent.CompressedContext{Summary: "earlier discussion", ThroughIndex: 2}
	if err := ms.writeCompressed("src", cc); err != nil {
		t.Fatalf("writeCompressed: %v", err)
	}

	if err := ms.CopyCompressed(ctx, "src", "dst"); err != nil {
		t.Fatalf("CopyCompressed: %v", err)
	}
	got, err := ms.readCompressed("dst")
	if err != nil || got == nil {
		t.Fatalf("readCompressed(dst) = %v, %v; want the copy", got, err)
	}
	if got.Summary != cc.Summary || got.ThroughIndex != cc.ThroughIndex {
		t.Fatalf("copied = %+v, want %+v", got, cc)
	}

	// No summary on the source: no-op, no error.
	if err := ms.CopyCompressed(ctx, "absent", "dst2"); err != nil {
		t.Fatalf("CopyCompressed(no summary): %v", err)
	}
	if got, _ := ms.readCompressed("dst2"); got != nil {
		t.Fatalf("readCompressed(dst2) = %+v, want nil", got)
	}
}
