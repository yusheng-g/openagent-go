---
name: huaweicloud-workspaceapp
description: HuaweiCloud WorkspaceApp API guide. 174 APIs covering hotspotSession, 云存储, 产品套餐管理, 任务管理, 使用记录.
---

# HuaweiCloud WorkspaceApp API Guide

174 APIs. Tags: hotspotSession, 云存储, 产品套餐管理, 任务管理, 使用记录, 可用区管理, 存储管理, 定时任务, 应用仓库管理, 应用授权管理, 应用管理, 应用组管理, 服务器管理, 服务器组标签管理, 服务器组管理, 磁盘管理, 租户配置, 策略管理, 订单, 配额管理, 镜像管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddAppGroupAuthorization` | POST | `/v1/{project_id}/app-groups/authorizations` | 增加应用组授权 |
| `AttachImageServerApp` | POST | `/v1/{project_id}/image-servers/{server_id}/actions/attach-app` | 分发软件信息至镜像实例 |
| `AuthorizeObs` | POST | `/v1/{project_id}/app-warehouse/action/authorize` | 获取上传至OBS桶的临时ak/sk |
| `BatchChangeServerImage` | POST | `/v1/{project_id}/app-servers/actions/batch-change-image` | 批量修改服务器的镜像 |
| `BatchChangeServerMaintainMode` | PATCH | `/v1/{project_id}/app-servers/actions/batch-maint` | 标记服务器维护状态 |
| `BatchCreateServerGroupTags` | POST | `/v1/{project_id}/server-group/tags/batch-create` | 批量添加服务器组标签 |
| `BatchDeleteAppGroup` | POST | `/v1/{project_id}/app-groups/batch-delete` | 批量删除应用组 |
| `BatchDeleteAppGroupAuthorization` | POST | `/v1/{project_id}/app-groups/actions/batch-delete-authorization` | 移除应用组授权 |
| `BatchDeleteAppSubJobs` | POST | `/v1/{project_id}/app-server-sub-jobs/actions/batch-delete` | 批量删除子任务 |
| `BatchDeleteCloudStorage` | POST | `/v1/{project_id}/cloud-storages/actions/batch-delete-cloud-storages` | 批量删除云存储 |
| `BatchDeleteImageServer` | PATCH | `/v1/{project_id}/image-servers/actions/batch-delete` | 批量删除镜像实例 |
| `BatchDeleteImageSubJobs` | PATCH | `/v1/{project_id}/image-server-sub-jobs/actions/batch-delete` | 批量删除镜像子任务 |
| `BatchDeletePersistentStorage` | POST | `/v1/{project_id}/persistent-storages/actions/batch-delete` | 批量删除WKS存储 |
| `BatchDeleteScheduleTask` | POST | `/v1/{project_id}/schedule-task/actions/batch-delete` | 批量删除定时任务 |
| `BatchDeleteServer` | POST | `/v1/{project_id}/app-servers/actions/batch-delete` | 批量删除服务器 |
| `BatchDeleteServerGroupTags` | DELETE | `/v1/{project_id}/server-group/tags/batch-delete` | 批量删除服务器组标签 |
| `BatchDeleteWarehouseApp` | POST | `/v1/{project_id}/app-warehouse/actions/batch-delete` | 批量删除应用仓库中的指定应用 |
| `BatchDisableApp` | POST | `/v1/{project_id}/app-groups/{app_group_id}/apps/actions/disable` | 批量禁用应用 |
| `BatchEnableApp` | POST | `/v1/{project_id}/app-groups/{app_group_id}/apps/actions/enable` | 批量启用应用 |
| `BatchMigrateHostsServer` | PATCH | `/v1/{project_id}/app-servers/hosts/batch-migrate` | 迁移云办公主机下面的服务器到目标云办公主机 |
| `BatchRebootServer` | PATCH | `/v1/{project_id}/app-servers/actions/batch-reboot` | 重启服务器 |
| `BatchReinstallServer` | POST | `/v1/{project_id}/app-servers/actions/batch-reinstall` | 批量重装服务器 |
| `BatchRejoinDomain` | PATCH | `/v1/{project_id}/app-servers/actions/batch-rejoin-domain` | 批量服务器重新加域 |
| `BatchStartServer` | PATCH | `/v1/{project_id}/app-servers/actions/batch-start` | 启动服务器 |
| `BatchStopServer` | PATCH | `/v1/{project_id}/app-servers/actions/batch-stop` | 关闭服务器 |
| `BatchUpdateTsvi` | PATCH | `/v1/{project_id}/app-servers/actions/batch-update-tsvi` | 批量更新服务器虚拟会话IP配置 |
| `BatchUpgradeHdaVersion` | PATCH | `/v1/{project_id}/app-servers/access-agent/actions/upgrade` | 批量升级服务器HDA版本 |
| `BindAppWarehouseBucket` | POST | `/v1/{project_id}/app-warehouse/bucket` | 添加用户应用仓库桶及桶授权 |
| `ChangeCluster` | POST | `/v1/{project_id}/cloud-storages/{storage_id}/actions/change-cluster` | 切换文件夹归属集群 |
| `ChangeServerImage` | POST | `/v1/{project_id}/app-servers/{server_id}/actions/change-image` | 修改服务器的镜像 |

... and 144 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
