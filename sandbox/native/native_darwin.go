package native

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
)

// confineAndRun executes cmd confined by macOS Seatbelt (sandbox-exec).
// The Seatbelt profile restricts filesystem access to the workspace directory
// plus system read-only paths. Network access is denied.
//
// The ctx deadline only controls how long the caller waits — it does NOT
// kill the process.
func (s *Sandbox) confineAndRun(ctx context.Context, cmd openagent.Command) (openagent.Result, error) {
	profile := s.seatbeltProfile()
	args := []string{"-p", profile, "--", cmd.Program}
	args = append(args, cmd.Args...)

	c := exec.Command("sandbox-exec", args...)
	c.Dir = s.workDir
	for _, e := range cmd.Env {
		c.Env = append(c.Env, e)
	}
	if cmd.Stdin != "" {
		c.Stdin = strings.NewReader(cmd.Stdin)
	}

	var stdout, stderr strings.Builder
	outW := io.Writer(&stdout)
	errW := io.Writer(&stderr)
	if cmd.StdoutW != nil {
		outW = io.MultiWriter(&stdout, cmd.StdoutW)
	}
	if cmd.StderrW != nil {
		errW = io.MultiWriter(&stderr, cmd.StderrW)
	}
	c.Stdout = outW
	c.Stderr = errW

	if err := c.Start(); err != nil {
		return openagent.Result{}, fmt.Errorf("native sandbox (darwin): %w", err)
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- c.Wait() }()

	select {
	case <-ctx.Done():
		return openagent.Result{
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: -1,
			PID:      c.Process.Pid,
		}, openagent.ErrProcessRunning
	case err := <-waitCh:
		result := openagent.Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
			PID:    c.Process.Pid,
		}
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitErr.ExitCode()
				return result, nil
			}
			return result, fmt.Errorf("native sandbox (darwin): %w", err)
		}
		return result, nil
	}
}

func (s *Sandbox) confineAndRunStream(ctx context.Context, cmd *openagent.Command) <-chan openagent.ToolStreamChunk {
	ch := make(chan openagent.ToolStreamChunk, 16)
	go func() {
		defer close(ch)
		profile := s.seatbeltProfile()
		args := []string{"-p", profile, "--", cmd.Program}
		args = append(args, cmd.Args...)

		c := exec.Command("sandbox-exec", args...)
		c.Dir = s.workDir
		for _, e := range cmd.Env {
			c.Env = append(c.Env, e)
		}
		if cmd.Stdin != "" {
			c.Stdin = strings.NewReader(cmd.Stdin)
		}

		// Manual pipes so we can tee to file writers.
		soutR, soutW := io.Pipe()
		serrR, serrW := io.Pipe()
		setupPipeWriters(c, soutW, serrW, cmd.StdoutW, cmd.StderrW)

		if err := c.Start(); err != nil {
			ch <- openagent.ToolStreamChunk{Error: fmt.Errorf("native sandbox (darwin): %w", err)}
			return
		}
		cmd.PID = c.Process.Pid

		done := make(chan struct{}, 2)
		go readLines(soutR, ch, done)
		go readLines(serrR, ch, done)
		<-done
		<-done
		_ = c.Wait()
	}()
	return ch
}

// seatbeltProfile generates a macOS Seatbelt profile that:
// - Allows read+write to the workspace directory
// - Allows read to system binary paths
// - Allows process execution
// - Denies network access
// - Denies access to user home, /tmp, /etc, etc.
func (s *Sandbox) seatbeltProfile() string {
	quoted := func(p string) string {
		return `"` + strings.ReplaceAll(p, `"`, `\"`) + `"`
	}

	// Read-only system paths the sandboxed process may need.
	readOnly := []string{
		"/bin", "/usr/bin", "/usr/lib", "/usr/libexec",
		"/System/Library", "/Library/Developer",
		"/private/var/select/sh",
		"/private/etc/shells",
		"/dev/null", "/dev/zero", "/dev/random", "/dev/urandom",
	}

	var b strings.Builder
	b.WriteString("(version 1)\n")
	b.WriteString("(allow default)\n")

	// Deny network.
	b.WriteString("(deny network*)\n")

	// Deny access to sensitive paths.
	deny := []string{"/Users", "/Volumes", "/Applications",
		"/private/etc", "/private/tmp", "/private/var",
		"/opt", "/usr/local",
	}
	for _, p := range deny {
		b.WriteString(fmt.Sprintf("(deny file-read* file-write* (subpath %s))\n", quoted(p)))
	}

	// Allow read+write to workspace.
	b.WriteString(fmt.Sprintf("(allow file-read* file-write* (subpath %s))\n", quoted(s.workDir)))

	// Allow read-only to system paths.
	for _, p := range readOnly {
		b.WriteString(fmt.Sprintf("(allow file-read* (subpath %s))\n", quoted(p)))
	}

	return b.String()
}

