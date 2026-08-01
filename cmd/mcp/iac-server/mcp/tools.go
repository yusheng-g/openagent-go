// Package mcp defines the deployment tools exposed by iac-server over MCP.
//
// Tools are split into two groups:
//   - LLM tools (propose_architecture, specify_resources, generate_plan,
//     estimate_cost, troubleshoot_deployment, query_cloud): delegate to agent.Planner
//   - Execution tools (apply, destroy, get_status, list): call iac.Client
//     directly
//
// All tools return JSON strings. Server-side execution is unconditional —
// approval is the client's concern, not the server's.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/cmd/mcp/iac-server/agent"
	"github.com/yusheng-g/openagent-go/cmd/mcp/iac-server/provider"
	"github.com/yusheng-g/openagent-go/iac"
)

// Config holds shared dependencies for all tools.
type Config struct {
	Planner         *agent.Planner
	Cloud           provider.CloudProvider
	DeploymentsDir  string   // root dir for deployment workspaces
	DryRun          bool     // pass to iac.Config.DryRun
	BinaryMirrors   []string // terraform binary download mirrors
	ProviderMirrors []string // provider download mirrors (URLs or local paths)
}

// NewTools builds the 11 tools exposed by iac-server.
func NewTools(cfg Config) []openagent.Tool {
	return []openagent.Tool{
		&proposeArchitectureTool{cfg: cfg},
		&specifyResourcesTool{cfg: cfg},
		&generatePlanTool{cfg: cfg},
		&estimateCostTool{cfg: cfg},
		&troubleshootDeploymentTool{cfg: cfg},
		&applyDeploymentTool{cfg: cfg},
		&destroyDeploymentTool{cfg: cfg},
		&getDeploymentStatusTool{cfg: cfg},
		&listDeploymentsTool{cfg: cfg},
		&queryCloudTool{cfg: cfg},
		&updateDeploymentTool{cfg: cfg},
	}
}

// workDir returns the workspace path for a deployment ID.
func workDir(deploymentsDir, deploymentID string) string {
	return filepath.Join(deploymentsDir, deploymentID)
}

// validDeploymentID reports whether id is a safe deployment identifier:
// non-empty, no path separators, no parent-dir components. This prevents
// deployment_id values like "../etc" from escaping deploymentsDir.
func validDeploymentID(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	if strings.ContainsAny(id, `/\`) {
		return false
	}
	// Reject any segment that is ".." after cleaning.
	cleaned := filepath.Clean(id)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

// iacConfig builds an iac.Config with cloud credentials and mirror settings.
func iacConfig(cloud provider.CloudProvider, dryRun bool, binaryMirrors, providerMirrors []string) iac.Config {
	return iac.Config{
		Env:             cloud.Env(),
		DryRun:          dryRun,
		BinaryMirrors:   binaryMirrors,
		ProviderMirrors: providerMirrors,
	}
}

// ── propose_architecture ──

type proposeArchitectureTool struct{ cfg Config }

func (t *proposeArchitectureTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "propose_architecture",
		Description: "Step 1 of deployment: Analyze a deployment request and recommend a cloud architecture. Returns architecture name, required services, reasoning, and a deployment_id. Does NOT write .tf files. The user should confirm the architecture before calling specify_resources.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"request": {
					"type": "string",
					"description": "Free-text deployment request, e.g. \"deploy a WordPress site to cn-east-3, single instance, budget 100/month\""
				}
			},
			"required": ["request"]
		}`),
	}
}

func (t *proposeArchitectureTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Request string `json:"request"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("propose_architecture: %w", err)
	}
	if params.Request == "" {
		return "", fmt.Errorf("propose_architecture: request is required")
	}
	return t.cfg.Planner.ProposeArchitecture(ctx, params.Request)
}

// ── specify_resources ──

type specifyResourcesTool struct{ cfg Config }

func (t *specifyResourcesTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "specify_resources",
		Description: "Step 2 of deployment: Determine concrete resource specs (flavor, image, disk, CIDR, etc.) for a proposed architecture. Reads the architecture from the prior propose_architecture call. Optional adjustments let the user modify specs. The user should confirm the resources before calling generate_plan.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"deployment_id": {
					"type": "string",
					"description": "Deployment ID from propose_architecture"
				},
				"adjustments": {
					"type": "string",
					"description": "Optional free-text adjustments, e.g. \"use s6.xlarge.2 instead\" or \"add a 100GB data disk\""
				}
			},
			"required": ["deployment_id"]
		}`),
	}
}

