// Package agent provides server-side LLM reasoning for iac-server.
//
// The Planner uses a separate LLM (configured via LLM_* env vars) to:
//   - Read embedded terraform examples and generate .tf files for a request
//   - Query cloud pricing via the BSS API and web search
//   - Diagnose deployment errors and suggest fixes
//
// Skills (SKILL.md guides) are statically loaded via the SkillLoader and
// injected directly into each agent's system prompt — no runtime load_skill
// tool call needed. The LLM browses reference files (examples, API swagger)
// with standard read/grep/ls tools.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/cmd/mcp/iac-server/provider"
	sloghooks "github.com/yusheng-g/openagent-go/hooks/slog"
	"github.com/yusheng-g/openagent-go/iac"
	opentool "github.com/yusheng-g/openagent-go/tool"
)

// Planner holds the dependencies for server-side LLM reasoning.
type Planner struct {
	model           openagent.Model
	cloud           provider.CloudProvider
	loader          openagent.SkillLoader // loads skills from extracted skills dir
	memory          openagent.Memory     // shared across calls, scoped by deployment_id
	workDir         string               // cloud home dir (parent of skills/ and deployments/), workDir for read/grep/ls
	deploymentsDir  string
	dryRun          bool
	binaryMirrors   []string // terraform binary download mirrors
	providerMirrors []string // provider download mirrors
}

// New creates a Planner. workDir should be the cloud home directory
// (parent of skills/ and deployments/) so read/grep/ls can access both.
// memory is shared across all LLM calls and scoped by deployment_id —
// estimate_cost can see plan_deployment's reasoning, troubleshoot can see
// prior attempts. nil disables memory (each call is isolated).
func New(model openagent.Model, cloud provider.CloudProvider, loader openagent.SkillLoader, memory openagent.Memory, workDir, deploymentsDir string, dryRun bool, binaryMirrors, providerMirrors []string) *Planner {
	return &Planner{
		model:           model,
		cloud:           cloud,
		loader:          loader,
		memory:          memory,
		workDir:         workDir,
		deploymentsDir:  deploymentsDir,
		dryRun:          dryRun,
		binaryMirrors:   binaryMirrors,
		providerMirrors: providerMirrors,
	}
}

// sessionID returns the Memory session key for a deployment.
func sessionID(deploymentID string) string {
	return "dep-" + deploymentID
}

// planResult is the JSON returned by plan_deployment.
type planResult struct {
	Status         string   `json:"status"` // "need_input" or "ready"
	Questions      []string `json:"questions,omitempty"`
	DeploymentID   string   `json:"deployment_id,omitempty"`
	Plan           any      `json:"plan,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`
}

// serverContext is the shared context injected into every server-side LLM
// agent. It explains the MCP server's role, the client, the interaction
// model, and the output contract — without this the LLM doesn't know who
// it is serving or how its output is consumed.
const serverContext = `You are the server-side LLM of an MCP server (iac-server) that provides cloud infrastructure deployment and query capabilities over the MCP protocol.

## Your role
- You run on the SERVER side. You never talk to the end user directly.
- The MCP CLIENT (e.g. Claude Code, Cursor, openagent) calls one of the 11 MCP tools and forwards the user's request to you.
- Your output is returned to the client as the tool result. The client then decides what to show the user and whether to proceed.
- You do NOT need user approval for any action — approval is the client's concern, not yours.

## Deployment workflow (6 steps, user confirms between each)
  1. propose_architecture  — recommend a cloud architecture (services + reasoning), no .tf files
  2. specify_resources     — determine concrete resource specs (flavor, image, CIDR, etc.)
  3. generate_plan         — write .tf files + run terraform plan, return preview
  4. estimate_cost         — query cloud pricing for the planned resources
  5. apply_deployment       — terraform apply (executed by the server, not you)
  6. troubleshoot_deployment — diagnose errors if any step fails

update_deployment re-runs specify_resources + generate_plan with user adjustments. destroy_deployment, get_deployment_status, and list_deployments do not involve you. query_cloud is for read-only queries about existing resources/bills.

## Credentials
Cloud credentials (e.g. HW_ACCESS_KEY, HW_SECRET_KEY, HW_REGION) are injected by the server into the terraform subprocess environment. NEVER hardcode credentials in .tf files, NEVER ask for them, NEVER put them in variables or tfvars.

## Tools
- read / grep / ls — browse the workspace: skills/ (references, guides) and deployments/ (.tf files)
- http_request — send authenticated HTTP requests to cloud APIs (signing is automatic, do NOT pass credentials). Use ONLY for read-only queries (List/Show/Get APIs). NEVER call Create/Update/Delete/Post/Put APIs to create or modify cloud resources directly — resource provisioning is done through terraform, not through API calls.
- WebSearch / WebFetch — query official cloud docs and pricing pages
- load_skill / reload_skills — (query_cloud only) dynamically load cloud-service skills on demand

## Skills
For propose/specify/generate/estimate/troubleshoot: the relevant skill guide (SKILL.md) is already loaded into your system prompt — you do not need to call any tool to load it. Use read/grep/ls to browse the skill's references/ directory for detailed examples and API definitions.
For query_cloud: use the load_skill tool to load the relevant cloud-service skill on demand (the skill catalog is in your system prompt).

## Output contract
Return ONLY valid JSON as specified by each tool's instructions. Do not wrap in markdown fences. Do not add conversational text outside the JSON. The server parses your output programmatically — any non-JSON text will cause a parse failure.`

