---
name: huaweicloud-aos
description: HuaweiCloud AOS API guide. 75 APIs covering 资源编排-Hook, 资源编排-执行计划, 资源编排-模块管理, 资源编排-模板分析, 资源编排-模板管理.
---

# HuaweiCloud AOS API Guide

75 APIs. Tags: 资源编排-Hook, 资源编排-执行计划, 资源编排-模块管理, 资源编排-模板分析, 资源编排-模板管理, 资源编排-自定义provider, 资源编排-资源栈, 资源编排-资源栈集

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ApplyExecutionPlan` | POST | `/v1/{project_id}/stacks/{stack_name}/execution-plans/{execution_plan_name}` | 执行执行计划 |
| `ContinueDeployStack` | POST | `/v1/{project_id}/stacks/{stack_name}/continuations` | 继续部署资源栈 |
| `ContinueRollbackStack` | POST | `/v1/{project_id}/stacks/{stack_name}/rollbacks` | 继续回滚资源栈 |
| `CreateExecutionPlan` | POST | `/v1/{project_id}/stacks/{stack_name}/execution-plans` | 创建执行计划 |
| `CreatePrivateHook` | POST | `/v1/private-hooks` | 创建私有hook |
| `CreatePrivateHookVersion` | POST | `/v1/private-hooks/{hook_name}/versions` | 创建私有hook版本 |
| `CreatePrivateModule` | POST | `/v1/private-modules` | 创建私有模块 |
| `CreatePrivateModuleVersion` | POST | `/v1/private-modules/{module_name}/versions` | 创建私有模块版本 |
| `CreatePrivateProvider` | POST | `/v1/private-providers` | 创建私有provider |
| `CreatePrivateProviderVersion` | POST | `/v1/private-providers/{provider_name}/versions` | 创建私有provider版本 |
| `CreateStack` | POST | `/v1/{project_id}/stacks` | 创建资源栈 |
| `CreateStackInstance` | POST | `/v1/stack-sets/{stack_set_name}/stack-instances` | 创建资源栈实例 |
| `CreateStackSet` | POST | `/v1/stack-sets` | 创建资源栈集 |
| `CreateTemplate` | POST | `/v1/{project_id}/templates` | 创建模板 |
| `CreateTemplateVersion` | POST | `/v1/{project_id}/templates/{template_name}/versions` | 创建模板版本 |
| `DeleteExecutionPlan` | DELETE | `/v1/{project_id}/stacks/{stack_name}/execution-plans/{execution_plan_name}` | 删除执行计划 |
| `DeletePrivateHook` | DELETE | `/v1/private-hooks/{hook_name}` | 删除私有hook |
| `DeletePrivateHookVersion` | DELETE | `/v1/private-hooks/{hook_name}/versions/{hook_version}` | 删除私有hook版本 |
| `DeletePrivateModule` | DELETE | `/v1/private-modules/{module_name}` | 删除私有模块 |
| `DeletePrivateModuleVersion` | DELETE | `/v1/private-modules/{module_name}/versions/{module_version}` | 删除私有模块版本 |
| `DeletePrivateProvider` | DELETE | `/v1/private-providers/{provider_name}` | 删除私有provider |
| `DeletePrivateProviderVersion` | DELETE | `/v1/private-providers/{provider_name}/versions/{provider_version}` | 删除私有provider版本 |
| `DeleteStack` | DELETE | `/v1/{project_id}/stacks/{stack_name}` | 删除资源栈 |
| `DeleteStackEnhanced` | POST | `/v1/{project_id}/stacks/{stack_name}/deletion` | 条件删除资源栈 |
| `DeleteStackInstance` | POST | `/v1/stack-sets/{stack_set_name}/stack-instances/deletion` | 删除资源栈实例 |
| `DeleteStackInstanceDeprecated` | DELETE | `/v1/stack-sets/{stack_set_name}/stack-instances` | 删除资源栈实例-已废弃 |
| `DeleteStackSet` | DELETE | `/v1/stack-sets/{stack_set_name}` | 删除资源栈集 |
| `DeleteTemplate` | DELETE | `/v1/{project_id}/templates/{template_name}` | 删除模板 |
| `DeleteTemplateVersion` | DELETE | `/v1/{project_id}/templates/{template_name}/versions/{version_id}` | 删除模板版本 |
| `DeployStack` | POST | `/v1/{project_id}/stacks/{stack_name}/deployments` | 部署资源栈 |

... and 45 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