func (t *specifyResourcesTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		DeploymentID string `json:"deployment_id"`
		Adjustments  string `json:"adjustments"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("specify_resources: %w", err)
	}
	if params.DeploymentID == "" {
		return "", fmt.Errorf("specify_resources: deployment_id is required")
	}
	if !validDeploymentID(params.DeploymentID) {
		return "", fmt.Errorf("specify_resources: invalid deployment_id %q", params.DeploymentID)
	}
	return t.cfg.Planner.SpecifyResources(ctx, params.DeploymentID, params.Adjustments)
}

// ── generate_plan ──

type generatePlanTool struct{ cfg Config }

func (t *generatePlanTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "generate_plan",
		Description: "Step 3 of deployment: Write .tf files based on the confirmed architecture and resource specs, then run terraform init + plan. Returns the .tf files and a plan preview. The user should review the plan before calling estimate_cost.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"deployment_id": {
					"type": "string",
					"description": "Deployment ID from propose_architecture"
				}
			},
			"required": ["deployment_id"]
		}`),
	}
}

func (t *generatePlanTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		DeploymentID string `json:"deployment_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("generate_plan: %w", err)
	}
	if params.DeploymentID == "" {
		return "", fmt.Errorf("generate_plan: deployment_id is required")
	}
	if !validDeploymentID(params.DeploymentID) {
		return "", fmt.Errorf("generate_plan: invalid deployment_id %q", params.DeploymentID)
	}
	return t.cfg.Planner.GeneratePlan(ctx, params.DeploymentID)
}

// ── update_deployment ──

type updateDeploymentTool struct{ cfg Config }

func (t *updateDeploymentTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "update_deployment",
		Description: "Modify an existing deployment. Re-runs specify_resources (with user adjustments) and generate_plan. Use this when the user wants to adjust an existing deployment (e.g. \"change ECS flavor to s6.xlarge.2\"). Returns the updated plan with the same deployment_id. After updating, call estimate_cost again to see the new pricing before apply_deployment.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"deployment_id": {
					"type": "string",
					"description": "Deployment ID to update"
				},
				"change_request": {
					"type": "string",
					"description": "Free-text change request, e.g. \"change ECS flavor to s6.xlarge.2\" or \"rename vpc.test to vpc.main\""
				}
			},
			"required": ["deployment_id", "change_request"]
		}`),
	}
}

func (t *updateDeploymentTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		DeploymentID  string `json:"deployment_id"`
		ChangeRequest string `json:"change_request"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("update_deployment: %w", err)
	}
	if params.DeploymentID == "" || params.ChangeRequest == "" {
		return "", fmt.Errorf("update_deployment: deployment_id and change_request are required")
	}
	if !validDeploymentID(params.DeploymentID) {
		return "", fmt.Errorf("update_deployment: invalid deployment_id %q", params.DeploymentID)
	}
	return t.cfg.Planner.UpdateDeployment(ctx, params.DeploymentID, params.ChangeRequest)
}

// ── estimate_cost ──

type estimateCostTool struct{ cfg Config }

func (t *estimateCostTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "estimate_cost",
		Description: "Step 4 of deployment: Estimate the monthly cost of a PLANNED deployment (resources not yet created). MUST be called after generate_plan and before apply_deployment. This forecasts future costs based on the terraform plan — it does NOT query past billing. For existing bills/costs, use query_cloud.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"deployment_id": {
					"type": "string",
					"description": "Deployment ID from plan_deployment"
				}
			},
			"required": ["deployment_id"]
		}`),
	}
}

