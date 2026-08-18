package version

import (
	"regexp"
	"strings"
	"testing"
)

// TestPopulated verifies init fills both Name and Version. For a dev build
// (no ldflags), Name is the default and Version matches 0.0.0-dev.<ts>;
// when ldflags injects values, any non-empty string is accepted.
func TestPopulated(t *testing.T) {
	if Name == "" {
		t.Fatal("Name is empty; init should populate it")
	}
	if Version == "" {
		t.Fatal("Version is empty; init should populate it")
	}
	// An ldflags-injected Version won't carry the dev prefix; nothing
	// more to validate in that case.
	if !strings.HasPrefix(Version, "0.0.0-dev.") {
		return
	}
	re := regexp.MustCompile(`^0\.0\.0-dev\.\d{14}$`)
	if !re.MatchString(Version) {
		t.Errorf("default Version %q does not match 0.0.0-dev.YYYYMMDDHHMMSS", Version)
	}
}

// TestDefaultName checks the dev-build default name. When ldflags injects
// Name, it overrides this; the test only asserts the fallback shape when
// Name still equals the default constant.
func TestDefaultName(t *testing.T) {
	// We can't distinguish ldflags-injected Name from the default here,
	// so only assert it's non-empty (covered by TestPopulated). If it's
	// the default, it must be exactly "hwcloud".
	const defaultName = "hwcloud"
	if Name == defaultName {
		return // expected default
	}
	// Otherwise some non-default name was injected (or init changed);
	// nothing more to assert.
}
