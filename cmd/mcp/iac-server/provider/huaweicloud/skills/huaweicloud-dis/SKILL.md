---
name: huaweicloud-dis
description: HuaweiCloud DIS API guide. 38 APIs covering App管理, Checkpoint管理, 数据管理, 标签管理, 监控管理.
---

# HuaweiCloud DIS API Guide

38 APIs. Tags: App管理, Checkpoint管理, 数据管理, 标签管理, 监控管理, 转储任务管理, 通道管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateTags` | POST | `/v2/{project_id}/stream/{stream_id}/tags/action` | 批量添加资源标签 |
| `BatchDeleteTags` | POST | `/v2/{project_id}/stream/{stream_id}/tags/action` | 批量删除资源标签 |
| `BatchStartTransferTask` | POST | `/v2/{project_id}/streams/{stream_name}/transfer-tasks/action` | 批量启动转储任务 |
| `BatchStopTransferTask` | POST | `/v2/{project_id}/streams/{stream_name}/transfer-tasks/action` | 批量暂停转储任务 |
| `CommitCheckpoint` | POST | `/v2/{project_id}/checkpoints` | 提交Checkpoint |
| `ConsumeRecords` | GET | `/v2/{project_id}/records` | 下载数据 |
| `CreateApp` | POST | `/v2/{project_id}/apps` | 创建消费App |
| `CreateCloudTableTransferTask` | POST | `/v2/{project_id}/streams/{stream_name}/transfer-tasks` | 添加CloudTable转储任务 |
| `CreateDliTransferTask` | POST | `/v2/{project_id}/streams/{stream_name}/transfer-tasks` | 添加DLI转储任务 |
| `CreateDwsTransferTask` | POST | `/v2/{project_id}/streams/{stream_name}/transfer-tasks` | 添加DWS转储任务 |
| `CreateMrsTransferTask` | POST | `/v2/{project_id}/streams/{stream_name}/transfer-tasks` | 添加MRS转储任务 |
| `CreateObsTransferTask` | POST | `/v2/{project_id}/streams/{stream_name}/transfer-tasks` | 添加OBS转储任务 |
| `CreatePolicies` | POST | `/v2/{project_id}/streams/{stream_name}/policies` | 添加权限策略 |
| `CreateStream` | POST | `/v2/{project_id}/streams` | 创建通道 |
| `CreateTag` | POST | `/v2/{project_id}/stream/{stream_id}/tags` | 给指定通道添加标签 |
| `DeleteApp` | DELETE | `/v2/{project_id}/apps/{app_name}` | 删除App |
| `DeleteCheckpoint` | DELETE | `/v2/{project_id}/checkpoints` | 删除Checkpoint |
| `DeleteStream` | DELETE | `/v2/{project_id}/streams/{stream_name}` | 删除指定通道 |
| `DeleteTag` | DELETE | `/v2/{project_id}/stream/{stream_id}/tags/{key}` | 删除指定通道的标签 |
| `DeleteTransferTask` | DELETE | `/v2/{project_id}/streams/{stream_name}/transfer-tasks/{task_name}` | 删除转储任务 |
| `ListApp` | GET | `/v2/{project_id}/apps` | 查询App列表 |
| `ListPolicies` | GET | `/v2/{project_id}/streams/{stream_name}/policies` | 查询权限策略列表 |
| `ListResourcesByTags` | POST | `/v2/{project_id}/stream/resource_instances/action` | 使用标签过滤资源(通道等) |
| `ListStreams` | GET | `/v2/{project_id}/streams` | 查询通道列表 |
| `ListTags` | GET | `/v2/{project_id}/stream/tags` | 查询指定区域所有标签集合 |
| `ListTransferTasks` | GET | `/v2/{project_id}/streams/{stream_name}/transfer-tasks` | 查询转储任务列表 |
| `SendRecords` | POST | `/v2/{project_id}/records` | 上传数据 |
| `ShowApp` | GET | `/v2/{project_id}/apps/{app_name}` | 查看App详情 |
| `ShowCheckpoint` | GET | `/v2/{project_id}/checkpoints` | 查询Checkpoint详情 |
| `ShowConsumerState` | GET | `/v2/{project_id}/apps/{app_name}/streams/{stream_name}` | 查看App消费状态 |

... and 8 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
