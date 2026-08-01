---
name: huaweicloud-dataartsinsight
description: HuaweiCloud DataArtsInsight API guide. 33 APIs covering 产品实例, 仪表板大屏资源统一, 协同授权, 导入导出参数, 工作空间.
---

# HuaweiCloud DataArtsInsight API Guide

33 APIs. Tags: 产品实例, 仪表板大屏资源统一, 协同授权, 导入导出参数, 工作空间, 数据源, 数据集, 数据集权限, 用户标签, 资源迁移

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchSaveAuth` | POST | `/v1/{project_id}/authorization/cooperate-authorization/rules/batch-save` | 批量保存、修改、删除协同授权 |
| `CreateAndUpdateExportConfig` | POST | `/v1/{project_id}/exports/configs` | 配置导出参数 |
| `CreateAndUpdatePermission` | POST | `/v1/{project_id}/datasets/{dataset_id}/permissions` | 配置数据集权限 |
| `CreateDataConnection` | POST | `/v1/{project_id}/connections` | 数据源新增 |
| `CreateUserTag` | POST | `/v1/{project_id}/tags` | 创建用户标签 |
| `CreateWorkspace` | POST | `/v1/{project_id}/instances/{instance_id}/workspaces` | 创建工作空间 |
| `DeleteDataConnectionByConnectionId` | DELETE | `/v1/{project_id}/connections/{connection_id}` | 删除数据源 |
| `DeleteRule` | DELETE | `/v1/{project_id}/datasets/{dataset_id}/permissions/{permission_id}` | 删除权限 |
| `DeleteUserTag` | DELETE | `/v1/{project_id}/tags/{tag_id}` | 删除用户标签 |
| `DeleteWorkspace` | DELETE | `/v1/{project_id}/instances/{instance_id}/workspaces/{workspace_id}` | 删除工作空间 |
| `ExportResourcePackage` | POST | `/v1/{project_id}/resource-package/export` | API导出资源包 |
| `ImportResourcePackage` | POST | `/v1/{project_id}/resource-package/api-import` | API 导入资源包文件 |
| `ListAuthed` | GET | `/v1/{project_id}/authorization/cooperate-authorization/rules` | 协同授权列表 |
| `ListAuthProperties` | GET | `/v1/{project_id}/authorization/cooperate-authorization/properties` | 获取资源属性值 |
| `ListCubeAndCatalogList` | GET | `/v1/{project_id}/datasets` | 查询数据集和目录列表 |
| `ListDataConnection` | GET | `/v1/{project_id}/connections` | 获取数据源列表 |
| `ListInstances` | GET | `/v1/{project_id}/instances` | 查询用户已开通产品实例列表 |
| `ListPermission` | GET | `/v1/{project_id}/datasets/{dataset_id}/permissions` | 获取数据集权限列表 |
| `ListPermissionConfig` | GET | `/v1/{project_id}/datasets/{dataset_id}/permission-config` | 获取数据集权限配置信息 |
| `ListResources` | GET | `/v1/{project_id}/resources/{resource_type}` | 查询仪表板或者大屏列表 |
| `ListUserTagHead` | GET | `/v1/{project_id}/tags/head` | 获取用户标签头 |
| `ListUserTagValue` | GET | `/v1/{project_id}/tags/value` | 获取用户标签值 |
| `ListWorkspaces` | GET | `/v1/{project_id}/instances/{instance_id}/workspaces` | 查询工作空间 |
| `SaveDatasetForOpenApi` | POST | `/v1/{project_id}/datasets/save` | 保存数据集 |
| `SaveOrUpdateAuthProperties` | POST | `/v1/{project_id}/authorization/cooperate-authorization/properties` | 保存或修改资源属性值 |
| `SaveUserTagValue` | PUT | `/v1/{project_id}/tags/{tag_id}/values` | 保存用户标签内容(按用户) |
| `ShowDataConnectionByConnectionId` | GET | `/v1/{project_id}/connections/{connection_id}` | 获取数据源详情 |
| `ShowDatasetDetail` | GET | `/v1/{project_id}/datasets/{dataset_id}/metadata` | 获取数据集详情 |
| `ShowImportResourceTaskDetail` | GET | `/v1/{project_id}/resource-package/import-tasks/{task_id}` | 获取导入任务详情 |
| `UpdateDataConnection` | PUT | `/v1/{project_id}/connections/{connection_id}` | 数据源更新 |

... and 3 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
