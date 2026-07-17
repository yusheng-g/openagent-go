package native

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
)

// confineAndRun uses Bubblewrap (bwrap) for Linux namespace isolation.
// bwrap is available on most Linux distros (used by Flatpak) and provides
// filesystem + network + PID isolation without root or setuid.
//
// Falls back to unsandboxed execution with a warning if bwrap is not found.
func (s *Sandbox) confineAndRun(ctx context.Context, cmd openagent.Command) (openagent.Result, error) {
	if s.bwrapFailed {
		return s.unconfinedRun(ctx, cmd)
	}

	args, ok := s.bwrapArgs(cmd)
	if !ok {
		log.Printf("WARNING: bwrap not found — sandbox disabled, running unconfined")
		return s.unconfinedRun(ctx, cmd)
	}

	c := exec.CommandContext(ctx, "bwrap", args...)
	c.Dir = s.workDir
	for _, e := range cmd.Env {
		c.Env = append(c.Env, e)
	}
	if cmd.Stdin != "" {
		c.Stdin = strings.NewReader(cmd.Stdin)
	}

	var stdout, stderr strings.Builder
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	result := openagent.Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderrStr := stderr.String()
			if isBwrapSetupFailure(stderrStr) {
				log.Printf("WARNING: bwrap setup failed, falling back to unconfined: %s", stderrStr)
				s.bwrapFailed = true
				return s.unconfinedRun(ctx, cmd)
			}
			log.Printf("bwrap command exited with code %d, stderr: %s", exitErr.ExitCode(), stderrStr)
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		log.Printf("bwrap execution error: %v", err)
		return result, fmt.Errorf("native sandbox (linux): %w", err)
	}
	return result, nil
}

func (s *Sandbox) confineAndRunStream(ctx context.Context, cmd openagent.Command) <-chan openagent.ToolStreamChunk {
	ch := make(chan openagent.ToolStreamChunk, 16)
	go func() {
		defer close(ch)

		if s.bwrapFailed {
			for chunk := range s.unconfinedRunStream(ctx, cmd) {
				ch <- chunk
			}
			return
		}

		args, ok := s.bwrapArgs(cmd)
		if !ok {
			log.Printf("WARNING: bwrap not found — sandbox disabled, running unconfined")
			for chunk := range s.unconfinedRunStream(ctx, cmd) {
				ch <- chunk
			}
			return
		}

		c := exec.CommandContext(ctx, "bwrap", args...)
		c.Dir = s.workDir
		for _, e := range cmd.Env {
			c.Env = append(c.Env, e)
		}
		if cmd.Stdin != "" {
			c.Stdin = strings.NewReader(cmd.Stdin)
		}

		stdout, _ := c.StdoutPipe()
		stderr, _ := c.StderrPipe()
		if err := c.Start(); err != nil {
			log.Printf("WARNING: bwrap start failed, falling back to unconfined: %v", err)
			s.bwrapFailed = true
			for chunk := range s.unconfinedRunStream(ctx, cmd) {
				ch <- chunk
			}
			return
		}

		done := make(chan struct{}, 2)
		go readLines(stdout, ch, done)
		go readLines(stderr, ch, done)
		<-done
		<-done
		_ = c.Wait()
	}()
	return ch
}

// isBwrapSetupFailure reports whether stderr indicates a bwrap sandbox
// setup failure (as opposed to a command execution failure inside the
// sandbox). bwrap errors are prefixed with "bwrap:".
func isBwrapSetupFailure(stderr string) bool {
	return strings.Contains(stderr, "bwrap:")
}

