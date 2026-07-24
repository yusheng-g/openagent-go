package native

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
)

// confineAndRun uses Bubblewrap (bwrap) for Linux namespace isolation.
// bwrap is available on most Linux distros (used by Flatpak) and provides
// filesystem + network isolation without root or setuid.
//
// The ctx deadline only controls how long the caller waits — it does NOT
// kill the process. When ctx expires the process keeps running (bwrap
// continues with --die-with-parent, which only triggers when the parent
// openagent-cli exits, not when a single call times out).
//
// Falls back to unsandboxed execution with a warning if bwrap is not found
// or if bwrap fails to start (e.g. "setting up uid map: Permission denied"
// in containers that block user-namespace setup).
func (s *Sandbox) confineAndRun(ctx context.Context, cmd openagent.Command) (openagent.Result, error) {
	args, ok := s.bwrapArgs(cmd)
	if !ok {
		log.Printf("WARNING: bwrap not found — sandbox disabled, running unconfined")
		return s.unconfinedRun(ctx, cmd)
	}

	c := exec.Command("bwrap", args...)
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
		return openagent.Result{}, fmt.Errorf("native sandbox (linux): %w", err)
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
				// bwrap itself failed to set up the sandbox (not the inner
				// command failing). Detect by: empty stdout + stderr starts
				// with "bwrap:" — bwrap's own error messages have that prefix,
				// while output from the sandboxed command does not.
				if stdout.Len() == 0 && strings.HasPrefix(strings.TrimSpace(stderr.String()), "bwrap:") {
					return s.unconfinedRun(ctx, cmd)
				}
				result.ExitCode = exitErr.ExitCode()
				return result, nil
			}
			return result, fmt.Errorf("native sandbox (linux): %w", err)
		}
		return result, nil
	}
}

func (s *Sandbox) confineAndRunStream(ctx context.Context, cmd *openagent.Command) <-chan openagent.ToolStreamChunk {
	ch := make(chan openagent.ToolStreamChunk, 16)
	go func() {
		defer close(ch)

		args, ok := s.bwrapArgs(*cmd)
		if !ok {
			// Fallback: run unconfined.
			for chunk := range s.unconfinedRunStream(ctx, cmd) {
				ch <- chunk
			}
			return
		}

		c := exec.Command("bwrap", args...)
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
			for chunk := range s.unconfinedRunStream(ctx, cmd) {
				ch <- chunk
			}
			return
		}
		cmd.PID = c.Process.Pid

		// Read stderr first line early so we can detect bwrap setup
		// failures (which happen before the inner command produces any
		// stdout). If bwrap fails, stderr starts with "bwrap:" and the
		// process exits immediately with no stdout.
		done := make(chan struct{}, 2)
		var firstStderr string
		firstStderrCh := make(chan string, 1)
		go readLinesWithFirst(serrR, ch, done, firstStderrCh)
		go readLines(soutR, ch, done)

		waitErr := c.Wait()
		<-done
		<-done
		select {
		case firstStderr = <-firstStderrCh:
		default:
		}

		// Detect bwrap setup failure: no stdout + stderr starts with "bwrap:".
		if waitErr != nil && strings.HasPrefix(firstStderr, "bwrap:") {
			for chunk := range s.unconfinedRunStream(ctx, cmd) {
				ch <- chunk
			}
			return
		}
		if waitErr != nil {
			ch <- openagent.ToolStreamChunk{Error: fmt.Errorf("native sandbox (linux): %w", waitErr)}
		}
	}()
	return ch
}

// readLinesWithFirst is like readLines but also sends the first line it
// reads to firstCh. Used by confineAndRunStream to detect bwrap setup
// failures (whose first stderr line starts with "bwrap:").
func readLinesWithFirst(r io.Reader, ch chan<- openagent.ToolStreamChunk, done chan<- struct{}, firstCh chan<- string) {
	defer func() { done <- struct{}{} }()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 4096), 1024*1024)
	first := true
	for sc.Scan() {
		line := sc.Text()
		if first {
			first = false
			select {
			case firstCh <- line:
			default:
			}
		}
		ch <- openagent.ToolStreamChunk{Content: line + "\n"}
	}
}

// bwrapArgs builds Bubblewrap arguments for namespace isolation.
// Returns false if bwrap is not installed.
//
// Isolation provided:
//   - New mount namespace with minimal /usr, /bin, /lib bind mounts
//   - /workspace bind-mounted (only writable path, plus any WritablePaths)
//   - New UTS namespace (via --unshare-uts)
//   - New IPC namespace (via --unshare-ipc)
//   - User namespace attempted via --unshare-user-try (gracefully skipped
//     in environments that block user-namespace creation, e.g. some
//     containers; bwrap would still need CAP_SYS_ADMIN for the other
//     namespaces in that case — if it can't get them, confineAndRun
//     falls back to unconfined execution)
//
// PID namespace is intentionally NOT unshared so that the model can
// kill long-running processes by PID from another shell invocation.
//
// Network access is governed by s.policy.Network:
//   - "" or "host"     → host network namespace shared (no --unshare-net)
//   - "isolated"       → --unshare-net (new, empty network namespace)
//
// /etc network configuration files (resolv.conf, hosts, nsswitch.conf)
// and CA certificates (/etc/ssl) are bind-mounted read-only so that DNS
// resolution and TLS verification work inside the sandbox. Without these,
// even with --share-net / no network unshare, curl/wget/etc. fail with
// "Couldn't resolve host" because glibc can't read resolv.conf.
func (s *Sandbox) bwrapArgs(cmd openagent.Command) ([]string, bool) {
	if _, err := exec.LookPath("bwrap"); err != nil {
		return nil, false
	}

	args := []string{
		"--unshare-user-try",   // user namespace (graceful: skip if unavailable)
		"--unshare-ipc",        // IPC namespace
		"--unshare-uts",        // UTS namespace (hostname/domainname)
		"--unshare-cgroup-try", // cgroup namespace (graceful)
		"--new-session",        // new session, no controlling tty
		"--die-with-parent",    // kill container when parent dies
		"--proc", "/proc",      // mount proc
		"--dev", "/dev", // minimal /dev
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/bin", "/bin",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind", "/lib64", "/lib64",
		// Network configuration: needed for DNS resolution + TLS even
		// when the network namespace itself is shared with the host.
		// --ro-bind-try avoids bwrap failure on minimal systems that
		// lack some of these files.
		"--ro-bind-try", "/etc/resolv.conf", "/etc/resolv.conf",
		"--ro-bind-try", "/etc/hosts", "/etc/hosts",
		"--ro-bind-try", "/etc/nsswitch.conf", "/etc/nsswitch.conf",
		"--ro-bind-try", "/etc/ssl", "/etc/ssl",
	}

	// Network: only unshare for the isolated policy. Host/default
	// policies leave the network namespace shared with the parent
	// (no flag needed — bwrap shares by default when --unshare-net
	// is absent).
	if s.policy.Network == "isolated" {
		args = append(args, "--unshare-net")
	}

	// Bind workspace as writable /workspace.
	args = append(args, "--bind", s.workDir, "/workspace")
	args = append(args, "--chdir", "/workspace")

	// Extra writable paths (bind-mounted at the same host path).
	for _, p := range s.policy.WritablePaths {
		if p == "" {
			continue
		}
		args = append(args, "--bind", p, p)
	}

	// Extra read-only paths (bind-mounted at the same host path).
	for _, p := range s.policy.ReadablePaths {
		if p == "" {
			continue
		}
		args = append(args, "--ro-bind", p, p)
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
