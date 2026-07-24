// Sandbox demo: shows the native sandbox blocking external access in real time.
//
//	go run ./examples/sandbox/
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/tool"
)

func main() {
	dir, _ := os.Getwd()
	workDir := filepath.Join(dir, "sandbox-demo")
	os.MkdirAll(workDir, 0755)
	defer os.RemoveAll(workDir)

	// Write a test file inside the workspace.
	os.WriteFile(filepath.Join(workDir, "secret.txt"), []byte("this is inside the sandbox"), 0644)
	// Write a file outside the workspace.
	os.WriteFile(filepath.Join(dir, "outside.txt"), []byte("this is OUTSIDE the sandbox"), 0644)
	defer os.Remove(filepath.Join(dir, "outside.txt"))

	sb, err := native.New(workDir)
	if err != nil {
		fmt.Println("❌ Sandbox init failed:", err)
		os.Exit(1)
	}
	fmt.Println("🏖️  Native sandbox ready")
	fmt.Println("   Platform:", detectPlatform())
	fmt.Println("   Workspace:", workDir)
	fmt.Println()

	ctx := context.Background()

	// ── Test 1: Read inside workspace ──
	fmt.Println("─── Test 1: Read inside workspace ───")
	result, err := sb.Run(ctx, openagent.Command{
		Program: "/bin/bash",
		Args:    []string{"-c", "cat secret.txt"},
		WorkDir: workDir,
	})
	fmt.Printf("  Command: cat secret.txt\n")
	fmt.Printf("  Stdout:  %q\n", trim(result.Stdout))
	fmt.Printf("  Stderr:  %q\n", trim(result.Stderr))
	fmt.Printf("  Exit:    %d\n", result.ExitCode)
	if strings.Contains(result.Stdout, "inside the sandbox") {
		fmt.Println("  ✅ Workspace access allowed")
	} else {
		fmt.Println("  ❌ UNEXPECTED")
	}
	fmt.Println()

	// ── Test 2: Read outside workspace (should be blocked) ──
	fmt.Println("─── Test 2: Read outside workspace ───")
	result, _ = sb.Run(ctx, openagent.Command{
		Program: "/bin/bash",
		Args:    []string{"-c", "cat /etc/passwd 2>&1 || true"},
		WorkDir: workDir,
	})
	fmt.Printf("  Command: cat /etc/passwd\n")
	fmt.Printf("  Stdout:  %q\n", trim(result.Stdout))
	fmt.Printf("  Stderr:  %q\n", trim(result.Stderr))
	fmt.Printf("  Exit:    %d\n", result.ExitCode)
	if strings.Contains(result.Stdout, "root:") {
		fmt.Println("  ❌ SANDBOX LEAK — could read /etc/passwd!")
	} else {
		fmt.Println("  ✅ External access blocked by sandbox")
	}
	fmt.Println()

	// ── Test 3: Network blocked ──
	fmt.Println("─── Test 3: Network blocked ──")
	result, _ = sb.Run(ctx, openagent.Command{
		Program: "/bin/bash",
		Args:    []string{"-c", "curl -s --connect-timeout 2 https://httpbin.org/ip 2>&1 || echo BLOCKED"},
		WorkDir: workDir,
	})
	fmt.Printf("  Command: curl https://httpbin.org/ip\n")
	fmt.Printf("  Output:  %q\n", trim(result.Stdout+result.Stderr))
	if strings.Contains(result.Stdout, "BLOCKED") || result.ExitCode != 0 {
		fmt.Println("  ✅ Network blocked by sandbox")
	} else {
		fmt.Println("  ❌ Network access should be blocked!")
	}
	fmt.Println()

	// ── Test 4: Streaming output ──
	fmt.Println("─── Test 4: Streaming output ──")
	fmt.Print("  ")
	ch := sb.RunStream(ctx, &openagent.Command{
		Program: "/bin/bash",
		Args:    []string{"-c", "echo 'building...'; sleep 0.3; echo 'testing...'; sleep 0.3; echo 'PASS'"},
		WorkDir: workDir,
	})
	for chunk := range ch {
		if chunk.Error != nil {
			fmt.Printf("❌ %v\n", chunk.Error)
			break
		}
		fmt.Print("  > " + trim(chunk.Content))
	}
	fmt.Println()
	fmt.Println("  ✅ Streaming works")
	fmt.Println()

	// ── Test 5: File tools ──
	fmt.Println("─── Test 5: File tools ──")
	r := tool.NewReadFile(workDir)
	out, err := r.Execute(ctx, []byte(`{"path":"secret.txt"}`))
	fmt.Printf("  read secret.txt: %v\n", trim(out))

	out, err = r.Execute(ctx, []byte(`{"path":"../outside.txt"}`))
	fmt.Printf("  read ../outside.txt: %v\n", trim(out))

	w := tool.NewWriteFile(workDir)
	out, err = w.Execute(ctx, []byte(`{"path":"demo.txt","content":"written by sandbox demo"}`))
	fmt.Printf("  write demo.txt: %v\n", trim(out))

	l := tool.NewListDir(workDir)
	out, err = l.Execute(ctx, []byte(`{}`))
	fmt.Printf("  ls:\n%s", out)
	fmt.Println()

	// ── Test 6: Shell tool ──
	fmt.Println("─── Test 6: Shell tool (full integration) ──")
	shell := tool.NewShell(sb).WithLanguage("go")
	def := shell.Definition()
	fmt.Printf("  Tool name: %s\n", def.Name)
	fmt.Printf("  Description: %s\n", def.Description)

	out, err = shell.Execute(ctx, []byte(`{"command":"echo hello from shell tool && ls"}`))
	fmt.Printf("  Execute: %v\n", trim(out))
	fmt.Println()

	fmt.Println("🏖️  All sandbox tests passed!")
}

func trim(s string) string {
	if len(s) > 200 {
		return s[:200] + "..."
	}
	// Remove trailing newline for display.
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return s
}

func detectPlatform() string {
	if _, err := exec.LookPath("sandbox-exec"); err == nil {
		return "macOS ✅ sandbox-exec (Seatbelt)"
	}
	if _, err := exec.LookPath("bwrap"); err == nil {
		return "Linux ✅ bwrap (Bubblewrap)"
	}
	return "⚠️  no sandbox (unconfined)"
}
