---
name: huaweicloud-codeartsdeploy
description: HuaweiCloud CodeArtsDeploy API guide. 65 APIs covering 主机管理, 主机集群权限管理, 主机集群管理, 应用分组管理, 应用权限管理.
---

# HuaweiCloud CodeArtsDeploy API Guide

65 APIs. Tags: 主机管理, 主机集群权限管理, 主机集群管理, 应用分组管理, 应用权限管理, 应用管理, 环境权限管理, 环境管理, 部署记录度量

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchDeleteApp` | POST | `/v2/applications/batch-delete` | 批量删除项目下应用 |
| `BatchDeleteHosts` | POST | `/v1/resources/host-groups/{group_id}/hosts/batch-delete` | 批量删除主机集群下的主机 |
| `BatchUpdateApplicationPermissions` | PUT | `/v3/applications/permissions` | 批量修改应用权限 |
| `BatchUpdatePermissionLevel` | PUT | `/v3/applications/permission-level` | 批量配置应用下鉴权级别 |
| `CheckCanCreate` | GET | `/v1/applications/creatable` | 查询当前用户是否有项目下创建应用权限 |
| `CheckIsDuplicateAppName` | GET | `/v1/applications/exist` | 查询项目下是否存在同名应用 |
| `CheckWhetherHostGroupCanBeCreated` | GET | `/v1/host-groups/creatable/{project_id}/permissions` | 判断当前用户在项目下是否有权限创建主机集群 |
| `CopyApplication` | POST | `/v1/applications/{app_id}/duplicate` | 复制应用 |
| `CopyHostsToTarget` | POST | `/v1/resources/host-groups/{group_id}/hosts/replication` | 批量复制主机至目标主机集群 |
| `CreateApp` | POST | `/v1/applications` | 新建应用 |
| `CreateAppGroups` | POST | `/v1/projects/{project_id}/applications/groups` | 创建分组 |
| `CreateDeploymentGroup` | POST | `/v2/host-groups` | 新建主机集群 |
| `CreateDeploymentHost` | POST | `/v2/host-groups/{group_id}/hosts` | 新建主机 |
| `CreateDeployTaskByTemplate` | POST | `/v2/tasks/template-task` | 通过模板新建应用 |
| `CreateEnvironment` | POST | `/v1/applications/{application_id}/environments` | 应用下创建环境 |
| `CreateHost` | POST | `/v1/resources/host-groups/{group_id}/hosts` | 新建主机 |
| `CreateHostCluster` | POST | `/v1/resources/host-groups` | 新建主机集群 |
| `DeleteAppGroups` | DELETE | `/v1/projects/{project_id}/applications/groups/{group_id}` | 删除分组 |
| `DeleteApplication` | DELETE | `/v1/applications/{app_id}` | 删除应用 |
| `DeleteDeploymentGroup` | DELETE | `/v2/host-groups/{group_id}` | 删除主机集群 |
| `DeleteDeploymentHost` | DELETE | `/v2/host-groups/{group_id}/hosts/{host_id}` | 删除主机 |
| `DeleteDeployTask` | DELETE | `/v2/tasks/{task_id}` | 删除应用 |
| `DeleteEnvironment` | DELETE | `/v1/applications/{application_id}/environments/{environment_id}` | 删除应用下的环境 |
| `DeleteHost` | DELETE | `/v1/resources/host-groups/{group_id}/hosts/{host_id}` | 删除主机集群下主机 |
| `DeleteHostCluster` | DELETE | `/v1/resources/host-groups/{group_id}` | 删除主机集群 |
| `DeleteHostFromEnvironment` | DELETE | `/v1/applications/{application_id}/environments/{environment_id}/{host_id}` | 环境下删除主机 |
| `ImportHostToEnvironment` | POST | `/v1/applications/{application_id}/environments/{environment_id}/hosts/import` | 环境下导入主机 |
| `ListAllApp` | POST | `/v1/applications/list` | 获取应用列表 |
| `ListAppGroups` | GET | `/v1/projects/{project_id}/applications/groups` | 查询分组列表 |
| `ListApplicationPermissions` | GET | `/v3/applications/permissions` | 查询应用实例级/项目级权限矩阵 |

... and 35 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
