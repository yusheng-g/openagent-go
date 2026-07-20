//go:build linux

package native

import (
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
)

func TestBwrapArgsNetworkPolicy(t *testing.T) {
	cmd := openagent.Command{Program: "echo"}

	// Default (zero-value) policy: network shared with host →
	// neither --share-net nor --unshare-net in args (we simply don't
	// unshare the network namespace).
	sbDefault, err := NewWithPolicy(t.TempDir(), Policy{})
	if err != nil {
		t.Fatal(err)
	}
	args, ok := sbDefault.bwrapArgs(cmd)
	if !ok {
		t.Skip("bwrap not installed")
	}
	if sliceContains(args, "--unshare-net") {
		t.Errorf("default policy: expected NO --unshare-net in bwrap args, got %v", args)
	} else {
		t.Logf("✅ default policy: --unshare-net absent (network shared with host)")
	}
	if sliceContains(args, "--share-net") {
		t.Errorf("default policy: expected NO --share-net in bwrap args (obsolete with explicit-namespace approach), got %v", args)
	}

	// Explicit "host" policy: same as default.
	sbHost, err := NewWithPolicy(t.TempDir(), Policy{Network: "host"})
	if err != nil {
		t.Fatal(err)
	}
	args, ok = sbHost.bwrapArgs(cmd)
	if !ok {
		t.Skip("bwrap not installed")
	}
	if sliceContains(args, "--unshare-net") {
		t.Errorf("host policy: expected NO --unshare-net in bwrap args, got %v", args)
	}

	// Isolated policy: network denied → --unshare-net present.
	sbIsolated, err := NewWithPolicy(t.TempDir(), Policy{Network: "isolated"})
	if err != nil {
		t.Fatal(err)
	}
	args, ok = sbIsolated.bwrapArgs(cmd)
	if !ok {
		t.Skip("bwrap not installed")
	}
	if !sliceContains(args, "--unshare-net") {
		t.Errorf("isolated policy: expected --unshare-net in bwrap args, got %v", args)
	} else {
		t.Logf("✅ isolated policy: --unshare-net present")
	}
}

func TestBwrapArgsEtcMounts(t *testing.T) {
	sb, err := NewWithPolicy(t.TempDir(), Policy{})
	if err != nil {
		t.Fatal(err)
	}
	args, ok := sb.bwrapArgs(openagent.Command{Program: "echo"})
	if !ok {
		t.Skip("bwrap not installed")
	}

	// Network-critical /etc files must be mounted read-only so DNS
	// resolution and TLS verification work inside the sandbox.
	wantPaths := []string{
		"/etc/resolv.conf",
		"/etc/hosts",
		"/etc/nsswitch.conf",
		"/etc/ssl",
	}
	for _, want := range wantPaths {
		found := false
		for i, a := range args {
			if (a == "--ro-bind-try" || a == "--ro-bind") && i+2 < len(args) && args[i+1] == want && args[i+2] == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected read-only bind for %s in bwrap args, got: %v", want, args)
		} else {
			t.Logf("✅ %s mounted read-only", want)
		}
	}
}

func TestBwrapArgsExtraPaths(t *testing.T) {
	sb, err := NewWithPolicy(t.TempDir(), Policy{
		WritablePaths: []string{"/tmp/openagent-wtest"},
		ReadablePaths: []string{"/etc/ssl"},
	})
	if err != nil {
		t.Fatal(err)
	}
	args, ok := sb.bwrapArgs(openagent.Command{Program: "echo"})
	if !ok {
		t.Skip("bwrap not installed")
	}

	foundWritable := false
	foundReadable := false
	for i, a := range args {
		if a == "--bind" && i+2 < len(args) && args[i+1] == "/tmp/openagent-wtest" && args[i+2] == "/tmp/openagent-wtest" {
			foundWritable = true
		}
		if a == "--ro-bind" && i+2 < len(args) && args[i+1] == "/etc/ssl" && args[i+2] == "/etc/ssl" {
			foundReadable = true
		}
	}
	if !foundWritable {
		t.Errorf("expected writable bind for /tmp/openagent-wtest, args: %v", args)
	} else {
		t.Logf("✅ writable path bound")
	}
	if !foundReadable {
		t.Errorf("expected readable bind for /etc/ssl, args: %v", args)
	} else {
		t.Logf("✅ readable path bound")
	}
}

func sliceContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
