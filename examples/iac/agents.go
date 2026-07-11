package main

import (
	"embed"
	"encoding/json"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/skill/fs"

	"github.com/yusheng-g/openagent-go/examples/iac/tools"
)

//go:embed templates/*.tf.tmpl
var templatesFS embed.FS

//go:embed skills/*
var skillsFS embed.FS

// newFSLoader creates a skill loader from the given directory.
func newFSLoader(dir string) openagent.SkillLoader { return fs.New(dir) }

// ── Schema types ──

// ApplicationProfile is the structured output of intent parsing.
type ApplicationProfile struct {
	Runtime   string `json:"runtime"`
	Database  string `json:"database"`
	Cache     string `json:"cache"`
	Storage   string `json:"storage"`
	CDN       bool   `json:"cdn"`
	HTTPS     bool   `json:"https"`
	GPU       bool   `json:"gpu"`
	Container string `json:"container"`
	Traffic   string `json:"traffic"`
}

// ArchitecturePlan is one recommended architecture option.
type ArchitecturePlan struct {
	Name         string   `json:"name"`
	Resources    []string `json:"resources"`
	MonthlyPrice int      `json:"monthly_price"`
	Description  string   `json:"description"`
}

func parseApplicationProfile(raw string) (*ApplicationProfile, error) {
	var p ApplicationProfile
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func parseArchitecturePlans(raw string) ([]ArchitecturePlan, error) {
	var plans []ArchitecturePlan
	if err := json.Unmarshal([]byte(raw), &plans); err != nil {
		return nil, err
	}
	return plans, nil
}

// ── Agent definitions ──

type agentDef struct {
	Name        string
	Description string
	Runner      openagent.AgentRunner
}

// buildIACAgents creates the 6-agent IaC pipeline.
// fileTools are read_file/write_file/ls (nil if sandbox unavailable).
// skillDir is the workspace skills/ path for the skill loader.
func buildIACAgents(
	model openagent.Model,
	tfTool *tools.TerraformTool,
	fileTools []openagent.Tool,
	skillDir string,
) []agentDef {
	tfT := tfTool.AsTools()
	tfRO := []openagent.Tool{tfT[0], tfT[1], tfT[3]} // init, plan, output
	tfApply := []openagent.Tool{tfT[2]}               // apply

	var skillLoader openagent.SkillLoader
	if skillDir != "" {
		skillLoader = fs.New(skillDir)
	}

	intentParser := openagent.NewAgent("intent_parser",
		openagent.WithModel(model),
		openagent.WithDescription("Analyzes user input (GitHub URL or natural language) to infer application runtime, database, cache, storage, CDN, HTTPS needs. Outputs a structured ApplicationProfile JSON."),
		openagent.WithInstructions(`You are an Application Intent Parser. Analyze the user's deployment request and produce a JSON ApplicationProfile.

## Inference Rules
- GitHub URL with package.json → runtime=nodejs
- WordPress / blog → runtime=php, database=mysql, cache=redis, storage=obs, cdn=true, https=true
- Spring Boot / Java → runtime=java, database=postgres, cache=redis
- Static site / SPA / React / Vue → runtime=static, cdn=true, database=none
- AI / ML project → gpu=true
- "production" / "高可用" → traffic=high
- "Docker" / "K8s" / "container" → container=docker or k8s

Output ONLY the JSON object. No markdown, no explanation.

{
  "runtime": "nodejs",
  "database": "postgres",
  "cache": "redis",
  "storage": "obs",
  "cdn": true,
  "https": true,
  "gpu": false,
  "container": "none",
  "traffic": "medium"
}`),
		openagent.WithMaxTurns(1),
	)

	architect := openagent.NewAgent("architect",
		openagent.WithModel(model),
		openagent.WithDescription("Designs cloud architecture from an ApplicationProfile. Uses pricing and deployment pattern skills. Produces 3 architecture options (A/B/C) with resource lists and monthly cost estimates. Requires human approval to select which option to implement."),
		openagent.WithSkillLoader(skillLoader),
		openagent.WithInstructions(`You are a Cloud Architect. Design infrastructure for the given ApplicationProfile.

## Process
1. Load skills: use_skill("deployment-patterns") for pricing and patterns.
2. Based on the ApplicationProfile, design 3 options:

- 方案A (最低成本): Minimum viable. Smallest flavors, no HA.
- 方案B (推荐): Balanced cost/reliability. Standard production.
- 方案C (高可用): Multi-AZ HA for high traffic.

Use pricing from the deployment-patterns skill. Be specific about resource types and sizes.

## Output
Return ONLY a JSON array. No markdown fences.

[
  {
    "name": "方案A 最低成本",
    "resources": ["ECS 2C4G x1", "OBS 100GB"],
    "monthly_price": 180,
    "description": "Single ECS with local storage..."
  },
  ...
]`),
		openagent.WithMaxTurns(3),
	)

	// Tools for module_planner: read templates, write .tf files, terraform_init.
	moduleTools := []openagent.Tool{tfT[0]} // terraform_init
	if len(fileTools) > 0 {
		moduleTools = append(moduleTools, fileTools...)
	}

	modulePlanner := openagent.NewAgent("module_planner",
		openagent.WithModel(model),
		openagent.WithDescription("Translates a selected architecture plan into Terraform configuration files. Reads module templates from disk, fills variables, and writes .tf files."),
		openagent.WithSkillLoader(skillLoader),
		openagent.WithTools(moduleTools...),
		openagent.WithInstructions(`You are a Terraform Module Planner. Generate .tf files from templates for the chosen architecture plan.

## Process
1. Use read_file to read templates/templates/*.tf.tmpl to see what variables each module needs.
2. Load skills: use_skill("terraform-modules") for module documentation.
3. Use read_file to read each needed template, replace {{ .Name }}, {{ .Flavor }}, etc. with real values.
4. Use write_file to write the result to terraform/<name>.tf
5. Run terraform_init to verify the configuration.

## Template Files (read_file)
- templates/provider.tf.tmpl → terraform/provider.tf
- templates/ecs.tf.tmpl → terraform/ecs.tf
- templates/rds.tf.tmpl → terraform/rds.tf
- templates/obs.tf.tmpl → terraform/obs.tf
- templates/cdn.tf.tmpl → terraform/cdn.tf

## Important
- Replace ALL placeholders with concrete values before writing.
- Do NOT run terraform_plan or terraform_apply — that is the reviewer/applier's job.
- Use sensible defaults for unspecified values.`),
		openagent.WithMaxTurns(5),
	)

	reviewer := openagent.NewAgent("reviewer",
		openagent.WithModel(model),
		openagent.WithDescription("Reviews Terraform configurations: runs terraform init + plan, parses plan output, produces human-readable summary of changes, costs, and risks."),
		openagent.WithTools(tfRO...),
		openagent.WithInstructions(`You are a Terraform Plan Reviewer. Review configurations before they are applied.

## Process
1. Run terraform_init
2. Run terraform_plan
3. Review the plan output carefully
4. Produce a clear summary:

**Resources to Create**: list each with type
**Estimated Monthly Cost**: based on resource types
**Estimated Time**: rough apply duration
**Risk Assessment**: any concerns
**Recommendation**: Approve or Reject with reasoning

Be concise. This is an approval gate.`),
		openagent.WithMaxTurns(3),
	)

	applier := openagent.NewAgent("applier",
		openagent.WithModel(model),
		openagent.WithDescription("Applies approved Terraform plans. Runs terraform apply and reports results."),
		openagent.WithTools(tfApply...),
		openagent.WithInstructions(`You are the Terraform Applier. Apply an approved plan.

## Process
1. Confirm the plan has been reviewed and approved
2. Run terraform_apply
3. Report results: what was created, endpoints, next steps

## Safety
- NEVER apply without explicit approval
- If apply fails, report the error clearly
- After success, suggest running terraform_output for connection details`),
		openagent.WithMaxTurns(2),
	)

	monitor := openagent.NewAgent("monitor",
		openagent.WithModel(model),
		openagent.WithDescription("Post-deployment health checks and optimization recommendations."),
		openagent.WithTools(tfT[3]), // terraform_output
		openagent.WithInstructions(`You are a Post-Deployment Monitor.

## Process
1. Run terraform_output to get resource endpoints
2. Report health status for each resource: ✅ / ⚠️ / ❌
3. Suggest optimizations:
   - Cost: idle or oversized instances
   - Security: missing HTTPS, public exposure
   - Performance: add CDN, resize instances

## Output
- Health Summary (one per resource)
- Optimization Suggestions (numbered, with estimated savings)`),
		openagent.WithMaxTurns(2),
	)

	return []agentDef{
		{Name: "intent_parser", Description: intentParser.Description, Runner: intentParser},
		{Name: "architect", Description: architect.Description, Runner: architect},
		{Name: "module_planner", Description: modulePlanner.Description, Runner: modulePlanner},
		{Name: "reviewer", Description: reviewer.Description, Runner: reviewer},
		{Name: "applier", Description: applier.Description, Runner: applier},
		{Name: "monitor", Description: monitor.Description, Runner: monitor},
	}
}