// ProposeArchitecture analyzes a deployment request and recommends a cloud
// architecture (step 1 of the 6-step deployment flow). It does NOT write .tf
// files or browse reference examples — it only looks at the service category
// list and matches the request to a known architecture pattern.
//
// Returns a deployment_id (pre-allocated), the proposed architecture, required
// services, and reasoning. If information is incomplete, returns questions.
// The user confirms the architecture before calling specify_resources.
func (p *Planner) ProposeArchitecture(ctx context.Context, request string) (string, error) {
	progress := openagent.ProgressFromContext(ctx)

	// Pre-allocate the deployment ID so all subsequent steps share the same
	// Memory session and deployment directory.
	depID, dir, err := deploymentID(p.deploymentsDir)
	if err != nil {
		return "", fmt.Errorf("propose_architecture: %w", err)
	}

	progress("Loading deployment skill...", 0, 2)
	skillBody := p.loadSkillBody(ctx, "huaweicloud-deploy")
	agent := openagent.NewAgent("iac-architect",
		openagent.WithModel(p.model),
		openagent.WithTools(p.fileTools()...),
		openagent.WithMemory(p.memory),
		openagent.WithRunHooks(sloghooks.New(slog.Default()), newProgressHook()),
		openagent.WithSystemPrompts(
			serverContext,
			skillBody,
			`You are a HuaweiCloud architecture expert. Your job is to RECOMMEND an architecture, NOT write .tf files.

## What to do
1. Parse the user's deployment goal (what to deploy, region, HA, budget, etc.)
2. Run `+"`ls skills/huaweicloud-deploy/references/`"+` to see available service categories
3. Match the request to a known architecture pattern (see patterns below)
4. Return the architecture recommendation

## What NOT to do
- Do NOT read individual .tf files — that happens in generate_plan
- Do NOT browse deep into reference directories
- Do NOT generate .tf configuration

## Common architecture patterns
- **Single web server**: ECS + VPC + Subnet + Security Group + EIP
- **Web + database**: ECS + VPC + Subnet + Security Group + EIP + RDS
- **HA web tier**: ECS×2 + VPC + Subnet + Security Group + ELB + EIP + RDS
- **Web + cache + db**: ECS + VPC + Subnet + Security Group + EIP + DCS(Redis) + RDS
- **Container cluster**: CCE + VPC + Subnet + Security Group + EIP
- **Static site**: OBS + CDN

## Output
Return JSON:
`+"```json"+`
{
  "architecture": "short name, e.g. \"single ECS + VPC + EIP\"",
  "services": ["ecs", "vpc", "eip"],
  "reasoning": "why this architecture was chosen",
  "questions": ["..."]  // only if information is incomplete
}
`+"```"+`
If information is incomplete, return questions instead of architecture.`),
		openagent.WithMaxTurns(5),
	)

	progress("Analyzing deployment request...", 1, 2)
	session := openagent.Session{ID: sessionID(depID)}
	result, err := agent.Run(ctx, session, openagent.UserMessage(request))
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("propose_architecture: LLM run: %w", err)
	}

	raw := extractJSON(result.FinalOutput)
	if raw == "" {
		os.RemoveAll(dir)
		return "", fmt.Errorf("propose_architecture: LLM returned empty output (FinalOutput=%q)", result.FinalOutput)
	}

	var arch struct {
		Architecture string   `json:"architecture"`
		Services     []string `json:"services"`
		Reasoning    string   `json:"reasoning"`
		Questions    []string `json:"questions"`
	}
	if err := json.Unmarshal([]byte(raw), &arch); err != nil {
		os.RemoveAll(dir)
		return marshalResult(planResult{
			Status: "need_input",
			Questions: []string{
				"Could not parse the request. Please provide more details about what you want to deploy, the region, and any requirements.",
			},
		})
	}

	// Information incomplete — ask the client for clarification.
	if len(arch.Questions) > 0 {
		os.RemoveAll(dir)
		return marshalResult(planResult{
			Status:    "need_input",
			Questions: arch.Questions,
		})
	}

	out := map[string]any{
		"deployment_id": depID,
		"architecture":  arch.Architecture,
		"services":      arch.Services,
		"reasoning":     arch.Reasoning,
		"next_step":     "Call specify_resources with this deployment_id to determine resource specs. User should confirm the architecture first.",
	}
	data, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("propose_architecture: marshal: %w", err)
	}
	return string(data), nil
}

