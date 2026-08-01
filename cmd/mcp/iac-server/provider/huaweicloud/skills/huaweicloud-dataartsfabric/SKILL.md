---
name: huaweicloud-dataartsfabric
description: HuaweiCloud DataArtsFabric API guide. 44 APIs covering Agency, Agreement, ConfigCenter, Endpoint, Framework.
---

# HuaweiCloud DataArtsFabric API Guide

44 APIs. Tags: Agency, Agreement, ConfigCenter, Endpoint, Framework, Health, JobDefinition, Message, Metric, ModelDefinition, Specification, TMS, Workspace

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateFabricWorkspaceTags` | POST | `/v1/{project_id}/fabric-workspace/{workspace_id}/tags/create` | 批量打资源标签 |
| `BatchDeleteFabricWorkspaceTags` | POST | `/v1/{project_id}/fabric-workspace/{workspace_id}/tags/delete` | 批量删除资源标签 |
| `CleanupModel` | DELETE | `/v1/workspaces/{workspace_id}/models/{model_id}` | 删除未使用的模型定义 |
| `CountTagFabricWorkspaces` | POST | `/v1/{project_id}/fabric-workspace/resource-instances/count` | 查询资源实例数量 |
| `CreateAgency` | POST | `/v1/agency` | 创建服务委托 |
| `CreateAgreement` | POST | `/v1/agreement` | 注册租户协议 |
| `CreateEndpoint` | POST | `/v1/workspaces/{workspace_id}/endpoints` | 创建Endpoint |
| `CreateJob` | POST | `/v1/workspaces/{workspace_id}/jobs` | 创建作业 |
| `CreateMessageNotificationPolicy` | POST | `/v1/workspaces/{workspace_id}/messages` | 创建消息通知策略 |
| `CreateModelDefinition` | POST | `/v1/workspaces/{workspace_id}/models` | 创建模型 |
| `CreateWorkspace` | POST | `/v1/workspaces` | 创建Workspace |
| `DeleteAgency` | DELETE | `/v1/agency` | 删除服务委托 |
| `DeleteAgreement` | DELETE | `/v1/agreement` | 删除用户注册协议 |
| `DeleteEndpoint` | DELETE | `/v1/workspaces/{workspace_id}/endpoints/{endpoint_id}` | 删除endpioint |
| `DeleteJob` | DELETE | `/v1/workspaces/{workspace_id}/jobs/{job_id}` | 删除指定作业 |
| `DeleteJobVersion` | DELETE | `/v1/workspaces/{workspace_id}/jobs/{job_id}/versions/{version_id}` | 删除指定作业的指定版本 |
| `DeleteMessageNotificationPolicy` | DELETE | `/v1/workspaces/{workspace_id}/messages/{message_policy_id}` | 删除消息通知策略 |
| `DeleteModelVersion` | DELETE | `/v1/workspaces/{workspace_id}/models/{model_id}/versions/{version_id}` | 删除模型版本 |
| `DeleteWorkspace` | DELETE | `/v1/workspaces/{workspace_id}` | 删除Workspace |
| `ListAgency` | GET | `/v1/agency` | 查询服务委托 |
| `ListBaseModels` | GET | `/v1/base-models` | 列举基模型 |
| `ListEndpoints` | GET | `/v1/workspaces/{workspace_id}/endpoints` | 查询Endpoint列表 |
| `ListFabricProjectTags` | GET | `/v1/{project_id}/fabric-workspace/tags` | 查询项目标签 |
| `ListFabricWorkspaceTags` | GET | `/v1/{project_id}/fabric-workspace/{workspace_id}/tags` | 查询资源标签 |
| `ListFeatures` | GET | `/v1/features` | 查询用户支持特性 |
| `ListFrameworks` | GET | `/v1/workspaces/{workspace_id}/frameworks` | 查询Framework列表 |
| `ListJobs` | GET | `/v1/workspaces/{workspace_id}/jobs` | 查询作业列表 |
| `ListJobVersions` | GET | `/v1/workspaces/{workspace_id}/jobs/{job_id}/versions` | 查询指定作业的版本列表 |
| `ListMessageNotificationPolicy` | GET | `/v1/workspaces/{workspace_id}/messages` | 查询消息通知策略列表 |
| `ListModels` | GET | `/v1/workspaces/{workspace_id}/models` | 列举模型 |

... and 14 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
