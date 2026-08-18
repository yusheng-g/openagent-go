// Package version holds the build-time identity of the binary: the agent
// implementation name and version string. Both are injected via ldflags at
// compile time and fall back to dev defaults in init so a bare `go build`
// still reports a usable identity.
//
// Typical ldflags injection (see scripts/release.sh):
//
//	-X github.com/yusheng-g/openagent-go/version.Name=foo \
//	-X github.com/yusheng-g/openagent-go/version.Version=v1.2.3
//
// When ldflags are absent (development build), init fills Name with
// "hwcloud" and Version with "0.0.0-dev.<build-timestamp>".
package version

import "time"

// Name is the agent implementation name reported to peers (e.g. in ACP
// initialize agentInfo.name and MCP client identity). Inject via
// -X ...version.Name=<name>; defaults to "hwcloud" in init.
var Name = ""

// Version is the build version reported to peers and via `--version`.
// Inject via -X ...version.Version=<ver>; defaults to
// "0.0.0-dev.YYYYMMDDHHMMSS" in init.
var Version = ""

func init() {
	if Name == "" {
		Name = "hwcloud"
	}
	if Version == "" {
		Version = "0.0.0-dev." + time.Now().Format("20060102150405")
	}
}