// SpecifyResources determines concrete resource specs for a proposed architecture
// (step 2 of the 6-step deployment flow). Reads the architecture from Memory
// (stored by propose_architecture), queries cloud APIs for available specs
// (flavors, images, etc.), and returns a detailed resource list.
//
// The user confirms the resources before calling generate_plan.
func (p *Planner) SpecifyResources(ctx context.Context, deploymentID, adjustments string) (string, error) {
	progress := openagent.ProgressFromContext(ctx)

	dir := filepath.Join(p.deploymentsDir, deploymentID)
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("specify_resources: deployment %s not found — call propose_architecture first", deploymentID)
	}

	progress("Loading deployment skill...", 0, 3)
	skillBody := p.loadSkillBody(ctx, "huaweicloud-deploy")
	agent := openagent.NewAgent("iac-specifier",
		openagent.WithModel(p.model),
		openagent.WithTools(p.fileTools()...),
		openagent.WithMemory(p.memory),
		openagent.WithRunHooks(sloghooks.New(slog.Default()), newProgressHook()),
		openagent.WithSystemPrompts(
			serverContext,
			skillBody,
			`You are a HuaweiCloud resource specification expert. Your job is to determine concrete resource specs for a proposed architecture.

## What to do
1. Read the architecture recommendation from the conversation history (propose_architecture output)
2. Use http_request to query available specs if needed (e.g. ListFlavors for ECS, ListImages for OS images)
3. Determine concrete specs for each resource: flavor, image, disk size, CIDR, etc.
4. Apply any user adjustments to the specs

## What NOT to do
- Do NOT write .tf files — that happens in generate_plan
- Do NOT browse reference directories
- Do NOT create or modify any cloud resources — only read-only API calls (List/Show/Get)

## Output
Return JSON:
`+"```json"+`
{
  "resources": [
    {"type": "huaweicloud_compute_instance", "name": "web", "spec": {"flavor": "s6.large.2", "image": "Ubuntu 22.04", "disk": 40}},
    {"type": "huaweicloud_vpc", "name": "main", "spec": {"cidr": "192.168.0.0/16"}},
    {"type": "huaweicloud_vpc_subnet", "name": "subnet", "spec": {"cidr": "192.168.0.0/24"}},
    {"type": "huaweicloud_vpc_eip", "name": "eip", "spec": {"bandwidth": 5, "type": "5_bgp"}}
  ],
  "reasoning": "why these specs were chosen"
}
`+"```"+`
`),
		openagent.WithMaxTurns(8),
	)

	progress("Determining resource specs...", 1, 3)
	session := openagent.Session{ID: sessionID(deploymentID)}

	userMsg := "Determine concrete resource specs for this deployment based on the architecture from the previous step (propose_architecture). Return JSON with the resources array."
	if adjustments != "" {
		userMsg += "\n\nUser adjustments: " + adjustments
	}

	result, err := agent.Run(ctx, session, openagent.UserMessage(userMsg))
	if err != nil {
		return "", fmt.Errorf("specify_resources: LLM run: %w", err)
	}

	raw := extractJSON(result.FinalOutput)
	if raw == "" {
		return "", fmt.Errorf("specify_resources: LLM returned empty output (FinalOutput=%q)", result.FinalOutput)
	}

	var spec struct {
		Resources []map[string]any `json:"resources"`
		Reasoning string          `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return "", fmt.Errorf("specify_resources: parse: %w (raw=%q)", err, raw)
	}

	out := map[string]any{
		"deployment_id": deploymentID,
		"resources":     spec.Resources,
		"reasoning":     spec.Reasoning,
		"next_step":     "Call generate_plan with this deployment_id to write .tf files and run terraform plan. User should confirm the resources first.",
	}
	data, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("specify_resources: marshal: %w", err)
	}
	progress("Done", 3, 3)
	return string(data), nil
}

// GeneratePlan writes .tf files and runs terraform plan (step 3 of the 6-step
// deployment flow). Reads the architecture + resource specs from Memory, browses
// ONLY the relevant reference examples, generates .tf files, and runs terraform
// init + plan. Retries up to 3 times on plan failure.
//
// The user reviews the plan preview before calling estimate_cost.
func (p *Planner) GeneratePlan(ctx context.Context, deploymentID string) (string, error) {
	progress := openagent.ProgressFromContext(ctx)

	dir := filepath.Join(p.deploymentsDir, deploymentID)
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("generate_plan: deployment %s not found — call propose_architecture first", deploymentID)
	}

	progress("Loading deployment skill...", 0, 4)
	skillBody := p.loadSkillBody(ctx, "huaweicloud-deploy")
	agent := openagent.NewAgent("iac-planner",
		openagent.WithModel(p.model),
		openagent.WithTools(p.fileTools()...),
		openagent.WithMemory(p.memory),
		openagent.WithRunHooks(sloghooks.New(slog.Default()), newProgressHook()),
		openagent.WithSystemPrompts(
			serverContext,
			skillBody,
			`You are a HuaweiCloud terraform configuration expert. Generate .tf files based on the architecture and resource specs from the conversation history.

## What to do
1. Read the architecture + resource specs from the conversation history (propose_architecture + specify_resources output)
2. Browse ONLY the relevant reference examples — e.g. if deploying ECS, look at references/ecs/, NOT all directories
3. Generate .tf files: providers.tf, variables.tf, main.tf, terraform.tfvars
4. Follow the credential rules, naming conventions, and variable design from the skill guide

## What NOT to do
- Do NOT browse all reference directories — only the ones relevant to your resources
- Do NOT hardcode credentials
- Do NOT ask questions — use the specs from history

## Output
Return JSON:
`+"```json"+`
{
  "files": {
    "providers.tf": "...",
    "variables.tf": "...",
    "main.tf": "...",
    "terraform.tfvars": "..."
  },
  "reasoning": "why these .tf configs were generated"
}
`+"```"+``),
		openagent.WithMaxTurns(10),
	)

	session := openagent.Session{ID: sessionID(deploymentID)}
	msg := openagent.UserMessage("Generate terraform .tf files based on the architecture and resource specs from the conversation history.")

	var reasoning string
	for attempt := 0; attempt < 3; attempt++ {
		progress(fmt.Sprintf("Generating .tf files (attempt %d/3)...", attempt+1), float64(attempt), 3)
		result, err := agent.Run(ctx, session, msg)
		if err != nil {
			return "", fmt.Errorf("generate_plan: LLM run (attempt %d): %w", attempt+1, err)
		}

		var llmOutput struct {
			Files     map[string]string `json:"files"`
			Reasoning string            `json:"reasoning"`
		}
		raw := extractJSON(result.FinalOutput)
		if raw == "" {
			return "", fmt.Errorf("generate_plan: LLM returned empty output (attempt %d, FinalOutput=%q)", attempt+1, result.FinalOutput)
		}
		if err := json.Unmarshal([]byte(raw), &llmOutput); err != nil {
			return "", fmt.Errorf("generate_plan: parse (attempt %d): %w (raw=%q)", attempt+1, err, raw)
		}

		if len(llmOutput.Files) == 0 {
			return "", fmt.Errorf("generate_plan: no files generated (attempt %d)", attempt+1)
		}

		reasoning = llmOutput.Reasoning

		// Write .tf files to the deployment directory.
		for name, content := range llmOutput.Files {
			if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
				return "", fmt.Errorf("generate_plan: invalid filename %q", name)
			}
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
				return "", fmt.Errorf("generate_plan: write %s: %w", name, err)
			}
		}

		// terraform init + plan.
		progress("Running terraform init...", 2, 4)
		client, err := iac.NewClient(ctx, dir, iac.Config{
			Env:             p.cloud.Env(),
			DryRun:          p.dryRun,
			BinaryMirrors:   p.binaryMirrors,
			ProviderMirrors: p.providerMirrors,
		})
		if err != nil {
			return "", fmt.Errorf("generate_plan: create terraform client: %w", err)
		}
		if err := client.Init(ctx); err != nil {
			msg = retryMessage("generate .tf files", "terraform init", err, p.workDir, dir)
			continue
		}
		progress("Running terraform plan...", 3, 4)
		plan, err := client.Plan(ctx)
		if err == nil {
			out := map[string]any{
				"deployment_id":  deploymentID,
				"files":          llmOutput.Files,
				"plan_preview":   plan,
				"resource_count": len(llmOutput.Files),
				"reasoning":      reasoning,
				"next_step":      "Call estimate_cost with this deployment_id to get pricing. User should review the plan first.",
			}
			data, err := json.Marshal(out)
			if err != nil {
				return "", fmt.Errorf("generate_plan: marshal: %w", err)
			}
			return string(data), nil
		}
		msg = retryMessage("generate .tf files", "terraform plan", err, p.workDir, dir)
	}

	return "", fmt.Errorf("generate_plan: terraform plan failed after 3 attempts")
}

