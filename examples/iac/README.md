# openagent IaC Agent

Production-grade Infrastructure as Code agent built on openagent-go. Automates the full cloud deployment lifecycle:

**Intent → Application Profile → Architecture Design → Module Selection → Terraform Plan → Apply → Monitor**

## Quick Start

```bash
export OPENAGENT_API_KEY=sk-xxx
export OPENAGENT_MODEL=deepseek-v3
export OPENAGENT_BASE_URL=https://api.deepseek.com/v1

go run ./examples/iac/
```

Service starts on `:8080`. All Terraform commands run in **dry-run mode** by default — no real cloud resources are created.

## API Overview

```
POST /plan/sessions                — create plan session
POST /plan/sessions/{id}/generate  — generate DAG from goal (SSE streaming)
GET  /plan/sessions/{id}/plan      — get current plan (DAG JSON)
POST /plan/sessions/{id}/execute   — execute plan (SSE streaming)
GET  /plan/sessions/{id}/events    — SSE event stream (live pipeline panel)
POST /plan/sessions/{id}/approve   — approve/reject tool calls
POST /plan/sessions/{id}/replan    — replan with feedback (SSE)
GET  /iac/info                     — agent catalog + module list
GET  /health                       — health check
```

## 6-Stage Pipeline

| Stage | Agent | What it does |
|-------|-------|-------------|
| ① Intent Parser | `intent_parser` | GitHub URL / natural language → structured `ApplicationProfile` JSON |
| ② Architecture | `architect` | 3 options (A/B/C) with resource lists + monthly pricing |
| ③ Module Planner | `module_planner` | Selects Terraform modules, fills template variables, writes `.tf` files |
| ④ Plan Review | `reviewer` | `terraform init && plan` → human-readable change summary |
| ⑤ Apply | `applier` | `terraform apply` (requires approval), streams output |
| ⑥ Monitor | `monitor` | Health checks + optimization recommendations |

## Example Usage

```bash
# 1. Create a plan session
curl -s -X POST localhost:8080/plan/sessions \
  -d '{"goal":"Deploy a WordPress blog with CDN and HTTPS"}' | jq .id

# 2. Generate the DAG (streams LLM thinking)
curl -N -X POST localhost:8080/plan/sessions/{id}/generate \
  -d '{"goal":"Deploy a WordPress blog with CDN and HTTPS"}'

# 3. View the generated plan
curl -s localhost:8080/plan/sessions/{id}/plan | jq .

# 4. Execute (streams step-by-step progress)
curl -N -X POST localhost:8080/plan/sessions/{id}/execute

# 5. Monitor events (separate terminal)
curl -N localhost:8080/plan/sessions/{id}/events
```

## Real Deployments

Set `DRY_RUN=false` and provide HuaweiCloud credentials:

```bash
export DRY_RUN=false
export HW_ACCESS_KEY=xxx
export HW_SECRET_KEY=xxx
export HW_REGION=cn-north-4
go run ./examples/iac/
```

The agent will execute real `terraform init/plan/apply` against HuaweiCloud.

## Architecture

```
examples/iac/
  main.go           # REST server + middleware + wiring
  agents.go         # 6 agent definitions + schema types + skill loader
  tools/
    terraform.go    # TerraformTool (init/plan/apply/output/destroy) + dry-run simulation
  templates/        # Embedded Terraform module templates (Go embed)
    ecs.tf.tmpl     # ECS (VM) template
    rds.tf.tmpl     # RDS (database) template
    obs.tf.tmpl     # OBS (object storage) template
    cdn.tf.tmpl     # CDN template
    provider.tf.tmpl # HuaweiCloud provider config
  skills/           # Embedded skill knowledge (SKILL.md)
    huaweicloud/     # Provider config, regions, naming conventions
    pricing/         # Resource pricing reference (RMB)
    deployment-patterns/ # Web App / Static / HA patterns with cost estimates
    terraform-modules/   # Module catalog with parameters
```

## Design Decisions

- **Zero external deps at runtime** — templates and skills embedded via `//go:embed`
- **Dry-run by default** — `terraform plan` is safe; `terraform apply` requires opt-in
- **Self-approving read operations** — `terraform_init`, `terraform_plan`, `terraform_output` auto-approve
- **Approval gates on write operations** — `terraform_apply` and `terraform_destroy` require explicit approval
- **Uses `rest.OrchestrateHandler`** — same SSE-streaming API as the core framework's plan endpoint
- **LLMPlanner generates DAG** — the model decomposes user goals into ordered steps based on agent descriptions
