// IAC MCP Server — expose IaC agents as MCP tools for Claude Desktop.
//
// Install:
//
//	go install github.com/yusheng-g/openagent-go/cmd/iac-mcp@latest
//
// Claude Desktop config:
//
//	{
//	  "mcpServers": {
//	    "openagent-iac": {
//	      "command": "iac-mcp",
//	      "env": {
//	        "OPENAGENT_API_KEY": "sk-xxx",
//	        "OPENAGENT_MODEL": "claude-sonnet-5",
//	        "OPENAGENT_BASE_URL": "https://api.anthropic.com/v1",
//	        "DRY_RUN": "false",
//	        "HW_ACCESS_KEY": "xxx",
//	        "HW_SECRET_KEY": "xxx",
//	        "HW_REGION": "cn-north-4"
//	      }
//	    }
//	  }
//	}
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	openagent "github.com/yusheng-g/openagent-go"
	agentmcp "github.com/yusheng-g/openagent-go/mcp"
	"github.com/yusheng-g/openagent-go/model/openai"
	"github.com/yusheng-g/openagent-go/sandbox/native"
	"github.com/yusheng-g/openagent-go/skill/fs"
	opentool "github.com/yusheng-g/openagent-go/tool"

	iactools "github.com/yusheng-g/openagent-go/examples/iac/tools"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed templates/*.tf.tmpl
var templatesFS embed.FS

//go:embed skills/*
var skillsFS embed.FS

func main() {
	apiKey := os.Getenv("OPENAGENT_API_KEY")
	modelID := os.Getenv("OPENAGENT_MODEL")
	baseURL := os.Getenv("OPENAGENT_BASE_URL")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "OPENAGENT_API_KEY not set")
os.Exit(1)
	}
	if modelID == "" {
		modelID = "claude-sonnet-5"
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	dryRun := os.Getenv("DRY_RUN") != "false"

	workspace := os.Getenv("IAC_WORKSPACE")
	if workspace == "" {
		home, _ := os.UserHomeDir()
		workspace = filepath.Join(home, ".openagent", "iac", "workspace")
	}
	os.MkdirAll(workspace, 0755)

	extractTemplates(workspace)
	skillDir := extractSkills(workspace)

	model := openai.New(apiKey, modelID, baseURL).WithContextWindow(128_000)

	var fileTools []openagent.Tool
	if _, err := native.New(workspace); err == nil {
		fileTools = []openagent.Tool{
			opentool.NewReadFile(workspace),
			opentool.NewWriteFile(workspace),
			opentool.NewListDir(workspace),
		}
	}

	tfTool := iactools.NewTerraformTool(workspace, dryRun)
	tfTool.EnsureDir()

	server := agentmcp.NewServer("openagent-iac", "1.0.0", nil)

	for _, def := range buildIACAgents(model, tfTool, fileTools, skillDir) {
		agent := def.Runner.(*openagent.Agent)
		tool := agent.AsTool()
		// Override Description (AsTool copies agent Description, but we
		// want the MCP-facing description which may differ from the
		// agent's own Description used for Plan/Planner routing).
		if dryRun {
			tool = &descOverrideTool{tool, def.Description + " [DRY RUN]"}
		} else {
			tool = &descOverrideTool{tool, def.Description}
		}
		server.AddTool(tool)
	}

	if err := server.Run(context.Background(), &mcpsdk.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ── Asset extraction ──

func extractTemplates(workspace string) {
	tmplDir := filepath.Join(workspace, "templates")
	os.MkdirAll(tmplDir, 0755)
	entries, _ := templatesFS.ReadDir("templates")
	for _, e := range entries {
		b, _ := templatesFS.ReadFile("templates/" + e.Name())
		os.WriteFile(filepath.Join(tmplDir, e.Name()), b, 0644)
	}
}

func extractSkills(workspace string) string {
	skillDir := filepath.Join(workspace, "skills")
	entries, _ := skillsFS.ReadDir("skills")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := skillsFS.ReadFile("skills/" + e.Name() + "/SKILL.md")
		if err != nil {
			continue
		}
		os.MkdirAll(filepath.Join(skillDir, e.Name()), 0755)
		os.WriteFile(filepath.Join(skillDir, e.Name(), "SKILL.md"), b, 0644)
	}
	return skillDir
}

// ── Tool adapter ──

type descOverrideTool struct {
	inner openagent.Tool
	desc  string
}

func (t *descOverrideTool) Definition() openagent.FunctionDefinition {
	d := t.inner.Definition()
	d.Description = t.desc
	return d
}
func (t *descOverrideTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return t.inner.Execute(ctx, args)
}

// ── Agent definitions ──

type agentDef struct {
	Name, Description string
	Runner            openagent.AgentRunner
}

func buildIACAgents(
	model openagent.Model,
	tfTool *iactools.TerraformTool,
	fileTools []openagent.Tool,
	skillDir string,
) []agentDef {
	tfT := tfTool.AsTools()

	var skillLoader openagent.SkillLoader
	if skillDir != "" {
		skillLoader = fs.New(skillDir)
	}

	intentParser := openagent.NewAgent("iac_intent_parse",
		openagent.WithModel(model),
		openagent.WithDescription("Analyze a deployment goal (natural language or GitHub URL) to infer application runtime, database, cache, storage, CDN, and HTTPS requirements. Takes a goal string; returns a JSON ApplicationProfile with fields: runtime, database, cache, storage, cdn (bool), https (bool), gpu (bool), container, traffic."),
		openagent.WithSystemPrompts(`You are an Application Intent Parser. Analyze the deployment request and output a JSON ApplicationProfile.

## Inference Rules
- WordPress / blog → runtime=php, database=mysql, cache=redis, storage=obs, cdn=true, https=true
- Node.js / React → runtime=nodejs
- Spring Boot / Java → runtime=java, database=postgres, cache=redis
- "production" / "高可用" → traffic=high
- "Docker" / "K8s" → container=docker or k8s

Output ONLY the JSON object. No markdown, no explanation.`),
		openagent.WithMaxTurns(1),
	)

	architect := openagent.NewAgent("iac_architecture_design",
		openagent.WithModel(model),
		openagent.WithDescription("Design cloud architecture from an ApplicationProfile JSON. Uses pricing and deployment pattern skills. Produces 3 options (A/B/C) with resource lists and monthly cost estimates in RMB. The caller should present all 3 to the user."),
		openagent.WithSkillLoader(skillLoader),
		openagent.WithSystemPrompts(`You are a Cloud Architect. Given an ApplicationProfile JSON, design 3 options:
- 方案A (最低成本): Minimum. Smallest flavors, no HA.
- 方案B (推荐): Balanced cost/reliability. Standard production.
- 方案C (高可用): Multi-AZ HA for high traffic.

Load use_skill("deployment-patterns") for pricing and patterns. Output ONLY a JSON array.`),
		openagent.WithMaxTurns(3),
	)

	moduleTools := []openagent.Tool{tfT[0]} // terraform_init
	if len(fileTools) > 0 {
		moduleTools = append(moduleTools, fileTools...)
	}

	modulePlanner := openagent.NewAgent("iac_generate_terraform",
		openagent.WithModel(model),
		openagent.WithDescription("Generate Terraform .tf files from a chosen architecture plan. Reads module templates from disk, fills template variables with concrete values, and writes the resulting .tf files to the terraform/ directory. Takes the selected architecture option description as input. After writing, runs terraform_init to verify."),
		openagent.WithSkillLoader(skillLoader),
		openagent.WithTools(moduleTools...),
		openagent.WithSystemPrompts(`Read templates/*.tf.tmpl with read_file. Replace all {{ .Name }}, {{ .Flavor }}, etc. placeholders with concrete values from the architecture plan. Write results to terraform/<name>.tf. Run terraform_init to verify.`),
		openagent.WithMaxTurns(5),
	)

	reviewer := openagent.NewAgent("iac_review_plan",
		openagent.WithModel(model),
		openagent.WithDescription("Review Terraform configurations: run terraform init and plan, then produce a human-readable summary of what resources will be created, estimated monthly cost, risk assessment, and approve/reject recommendation. Safe — no changes are applied."),
		openagent.WithTools(tfT[0], tfT[1]),
		openagent.WithSystemPrompts(`Run terraform_init then terraform_plan. Produce: Resources to Create (list), Estimated Monthly Cost, Risk Assessment, Recommendation (Approve/Reject).`),
		openagent.WithMaxTurns(3),
	)

	applier := openagent.NewAgent("iac_apply",
		openagent.WithModel(model),
		openagent.WithDescription("Apply an approved Terraform plan. Creates real cloud resources — this is DESTRUCTIVE. Only call after the user has reviewed the plan output and explicitly approved."),
		openagent.WithTools(tfT[2]),
		openagent.WithSystemPrompts(`Run terraform_apply. Report results: what was created, endpoints, next steps. NEVER apply without explicit approval.`),
		openagent.WithMaxTurns(2),
	)

	monitor := openagent.NewAgent("iac_monitor",
		openagent.WithModel(model),
		openagent.WithDescription("Post-deployment monitoring. Run terraform output to get connection endpoints, report health status for each resource, and suggest cost/security/performance optimizations."),
		openagent.WithTools(tfT[3]),
		openagent.WithSystemPrompts(`Run terraform_output. Report endpoints and health status: ✅ operational, ⚠️ degraded, ❌ failed. Suggest optimizations with estimated savings.`),
		openagent.WithMaxTurns(2),
	)

	return []agentDef{
		{Name: "iac_intent_parse", Description: intentParser.Description, Runner: intentParser},
		{Name: "iac_architecture_design", Description: architect.Description, Runner: architect},
		{Name: "iac_generate_terraform", Description: modulePlanner.Description, Runner: modulePlanner},
		{Name: "iac_review_plan", Description: reviewer.Description, Runner: reviewer},
		{Name: "iac_apply", Description: applier.Description, Runner: applier},
		{Name: "iac_monitor", Description: monitor.Description, Runner: monitor},
	}
}