// UpdateDeployment modifies an existing deployment by re-running specify_resources
// (with user adjustments) + generate_plan. The deployment directory is reused.
//
// Use this when the user wants to adjust an already-planned deployment
// (e.g. change a flavor, rename a resource, add a tag) without starting
// from scratch. The change_request is passed as adjustments to specify_resources.
func (p *Planner) UpdateDeployment(ctx context.Context, deploymentID, changeRequest string) (string, error) {
	// Re-specify resources with the user's adjustments, then regenerate the plan.
	if _, err := p.SpecifyResources(ctx, deploymentID, changeRequest); err != nil {
		return "", fmt.Errorf("update_deployment: specify_resources: %w", err)
	}
	return p.GeneratePlan(ctx, deploymentID)
}

// retryMessage builds the user message for a plan retry attempt.
// workDir is the read/grep/ls workspace root; dir is the deployment
// directory. The LLM is told a path relative to workDir so read/grep/ls
// resolve correctly and we don't leak the server's absolute path.
func retryMessage(request, command string, planErr error, workDir, dir string) openagent.Message {
	tfFiles, _ := readTFFiles(dir)
	relDir, _ := filepath.Rel(workDir, dir)
	return openagent.UserMessage(fmt.Sprintf(`Original request: %s

%s failed with this error:

%s

The current .tf files are in directory: %s

%s

Fix the .tf files and return the corrected versions as JSON:
{"files": {"providers.tf": "...", "variables.tf": "...", "main.tf": "...", "terraform.tfvars": "..."}, "reasoning": "..."}`,
		request, command, planErr.Error(), relDir, tfFiles))
}

