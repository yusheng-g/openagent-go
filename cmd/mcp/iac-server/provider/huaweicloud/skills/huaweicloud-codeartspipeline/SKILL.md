---
name: huaweicloud-codeartspipeline
description: HuaweiCloud CodeArtsPipeline API guide. 87 APIs covering gitcode专用接口, 扩展插件管理, 模板管理, 流水线分组管理, 流水线模板管理--新.
---

# HuaweiCloud CodeArtsPipeline API Guide

87 APIs. Tags: gitcode专用接口, 扩展插件管理, 模板管理, 流水线分组管理, 流水线模板管理--新, 流水线管理, 流水线管理--新, 租户级策略管理, 规则管理, 项目级策略管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptManualReview` | POST | `/v5/{project_id}/api/pipelines/{pipeline_id}/pipeline-runs/{pipeline_run_id}/jobs/{job_run_id}/steps/{step_run_id}/pass` | 通过人工审核 |
| `BatchMovePipelineToGroup` | POST | `/v5/{project_id}/api/pipeline-group/pipeline/move` | 批量把流水线移动到分组下 |
| `BatchShowPipelinesLatestStatus` | POST | `/v5/{project_id}/api/pipelines/status` | 批量获取流水线状态 |
| `BatchShowPipelinesStatus` | GET | `/v3/pipelines/status` | 批量获取流水线状态 |
| `CreateBasicPlugin` | POST | `/v3/{domain_id}/extension/info/add` | 创建基础插件 |
| `CreatePipelineByTemplate` | POST | `/v3/templates/task` | 基于模板快速创建流水线及流水线内任务 |
| `CreatePipelineByTemplateId` | POST | `/v5/{project_id}/api/pipeline-templates/{template_id}/create-pipeline` | 基于模板创建流水线 |
| `CreatePipelineGroup` | POST | `/v5/{project_id}/api/pipeline-group/create` | 新建流水线分组 |
| `CreatePipelineNew` | POST | `/v5/{project_id}/api/pipelines` | 创建流水线 |
| `CreatePipelineTemplate` | POST | `/v5/{tenant_id}/api/pipeline-templates` | 创建流水线模板 |
| `CreatePluginDraft` | POST | `/v1/{domain_id}/agent-plugin/create-draft` | 创建插件草稿版本 |
| `CreatePluginVersion` | POST | `/v1/{domain_id}/agent-plugin/create` | 创建插件版本 |
| `CreatePublisher` | POST | `/v1/{domain_id}/publisher/create` | 创建发布商 |
| `CreateRule` | POST | `/v2/{domain_id}/rules/create` | 创建规则 |
| `CreateStrategy` | POST | `/v2/{domain_id}/tenant/rule-sets/create` | 创建策略 |
| `DeleteActionsRunPipeline` | DELETE | `/v6/{domain_id}/api/pac/pipelines/actions/{pipeline_id}/{pipeline_run_id}` | 删除gitcode流水线运行详情 |
| `DeleteBasicPlugin` | DELETE | `/v3/{domain_id}/extension/info/delete` | 删除基础插件 |
| `DeletePipeline` | DELETE | `/v5/{project_id}/api/pipelines/{pipeline_id}` | 删除流水线 |
| `DeletePipelineGroup` | DELETE | `/v5/{project_id}/api/pipeline-group/delete` | 删除流水线分组 |
| `DeletePipelineTemplate` | DELETE | `/v5/{tenant_id}/api/pipeline-templates/{template_id}` | 删除流水线模板 |
| `DeletePluginDraft` | DELETE | `/v1/{domain_id}/agent-plugin/delete-draft` | 删除插件草稿 |
| `DeletePublisher` | DELETE | `/v1/{domain_id}/publisher/delete` | 删除发布商 |
| `DeleteRule` | DELETE | `/v2/{domain_id}/rules/{rule_id}/delete` | 删除规则 |
| `DeleteStrategy` | DELETE | `/v2/{domain_id}/tenant/rule-sets/{rule_set_id}/delete` | 删除策略 |
| `ListActionsPipelineRuns` | POST | `/v6/{domain_id}/api/pac/pipelines/actions` | 查询gitcode流水线运行记录 |
| `ListActionsPipelineRunsByRunIds` | POST | `/v6/{domain_id}/api/pac/pipelines/actions/list` | 查询gitcode流水线action列表 |
| `ListAvailablePublisher` | GET | `/v1/{domain_id}/publisher/optional-publisher` | 查询可用发布商 |
| `ListBasePlugins` | GET | `/v1/{domain_id}/relation/plugin/single` | 查询基础插件列表 |
| `ListBasePluginsNewPost` | POST | `/v1/{domain_id}/relation/plugins` | 分页查询可选插件列表 |
| `ListPipelineRuns` | POST | `/v5/{project_id}/api/pipelines/{pipeline_id}/pipeline-runs/list` | 获取流水线执行记录 |

... and 57 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