func (t *estimateCostTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		DeploymentID string `json:"deployment_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("estimate_cost: %w", err)
	}
	if params.DeploymentID == "" {
		return "", fmt.Errorf("estimate_cost: deployment_id is required")
	}
	if !validDeploymentID(params.DeploymentID) {
		return "", fmt.Errorf("estimate_cost: invalid deployment_id %q", params.DeploymentID)
	}
	return t.cfg.Planner.EstimateCost(ctx, params.DeploymentID)
}

// ── troubleshoot_deployment ──

type troubleshootDeploymentTool struct{ cfg Config }

func (t *troubleshootDeploymentTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "troubleshoot_deployment",
		Description: "Diagnose a deployment error and suggest fixes. Reads the .tf files and error message, researches solutions via examples and web search, and returns a diagnosis with recommended actions.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"deployment_id": {
					"type": "string",
					"description": "Deployment ID to troubleshoot"
				},
				"error": {
					"type": "string",
					"description": "The error message from the failed operation"
				}
			},
			"required": ["deployment_id", "error"]
		}`),
	}
}

func (t *troubleshootDeploymentTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		DeploymentID string `json:"deployment_id"`
		Error        string `json:"error"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("troubleshoot_deployment: %w", err)
	}
	if params.DeploymentID == "" || params.Error == "" {
		return "", fmt.Errorf("troubleshoot_deployment: deployment_id and error are required")
	}
	if !validDeploymentID(params.DeploymentID) {
		return "", fmt.Errorf("troubleshoot_deployment: invalid deployment_id %q", params.DeploymentID)
	}
	return t.cfg.Planner.Troubleshoot(ctx, params.DeploymentID, params.Error)
}

// ── apply_deployment ──

type applyDeploymentTool struct{ cfg Config }

func (t *applyDeploymentTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "apply_deployment",
		Description: "Step 5 of deployment: Apply a saved terraform plan. This creates/modifies real cloud resources. The deployment must have been planned first (generate_plan succeeded). Call estimate_cost first so the user sees pricing.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"deployment_id": {
					"type": "string",
					"description": "Deployment ID to apply"
				}
			},
			"required": ["deployment_id"]
		}`),
	}
}

func (t *applyDeploymentTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		DeploymentID string `json:"deployment_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("apply_deployment: %w", err)
	}
	if params.DeploymentID == "" {
		return "", fmt.Errorf("apply_deployment: deployment_id is required")
	}
	if !validDeploymentID(params.DeploymentID) {
		return "", fmt.Errorf("apply_deployment: invalid deployment_id %q", params.DeploymentID)
	}

	dir := workDir(t.cfg.DeploymentsDir, params.DeploymentID)
	client, err := iac.NewClient(ctx, dir, iacConfig(t.cfg.Cloud, t.cfg.DryRun, t.cfg.BinaryMirrors, t.cfg.ProviderMirrors))
	if err != nil {
		return "", fmt.Errorf("apply_deployment: %w", err)
	}

	result, err := client.Apply(ctx)
	if err != nil {
		return "", fmt.Errorf("apply_deployment: %w", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("apply_deployment: marshal: %w", err)
	}
	return string(data), nil
}

// ── destroy_deployment ──

type destroyDeploymentTool struct{ cfg Config }

func (t *destroyDeploymentTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "destroy_deployment",
		Description: "Destroy all resources in a deployment. This permanently deletes cloud resources. Use with caution.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"deployment_id": {
					"type": "string",
					"description": "Deployment ID to destroy"
				}
			},
			"required": ["deployment_id"]
		}`),
	}
}

func (t *destroyDeploymentTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		DeploymentID string `json:"deployment_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("destroy_deployment: %w", err)
	}
	if params.DeploymentID == "" {
		return "", fmt.Errorf("destroy_deployment: deployment_id is required")
	}
	if !validDeploymentID(params.DeploymentID) {
		return "", fmt.Errorf("destroy_deployment: invalid deployment_id %q", params.DeploymentID)
	}

	dir := workDir(t.cfg.DeploymentsDir, params.DeploymentID)
	client, err := iac.NewClient(ctx, dir, iacConfig(t.cfg.Cloud, t.cfg.DryRun, t.cfg.BinaryMirrors, t.cfg.ProviderMirrors))
	if err != nil {
		return "", fmt.Errorf("destroy_deployment: %w", err)
	}

	resources, err := client.Destroy(ctx)
	if err != nil {
		return "", fmt.Errorf("destroy_deployment: %w", err)
	}

	result := map[string]any{
		"destroyed": true,
		"resources": resources,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("destroy_deployment: marshal: %w", err)
	}
	return string(data), nil
}

// ── get_deployment_status ──

type getDeploymentStatusTool struct{ cfg Config }

func (t *getDeploymentStatusTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "get_deployment_status",
		Description: "Read the terraform state for a deployment and return a status summary. Does not call terraform binary — reads the state file directly.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"deployment_id": {
					"type": "string",
					"description": "Deployment ID to check"
				}
			},
			"required": ["deployment_id"]
		}`),
	}
}