// EstimateCost reads the saved terraform plan for a deployment and queries
// cloud pricing via the LLM. The LLM loads the pricing skill and uses
// http_request (BSS API, auto-signed) to query prices, with WebSearch/WebFetch
// as a fallback for public pricing pages. This MUST be called before
// apply_deployment so the user sees the cost.
func (p *Planner) EstimateCost(ctx context.Context, deploymentID string) (string, error) {
	progress := openagent.ProgressFromContext(ctx)

	dir := filepath.Join(p.deploymentsDir, deploymentID)

	// Check that a plan exists — ShowPlan needs the tfplan file.
	planPath := filepath.Join(dir, "tfplan")
	if _, err := os.Stat(planPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("estimate_cost: no plan found for deployment %s — call plan_deployment first", deploymentID)
		}
		return "", fmt.Errorf("estimate_cost: check plan: %w", err)
	}

	// Read the saved plan to get exact resource specs.
	progress("Reading terraform plan...", 0, 3)
	client, err := iac.NewClient(ctx, dir, iac.Config{
		Env:             p.cloud.Env(),
		DryRun:          p.dryRun,
		BinaryMirrors:   p.binaryMirrors,
		ProviderMirrors: p.providerMirrors,
	})
	if err != nil {
		return "", fmt.Errorf("estimate_cost: create terraform client: %w", err)
	}
	plan, err := client.ShowPlan(ctx)
	if err != nil {
		return "", fmt.Errorf("estimate_cost: read plan: %w", err)
	}

	// Serialize plan changes (resource type + exact specs) for the LLM.
	planJSON, err := json.Marshal(plan.Changes)
	if err != nil {
		return "", fmt.Errorf("estimate_cost: marshal plan: %w", err)
	}

	// Check if plan changes have exact specs (After field). In dry-run mode
	// the simulated plan has no After, so the LLM can only estimate by type.
	hasSpecs := false
	for _, c := range plan.Changes {
		if len(c.After) > 0 {
			hasSpecs = true
			break
		}
	}

	skillBody := p.loadSkillBody(ctx, "huaweicloud-bss")
	progress("Loading pricing skill...", 1, 3)
	agent := openagent.NewAgent("iac-pricing",
		openagent.WithModel(p.model),
		openagent.WithTools(p.fileTools()...),
		openagent.WithMemory(p.memory),
		openagent.WithRunHooks(sloghooks.New(slog.Default()), newProgressHook()),
		openagent.WithSystemPrompts(
			serverContext,
			skillBody,
			"You are a HuaweiCloud pricing expert. "+
				"Use read/grep/ls to browse the skills/huaweicloud-bss/references/ directory "+
				"for BSS API definitions, use http_request to call the BSS pricing APIs (signing is automatic), "+
				"and use WebSearch/WebFetch as a fallback for public pricing pages. "+
				"You are given the planned resources with exact specs from terraform plan. "+
				"Query the monthly price for each resource. "+
				"Mark prices that cannot be determined as null — do NOT fabricate. "+
				"Return {\"items\": [{\"resource\": \"...\", \"spec\": \"...\", \"monthly\": price or null}], \"total_monthly\": ... or null, \"currency\": \"CNY\", \"note\": \"...\"}."),
		openagent.WithMaxTurns(8),
	)

	var userMsg string
	if hasSpecs {
		userMsg = "Query the prices for these planned resources (with exact specs):\n\n" + string(planJSON)
	} else {
		userMsg = "Query the prices for these planned resources (specs not available — estimate by resource type only):\n\n" + string(planJSON)
	}

	session := openagent.Session{ID: sessionID(deploymentID)}
	progress("Querying cloud pricing...", 2, 3)
	result, err := agent.Run(ctx, session, openagent.UserMessage(userMsg))
	if err != nil {
		return "", fmt.Errorf("estimate_cost: LLM run: %w", err)
	}

	// Parse the LLM output and add deployment_id.
	raw := extractJSON(result.FinalOutput)
	var cost struct {
		Items        []any `json:"items"`
		TotalMonthly any   `json:"total_monthly"`
		Currency     string `json:"currency"`
		Note         string `json:"note"`
	}
	if json.Unmarshal([]byte(raw), &cost) != nil {
		cost.Note = result.FinalOutput
	}
	out := map[string]any{
		"deployment_id": deploymentID,
		"items":         cost.Items,
		"total_monthly": cost.TotalMonthly,
		"currency":      cost.Currency,
		"note":          cost.Note,
	}
	data, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("estimate_cost: marshal: %w", err)
	}
	return string(data), nil
}

