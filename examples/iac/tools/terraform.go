// Package tools provides IaC-specific tool implementations for openagent-go.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
)

// TerraformTool wraps terraform CLI for use as an openagent.Tool.
// In dry-run mode, commands are simulated without calling tf binary.
type TerraformTool struct {
	workDir string
	dryRun  bool
}

// NewTerraformTool creates a Terraform tool rooted at workDir.
func NewTerraformTool(workDir string, dryRun bool) *TerraformTool {
	return &TerraformTool{workDir: workDir, dryRun: dryRun}
}

func (t *TerraformTool) WorkDir() string { return t.workDir }

// tfDir returns the terraform config directory.
func (t *TerraformTool) tfDir() string { return filepath.Join(t.workDir, "terraform") }

// ── Tool: terraform_init ──

func (t *TerraformTool) terraformInitDef() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "terraform_init",
		Description: "Initialize a Terraform working directory. Downloads providers and modules. Run this before any other terraform commands.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
	}
}

func (t *TerraformTool) CanSelfApprove(name string, _ json.RawMessage) bool {
	switch name {
	case "terraform_init", "terraform_plan", "terraform_output":
		return true
	default:
		return false
	}
}

func (t *TerraformTool) terraformInit(ctx context.Context) (string, error) {
	if t.dryRun {
		return "[DRY RUN] terraform init — would download providers and initialize working directory", nil
	}
	cmd := exec.CommandContext(ctx, "terraform", "init", "-input=false")
	cmd.Dir = t.tfDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("terraform init: %w\n%s", err, out)
	}
	return string(out), nil
}

// ── Tool: terraform_plan ──

func (t *TerraformTool) terraformPlanDef() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name: "terraform_plan",
		Description: "Generate a Terraform execution plan. Shows what resources will be created, modified, or destroyed. " +
			"Safe to run — no changes are applied.",
		Parameters: json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
	}
}

func (t *TerraformTool) terraformPlan(ctx context.Context) (string, error) {
	if t.dryRun {
		return t.simulatedPlan(), nil
	}
	cmd := exec.CommandContext(ctx, "terraform", "plan", "-input=false", "-out=tfplan")
	cmd.Dir = t.tfDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("terraform plan: %w\n%s", err, out)
	}

	// Also get structured JSON plan for parsing
	showCmd := exec.CommandContext(ctx, "terraform", "show", "-json", "tfplan")
	showCmd.Dir = t.tfDir()
	showOut, showErr := showCmd.CombinedOutput()
	if showErr != nil {
		return string(out) + "\n\nPlan JSON unavailable: " + showErr.Error(), nil
	}
	return string(out) + "\n\n## Plan JSON\n```json\n" + string(showOut) + "\n```", nil
}

// ── Tool: terraform_apply ──

func (t *TerraformTool) terraformApplyDef() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name: "terraform_apply",
		Description: "Apply the Terraform execution plan. This CREATES or MODIFIES real cloud resources. " +
			"Requires approval. Use only after reviewing terraform_plan output.",
		Parameters: json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
	}
}

func (t *TerraformTool) terraformApply(ctx context.Context) (string, error) {
	if t.dryRun {
		return "[DRY RUN] terraform apply — would create/modify cloud resources per the plan", nil
	}
	cmd := exec.CommandContext(ctx, "terraform", "apply", "-input=false", "-auto-approve", "tfplan")
	cmd.Dir = t.tfDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("terraform apply: %w\n%s", err, out)
	}
	return string(out), nil
}

// terraformApplyStream returns a channel streaming terraform apply output line by line.
func (t *TerraformTool) terraformApplyStream(ctx context.Context) <-chan openagent.ToolStreamChunk {
	ch := make(chan openagent.ToolStreamChunk, 16)
	go func() {
		defer close(ch)
		if t.dryRun {
			for _, line := range strings.Split(t.simulatedApply(), "\n") {
				ch <- openagent.ToolStreamChunk{Content: line + "\n"}
			}
			return
		}
		cmd := exec.CommandContext(ctx, "terraform", "apply", "-input=false", "-auto-approve", "tfplan")
		cmd.Dir = t.tfDir()
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err := cmd.Start(); err != nil {
			ch <- openagent.ToolStreamChunk{Error: err}
			return
		}
		go readLines(stdout, ch)
		go readLines(stderr, ch)
		_ = cmd.Wait()
	}()
	return ch
}

// ── Tool: terraform_output ──

func (t *TerraformTool) terraformOutputDef() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "terraform_output",
		Description: "Get outputs from the applied Terraform state (e.g., public IPs, connection strings).",
		Parameters:  json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
	}
}

func (t *TerraformTool) terraformOutput(ctx context.Context) (string, error) {
	if t.dryRun {
		return t.simulatedOutput(), nil
	}
	cmd := exec.CommandContext(ctx, "terraform", "output", "-json")
	cmd.Dir = t.tfDir()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("terraform output: %w\n%s", err, out)
	}
	return string(out), nil
}

// ── Tool: terraform_destroy ──

func (t *TerraformTool) terraformDestroyDef() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "terraform_destroy",
		Description: "Destroy all resources managed by this Terraform configuration. DESTRUCTIVE — requires approval.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
	}
}

type destroyStream struct{ tf *TerraformTool }

