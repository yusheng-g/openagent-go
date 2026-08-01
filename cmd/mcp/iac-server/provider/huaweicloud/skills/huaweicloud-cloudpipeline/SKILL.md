---
name: huaweicloud-cloudpipeline
description: HuaweiCloud CloudPipeline API guide. 21 APIs covering 模板管理, 流水线模板管理--新, 流水线管理, 流水线管理--新.
---

# HuaweiCloud CloudPipeline API Guide

21 APIs. Tags: 模板管理, 流水线模板管理--新, 流水线管理, 流水线管理--新

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchShowPipelinesLatestStatus` | POST | `/v5/{project_id}/api/pipelines/status` | 批量获取流水线状态 |
| `BatchShowPipelinesStatus` | GET | `/v3/pipelines/status` | 批量获取流水线状态 |
| `CreatePipelineByTemplate` | POST | `/v3/templates/task` | 基于模板快速创建流水线及流水线内任务 |
| `CreatePipelineByTemplateId` | POST | `/v5/{project_id}/api/pipeline-templates/{template_id}/create-pipeline` | 基于模板创建流水线 |
| `DeletePipeline` | DELETE | `/v5/{project_id}/api/pipelines/{pipeline_id}` | 删除流水线 |
| `ListPipelineRuns` | POST | `/v5/{project_id}/api/pipelines/{pipeline_id}/pipeline-runs/list` | 获取流水线执行记录 |
| `ListPipelines` | POST | `/v5/{project_id}/api/pipelines/list` | 获取流水线列表/获取项目下流水线执行状况 |
| `ListPipelineSimpleInfo` | POST | `/v3/pipelines/list` | 获取流水线列表接口 |
| `ListPipelineTemplates` | POST | `/v5/{tenant_id}/api/pipeline-templates/list` | 查询模板列表 |
| `ListPipleineBuildResult` | GET | `/v3/pipelines/build-result` | 获取项目下流水线执行状况 |
| `ListTemplates` | GET | `/v3/templates` | 查询模板列表 |
| `RemovePipeline` | DELETE | `/v3/pipelines/{pipeline_id}` | 删除流水线 |
| `RunPipeline` | POST | `/v5/{project_id}/api/pipelines/{pipeline_id}/run` | 启动流水线 |
| `ShowInstanceStatus` | GET | `/v3/templates/{task_id}/status` | 检查流水线创建状态 |
| `ShowPipelineRunDetail` | GET | `/v5/{project_id}/api/pipelines/{pipeline_id}/pipeline-runs/detail` | 获取流水线状态/获取流水线执行详情 |
| `ShowPipelineTemplateDetail` | GET | `/v5/{tenant_id}/api/pipeline-templates/{template_id}` | 查询模板详情 |
| `ShowPipleineStatus` | GET | `/v3/pipelines/{pipeline_id}/status` | 获取流水线状态 |
| `ShowTemplateDetail` | GET | `/v3/templates/{template_id}` | 查询模板详情 |
| `StartNewPipeline` | POST | `/v3/pipelines/{pipeline_id}/start` | 启动流水线 |
| `StopPipelineNew` | POST | `/v3/pipelines/{pipeline_id}/stop` | 停止流水线 |
| `StopPipelineRun` | POST | `/v5/{project_id}/api/pipelines/{pipeline_id}/pipeline-runs/{pipeline_run_id}/stop` | 停止流水线 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