// Troubleshoot diagnoses a deployment error and suggests fixes.
//
// The LLM loads the troubleshoot skill, browses examples for correct
// patterns, and searches the web for error solutions.
func (p *Planner) Troubleshoot(ctx context.Context, deploymentID, errorMsg string) (string, error) {
	progress := openagent.ProgressFromContext(ctx)

	dir := filepath.Join(p.deploymentsDir, deploymentID)

	tfFiles, err := readTFFiles(dir)
	if err != nil {
		return "", fmt.Errorf("troubleshoot: read .tf: %w", err)
	}

	skillBody := p.loadSkillBody(ctx, "huaweicloud-troubleshoot")
	progress("Loading troubleshoot skill...", 0, 2)
	agent := openagent.NewAgent("iac-troubleshooter",
		openagent.WithModel(p.model),
		openagent.WithTools(p.fileTools()...),
		openagent.WithMemory(p.memory),
		openagent.WithRunHooks(sloghooks.New(slog.Default()), newProgressHook()),
		openagent.WithSystemPrompts(
			serverContext,
			skillBody,
			"You are a HuaweiCloud infrastructure troubleshooting expert. "+
				"Use read/grep/ls to find correct patterns in skills/huaweicloud-deploy/references/, "+
				"use WebSearch/WebFetch to search for error solutions. "+
				"You are given the error message and the .tf files that failed. "+
				"Diagnose the root cause and suggest specific fixes. "+
				"Return {\"diagnosis\": \"...\", \"suggestion\": \"...\", \"alternatives\": [\"...\", ...]}."),
		openagent.WithMaxTurns(8),
	)

	// user message = dynamic content (error + .tf file path + .tf files)
	// Include the deployment directory path (relative to workDir) so the
	// LLM can use read/grep/ls to inspect the .tf files itself.
	relDir, _ := filepath.Rel(p.workDir, dir)
	userMsg := fmt.Sprintf("A deployment on %s failed with this error:\n\n%s\n\n"+
		"The terraform files are in directory: %s\n\n%s\n\n"+
		"You can use read/grep/ls with the path above to inspect the files. "+
		"Diagnose the error and suggest fixes.",
		p.cloud.Name(), errorMsg, relDir, tfFiles)

	session := openagent.Session{ID: sessionID(deploymentID)}
	progress("Diagnosing error...", 1, 2)
	result, err := agent.Run(ctx, session, openagent.UserMessage(userMsg))
	if err != nil {
		return "", fmt.Errorf("troubleshoot: LLM run: %w", err)
	}

	// Parse the LLM output into a structured diagnosis.
	raw := extractJSON(result.FinalOutput)
	var diag struct {
		Diagnosis    string   `json:"diagnosis"`
		Suggestion   string   `json:"suggestion"`
		Alternatives []string `json:"alternatives"`
	}
	if json.Unmarshal([]byte(raw), &diag) != nil || (diag.Diagnosis == "" && diag.Suggestion == "" && len(diag.Alternatives) == 0) {
		diag.Diagnosis = result.FinalOutput
	}
	data, err := json.Marshal(diag)
	if err != nil {
		return "", fmt.Errorf("troubleshoot: marshal: %w", err)
	}
	return string(data), nil
}