func (ds *destroyStream) terraformDestroyStream(ctx context.Context) <-chan openagent.ToolStreamChunk {
	ch := make(chan openagent.ToolStreamChunk, 16)
	go func() {
		defer close(ch)
		if ds.tf.dryRun {
			ch <- openagent.ToolStreamChunk{Content: "[DRY RUN] terraform destroy — would destroy all resources\n"}
			return
		}
		cmd := exec.CommandContext(ctx, "terraform", "destroy", "-input=false", "-auto-approve")
		cmd.Dir = ds.tf.tfDir()
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		if err := cmd.Start(); err != nil {
			ch <- openagent.ToolStreamChunk{Error: err}
			return
		}
		go readLines(stdout, ch)
		go readLines(stderr, ch)
		_ = cmd.Wait()
	}()
	return ch
}

// ── openagent.Tool interface ──

// AsTools returns all terraform sub-tools.
func (t *TerraformTool) AsTools() []openagent.Tool {
	return []openagent.Tool{
		&tfInitTool{t},
		&tfPlanTool{t},
		&tfApplyTool{t},
		&tfOutputTool{t},
		&tfDestroyTool{t},
	}
}

type tfInitTool struct{ tf *TerraformTool }

func (tt *tfInitTool) Definition() openagent.FunctionDefinition            { return tt.tf.terraformInitDef() }
func (tt *tfInitTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) { return tt.tf.terraformInit(ctx) }

type tfPlanTool struct{ tf *TerraformTool }

func (tt *tfPlanTool) Definition() openagent.FunctionDefinition                  { return tt.tf.terraformPlanDef() }
func (tt *tfPlanTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) { return tt.tf.terraformPlan(ctx) }

type tfApplyTool struct{ tf *TerraformTool }

func (tt *tfApplyTool) Definition() openagent.FunctionDefinition                  { return tt.tf.terraformApplyDef() }
func (tt *tfApplyTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) { return tt.tf.terraformApply(ctx) }

type tfOutputTool struct{ tf *TerraformTool }

func (tt *tfOutputTool) Definition() openagent.FunctionDefinition                  { return tt.tf.terraformOutputDef() }
func (tt *tfOutputTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) { return tt.tf.terraformOutput(ctx) }

type tfDestroyTool struct{ tf *TerraformTool }

func (tt *tfDestroyTool) Definition() openagent.FunctionDefinition                  { return tt.tf.terraformDestroyDef() }
func (tt *tfDestroyTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) {
	return "[requires streaming]", fmt.Errorf("use terraform_destroy_stream")
}

// ── Helpers ──

func readLines(r interface{ Read([]byte) (int, error) }, ch chan<- openagent.ToolStreamChunk) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			ch <- openagent.ToolStreamChunk{Content: string(buf[:n])}
		}
		if err != nil {
			return
		}
	}
}

// Simulated outputs for dry-run mode

func (t *TerraformTool) simulatedPlan() string {
	files, _ := filepath.Glob(filepath.Join(t.tfDir(), "*.tf"))
	if len(files) == 0 {
		return "[DRY RUN] No .tf files found in " + t.tfDir() + "\nTerraform plan would show 0 changes."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("[DRY RUN] Found %d .tf files in %s\n\n", len(files), t.tfDir()))
	b.WriteString("Plan: 3 to add, 0 to change, 0 to destroy.\n")
	b.WriteString("Resources to be created:\n")
	for _, f := range files {
		base := filepath.Base(f)
		if base == "provider.tf" {
			continue
		}
		b.WriteString(fmt.Sprintf("  + %s\n", strings.TrimSuffix(base, ".tf")))
	}
	b.WriteString("\nEstimated monthly cost: see architecture recommendation for pricing.")
	return b.String()
}

func (t *TerraformTool) simulatedApply() string {
	files, _ := filepath.Glob(filepath.Join(t.tfDir(), "*.tf"))
	if len(files) == 0 {
		return "[DRY RUN] No resources to apply.\n"
	}
	var b strings.Builder
	b.WriteString("[DRY RUN] Applying Terraform configuration...\n")
	for _, f := range files {
		base := filepath.Base(f)
		if base == "provider.tf" {
			continue
		}
		name := strings.TrimSuffix(base, ".tf")
		b.WriteString(fmt.Sprintf("Creating %s... ████████████ done\n", name))
	}
	b.WriteString("\nApply complete! Resources: 3 added, 0 changed, 0 destroyed.\n")
	return b.String()
}

func (t *TerraformTool) simulatedOutput() string {
	return `{
  "web_server_public_ip": {"value": "123.60.xxx.xxx"},
  "rds_private_ip": {"value": "192.168.1.xxx"},
  "rds_port": {"value": 5432},
  "obs_bucket_domain": {"value": "myapp-assets.obs.cn-north-4.myhuaweicloud.com"},
  "cdn_cname": {"value": "cdn.xxxxxx.com.c.cdnhwc1.com"}
}
[DRY RUN] These are simulated outputs.`
}

// Ensure tempDir exists for terraform configs.
func (t *TerraformTool) EnsureDir() error {
	return os.MkdirAll(t.tfDir(), 0755)
}

// Used by main.go to embed templates.
var (
	ProviderTemplate []byte
	ECSTemplate      []byte
	RDSTemplate      []byte
	OBSTemplate      []byte
	CDNTemplate      []byte
)

// WriteTemplate writes an embedded template to the terraform directory.
func (t *TerraformTool) WriteTemplate(name string, tmpl []byte, data map[string]any) error {
	// Simple variable substitution for the Go templates.
	content := string(tmpl)
	for k, v := range data {
		placeholder := "{{ ." + k + " }}"
		if sv, ok := v.(string); ok {
			content = strings.ReplaceAll(content, placeholder, sv)
		}
	}
	return os.WriteFile(filepath.Join(t.tfDir(), name), []byte(content), 0644)
}
