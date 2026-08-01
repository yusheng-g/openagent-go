---
name: huaweicloud-ges
description: HuaweiCloud GES API guide. 61 APIs covering GraphPlugins管理API, 任务中心API, 元数据管理API, 图管理API, 备份管理API.
---

# HuaweiCloud GES API Guide

61 APIs. Tags: GraphPlugins管理API, 任务中心API, 元数据管理API, 图管理API, 备份管理API, 系统管理API

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AttachEip` | POST | `/v1.0/{project_id}/graphs/{graph_id}/action` | 绑定EIP(1.0.6) |
| `AttachEip2` | POST | `/v2/{project_id}/graphs/{graph_id}/bind-eip` | 绑定EIP |
| `ChangeSecurityGroup` | POST | `/v2/{project_id}/graphs/{graph_id}/sg/change` | 切换安全组 |
| `ClearGraph` | POST | `/v1.0/{project_id}/graphs/{graph_id}/action` | 清空图(2.1.2) |
| `ClearGraph2` | POST | `/v2/{project_id}/graphs/{graph_id}/clear-graph` | 清空图 |
| `CreateBackup` | POST | `/v1.0/{project_id}/graphs/{graph_id}/backups` | 新增备份(1.0.0) |
| `CreateBackup2` | POST | `/v2/{project_id}/graphs/{graph_id}/backups` | 新增备份 |
| `CreateGraph` | POST | `/v1.0/{project_id}/graphs` | 创建图(2.2.2) |
| `CreateGraph2` | POST | `/v2/{project_id}/graphs` | 创建图 |
| `CreateMetadata` | POST | `/v1.0/{project_id}/graphs/metadatas` | 新增元数据(2.1.18) |
| `CreateMetadata2` | POST | `/v2/{project_id}/graphs/metadatas` | 新增元数据 |
| `DeleteBackup` | DELETE | `/v1.0/{project_id}/graphs/{graph_id}/backups/{backup_id}` | 删除备份(1.0.0) |
| `DeleteBackup2` | DELETE | `/v2/{project_id}/graphs/{graph_id}/backups/{backup_id}` | 删除备份 |
| `DeleteGraph` | DELETE | `/v1.0/{project_id}/graphs/{graph_id}` | 删除图(1.0.0) |
| `DeleteGraph2` | DELETE | `/v2/{project_id}/graphs/{graph_id}` | 删除图 |
| `DeleteMetadata` | DELETE | `/v1.0/{project_id}/graphs/metadatas/{metadata_id}` | 删除元数据(1.0.2) |
| `DeleteMetadata2` | DELETE | `/v2/{project_id}/graphs/metadatas/{metadata_id}` | 删除元数据 |
| `DeregisterScenes2` | POST | `/v2/{project_id}/graphs/{graph_id}/scenes/unregister` | 取消订阅场景分析插件 |
| `DetachEip` | POST | `/v1.0/{project_id}/graphs/{graph_id}/action` | 解绑EIP(1.0.6) |
| `DetachEip2` | POST | `/v2/{project_id}/graphs/{graph_id}/unbind-eip` | 解绑EIP |
| `ExpandGraph` | POST | `/v1.0/{project_id}/graphs/{graph_id}/expand` | 扩副本(2.2.23) |
| `ExpandGraph2` | POST | `/v2/{project_id}/graphs/{graph_id}/expand` | 扩副本 |
| `ExportBackup2` | POST | `/v2/{project_id}/graphs/{graph_id}/backups/export` | 导出备份 |
| `ExportGraph` | POST | `/v1.0/{project_id}/graphs/{graph_id}/action` | 导出图(1.0.5) |
| `ExportGraph2` | POST | `/v2/{project_id}/graphs/{graph_id}/export-graph` | 导出图 |
| `ImportBackup2` | POST | `/v2/{project_id}/graphs/{graph_id}/backups/import` | 导入备份 |
| `ImportGraph` | POST | `/v1.0/{project_id}/graphs/{graph_id}/action` | 增量导入图(2.1.14) |
| `ImportGraph2` | POST | `/v2/{project_id}/graphs/{graph_id}/import-graph` | 增量导入图 |
| `ListBackups` | GET | `/v1.0/{project_id}/graphs/backups` | 查看所有备份列表(1.0.0) |
| `ListBackups2` | GET | `/v2/{project_id}/graphs/backups` | 查看所有备份列表 |

... and 31 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