// QueryCloud answers read-only queries about existing cloud resources, specs,
// bills, or quotas. Unlike the other 4 agents, this one uses dynamic skill
// loading (WithSkillLoader) — the LLM sees the skill catalog and calls
// load_skill to load the relevant cloud-service skill on demand.
func (p *Planner) QueryCloud(ctx context.Context, query string) (string, error) {
	progress := openagent.ProgressFromContext(ctx)

	progress("Setting up query agent...", 0, 2)
	agent := openagent.NewAgent("iac-queryer",
		openagent.WithModel(p.model),
		openagent.WithTools(p.fileTools()...),
		openagent.WithMemory(p.memory),
		openagent.WithRunHooks(sloghooks.New(slog.Default()), newProgressHook()),
		openagent.WithSkillLoader(p.loader),
		openagent.WithSystemPrompts(
			serverContext,
			"You are a HuaweiCloud cloud query expert. "+
				"Use load_skill to load the relevant skill for the cloud service being queried "+
				"(e.g. load_skill(\"huaweicloud-ecs\") for ECS instances/flavors, "+
				"load_skill(\"huaweicloud-vpc\") for VPCs/subnets/security groups, "+
				"load_skill(\"huaweicloud-bss\") for billing/pricing/orders). "+
				"Then use http_request to call the API with the correct endpoint and parameters. "+
				"CRITICAL: Only call read-only APIs (List/Show/Get). NEVER call Create/Update/Delete APIs — "+
				"this tool is for querying existing resources only, not for creating or modifying them. "+
				"Return {\"results\": [...], \"note\": \"...\"}."),
		openagent.WithMaxTurns(10),
	)

	session := openagent.Session{ID: "query"}
	progress("Querying cloud resources...", 1, 2)
	result, err := agent.Run(ctx, session, openagent.UserMessage(query))
	if err != nil {
		return "", fmt.Errorf("query_cloud: LLM run: %w", err)
	}

	// Parse the LLM output. If it's already valid JSON, pass through.
	raw := extractJSON(result.FinalOutput)
	var qc struct {
		Results []any  `json:"results"`
		Note    string `json:"note"`
	}
	if json.Unmarshal([]byte(raw), &qc) != nil {
		qc.Note = result.FinalOutput
	}
	out := map[string]any{
		"results": qc.Results,
		"note":    qc.Note,
	}
	data, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("query_cloud: marshal: %w", err)
	}
	return string(data), nil
}