func (t *getDeploymentStatusTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		DeploymentID string `json:"deployment_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("get_deployment_status: %w", err)
	}
	if params.DeploymentID == "" {
		return "", fmt.Errorf("get_deployment_status: deployment_id is required")
	}
	if !validDeploymentID(params.DeploymentID) {
		return "", fmt.Errorf("get_deployment_status: invalid deployment_id %q", params.DeploymentID)
	}

	dir := workDir(t.cfg.DeploymentsDir, params.DeploymentID)
	statePath := filepath.Join(dir, "terraform.tfstate")

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("get_deployment_status: no state file for deployment %s — has it been planned/applied?", params.DeploymentID)
		}
		return "", fmt.Errorf("get_deployment_status: %w", err)
	}

	// Parse the state file to extract a summary.
	var state struct {
		Resources []struct {
			Address string `json:"address"`
			Type    string `json:"type"`
			Name    string `json:"name"`
		} `json:"resources"`
		Outputs map[string]struct {
			Value any `json:"value"`
			Type  any `json:"type"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return "", fmt.Errorf("get_deployment_status: parse state: %w", err)
	}

	summary := map[string]any{
		"deployment_id":  params.DeploymentID,
		"resource_count": len(state.Resources),
		"resources":      state.Resources,
		"outputs":        state.Outputs,
	}
	result, err := json.Marshal(summary)
	if err != nil {
		return "", fmt.Errorf("get_deployment_status: marshal: %w", err)
	}
	return string(result), nil
}

// ── query_cloud ──

type queryCloudTool struct{ cfg Config }

func (t *queryCloudTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "query_cloud",
		Description: "Query EXISTING cloud resources, specs, bills, costs, or quotas. Use this for any read-only query about the current cloud account state — e.g. \"list all ECS instances\", \"what specs does s6.large.2 have\", \"how much did I spend this month\", \"show my bills for 2025-07\". This queries real cloud APIs for already-existing resources and past billing data. Does NOT modify any resources. For estimating FUTURE costs of a planned deployment, use estimate_cost.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Natural language query, e.g. \"list all ECS instances in cn-east-3\" or \"how much did I spend this month\""
				}
			},
			"required": ["query"]
		}`),
	}
}

func (t *queryCloudTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("query_cloud: %w", err)
	}
	if params.Query == "" {
		return "", fmt.Errorf("query_cloud: query is required")
	}
	return t.cfg.Planner.QueryCloud(ctx, params.Query)
}

// ── list_deployments ──

type listDeploymentsTool struct{ cfg Config }

func (t *listDeploymentsTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name:        "list_deployments",
		Description: "List all deployments by scanning the deployments directory. Returns deployment IDs and whether each has a state file.",
		Parameters:  json.RawMessage(`{"type": "object", "properties": {}}`),
	}
}

func (t *listDeploymentsTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) {
	entries, err := os.ReadDir(t.cfg.DeploymentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "[]", nil // no deployments yet
		}
		return "", fmt.Errorf("list_deployments: %w", err)
	}

	type deployment struct {
		ID       string `json:"id"`
		HasState bool   `json:"has_state"`
		HasPlan  bool   `json:"has_plan"`
	}

	var deployments []deployment
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(t.cfg.DeploymentsDir, entry.Name())
		_, stateErr := os.Stat(filepath.Join(dir, "terraform.tfstate"))
		_, planErr := os.Stat(filepath.Join(dir, "tfplan"))
		deployments = append(deployments, deployment{
			ID:       entry.Name(),
			HasState: stateErr == nil,
			HasPlan:  planErr == nil,
		})
	}

	// Sort by ID for stable output.
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].ID < deployments[j].ID
	})

	data, err := json.Marshal(deployments)
	if err != nil {
		return "", fmt.Errorf("list_deployments: marshal: %w", err)
	}
	return string(data), nil
}