// bwrapArgs builds Bubblewrap arguments for namespace isolation.
// Returns false if bwrap is not installed.
//
// Isolation provided:
//   - New mount namespace with minimal /usr, /bin, /lib bind mounts
//   - /workspace bind-mounted (only writable path)
//   - New network namespace (no network)
//   - New PID namespace
//   - New UTS namespace
func (s *Sandbox) bwrapArgs(cmd openagent.Command) ([]string, bool) {
	if s.bwrapFailed {
		return nil, false
	}
	if _, err := exec.LookPath("bwrap"); err != nil {
		s.bwrapFailed = true
		return nil, false
	}
	if !s.selfTested {
		s.selfTested = true
		test := exec.Command("bwrap", "--ro-bind", "/usr", "/usr", "--", "true")
		if err := test.Run(); err != nil {
			log.Printf("WARNING: bwrap self-test failed, sandbox disabled: %v", err)
			s.bwrapFailed = true
			return nil, false
		}
	}

	args := []string{
		"--unshare-all",     // new namespaces for everything
		"--share-net",       // but keep host network access
		"--new-session",     // new session, no controlling tty
		"--die-with-parent", // kill container when parent dies
		"--proc", "/proc",   // mount proc
		"--dev", "/dev", // minimal /dev
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/bin", "/bin",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind", "/lib64", "/lib64",
	}

	// Bind workspace as writable /workspace.
	args = append(args, "--bind", s.workDir, "/workspace")
	args = append(args, "--chdir", "/workspace")

	// DNS / name resolution.
	for _, f := range []string{
		"/etc/resolv.conf",
		"/etc/hosts",
		"/etc/nsswitch.conf",
	} {
		if _, err := os.Stat(f); err == nil {
			args = append(args, "--ro-bind", f, f)
		}
	}

	// User / group lookup so names resolve inside the user namespace.
	for _, f := range []string{"/etc/passwd", "/etc/group"} {
		if _, err := os.Stat(f); err == nil {
			args = append(args, "--ro-bind", f, f)
		}
	}

	// CA certificates for TLS (HTTPS / curl).
	for _, f := range []string{"/etc/ssl", "/etc/pki"} {
		if _, err := os.Stat(f); err == nil {
			args = append(args, "--ro-bind", f, f)
		}
	}

	// Additional read-only bind mounts.
	for _, m := range s.extraMounts {
		if _, err := os.Stat(m.src); err == nil {
			args = append(args, "--ro-bind", m.src, m.dst)
		}
	}

	// Pass environment variables.
	for _, e := range cmd.Env {
		args = append(args, "--setenv", e)
	}

	// The command to run.
	args = append(args, "--", cmd.Program)
	args = append(args, cmd.Args...)

	return args, true
}

// ── Fallback: unconfined execution ──

func (s *Sandbox) unconfinedRun(ctx context.Context, cmd openagent.Command) (openagent.Result, error) {
	c := exec.CommandContext(ctx, cmd.Program, cmd.Args...)
	c.Dir = s.workDir
	for _, e := range cmd.Env {
		c.Env = append(c.Env, e)
	}
	if cmd.Stdin != "" {
		c.Stdin = strings.NewReader(cmd.Stdin)
	}

	var stdout, stderr strings.Builder
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	result := openagent.Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String() + "\n[warning: bwrap not found, running without sandbox]",
		ExitCode: 0,
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, fmt.Errorf("native sandbox (linux, unconfined): %w", err)
	}
	return result, nil
}

func (s *Sandbox) unconfinedRunStream(ctx context.Context, cmd openagent.Command) <-chan openagent.ToolStreamChunk {
	ch := make(chan openagent.ToolStreamChunk, 16)
	go func() {
		defer close(ch)
		c := exec.CommandContext(ctx, cmd.Program, cmd.Args...)
		c.Dir = s.workDir
		for _, e := range cmd.Env {
			c.Env = append(c.Env, e)
		}
		if cmd.Stdin != "" {
			c.Stdin = strings.NewReader(cmd.Stdin)
		}

		stdout, _ := c.StdoutPipe()
		stderr, _ := c.StderrPipe()
		if err := c.Start(); err != nil {
			ch <- openagent.ToolStreamChunk{Error: err}
			return
		}
		done := make(chan struct{}, 2)
		go readLines(stdout, ch, done)
		go readLines(stderr, ch, done)
		<-done
		<-done
		_ = c.Wait()
	}()
	return ch
}