// loadSkillBody statically loads a skill's SKILL.md body by name.
// The body is injected directly into the agent's system prompt instead of
// relying on the LLM to call load_skill at runtime — this is deterministic,
// saves a tool-call round-trip, and avoids injecting the skill catalog +
// load_skill/reload_skills tool definitions.
func (p *Planner) loadSkillBody(ctx context.Context, name string) string {
	skills, err := p.loader.Discover(ctx)
	if err != nil {
		return ""
	}
	for _, s := range skills {
		if s.Name == name {
			body, err := p.loader.Load(ctx, s)
			if err != nil {
				return ""
			}
			return body
		}
	}
	return ""
}

// fileTools returns the standard file + web tools for all LLM agents.
// read/grep/ls operate with workDir = cloud home so the LLM can browse
// both skills/ (references, guides) and deployments/ (.tf files).
// If the cloud provider exposes an http_request tool (e.g. huaweicloud
// with SDK-HMAC-SHA256 signing), it is included for calling cloud APIs.
func (p *Planner) fileTools() []openagent.Tool {
	tools := []openagent.Tool{
		opentool.NewReadFile(p.workDir),
		opentool.NewGrep(p.workDir),
		opentool.NewListDir(p.workDir),
		opentool.NewWebSearch(),
		opentool.NewWebFetch(),
	}
	if ht, ok := p.cloud.(interface{ HTTPRequest() openagent.Tool }); ok {
		tools = append(tools, ht.HTTPRequest())
	}
	return tools
}

// readTFFiles reads all .tf files in a directory and returns them as a string.
func readTFFiles(dir string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.tf"))
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		b.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", filepath.Base(f), data))
	}
	return b.String(), nil
}

// backupTFFiles reads all .tf and .tfvars files in a directory into a map
// of filename → content. Used by UpdateDeployment to restore on failure.
func backupTFFiles(dir string) (map[string]string, error) {
	patterns := []string{filepath.Join(dir, "*.tf"), filepath.Join(dir, "*.tfvars")}
	backup := make(map[string]string)
	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			backup[filepath.Base(f)] = string(data)
		}
	}
	return backup, nil
}

// restoreTFFiles writes backed-up files back to a directory.
func restoreTFFiles(dir string, backup map[string]string) {
	for name, content := range backup {
		os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	}
}

// extractJSON finds the first JSON object in a string (LLM output may have
// surrounding text or markdown fences).
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end <= start {
		return s
	}
	return s[start : end+1]
}

// marshalResult marshals a planResult to JSON string.
func marshalResult(r planResult) (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(data), nil
}

// deploymentID allocates a unique deployment ID by atomically creating its
// directory. Race-safe: two concurrent callers cannot get the same ID.
func deploymentID(deploymentsDir string) (string, string, error) {
	entries, _ := os.ReadDir(deploymentsDir)
	maxNum := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "d-") {
			var num int
			fmt.Sscanf(name, "d-%d", &num)
			if num > maxNum {
				maxNum = num
			}
		}
	}
	for n := maxNum + 1; n < maxNum+1000; n++ {
		id := fmt.Sprintf("d-%03d", n)
		dir := filepath.Join(deploymentsDir, id)
		if err := os.Mkdir(dir, 0755); err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", "", fmt.Errorf("create deployment dir: %w", err)
		}
		return id, dir, nil
	}
	return "", "", fmt.Errorf("no free deployment ID found under %s", deploymentsDir)
}
