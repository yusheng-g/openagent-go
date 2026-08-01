---
name: huaweicloud-modelarts
description: HuaweiCloud ModelArts API guide. 315 APIs covering AI应用管理, APP认证管理, HPA自动扩缩容策略, HRA水平资源自动扩缩容策略, Lite Server管理.
---

# HuaweiCloud ModelArts API Guide

315 APIs. Tags: AI应用管理, APP认证管理, HPA自动扩缩容策略, HRA水平资源自动扩缩容策略, Lite Server管理, Workflow工作流管理, 任务管理, 内网接入管理, 在线服务生命周期管理, 工作空间管理, 应用密钥管理, 开发环境管理, 授权管理, 插件管理, 服务标签管理, 服务管理, 服务部署实例管理, 服务部署版本管理, 服务部署生命周期管理, 节点池管理, 节点配置模板, 计划事件, 订单管理, 训练管理, 资源标签管理, 资源池信息, 资源管理, 轻量算力节点, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptScheduledEvent` | POST | `/v1/{project_id}/scheduled-events/{event_id}/accept` | 计划事件授权 |
| `ApigAppExists` | GET | `/v1/{project_id}/app-auth/apps/{app_name}/exists` | 查询APP是否存在 |
| `AttachDevServerVolume` | POST | `/v1/{project_id}/dev-servers/{id}/attachvolume` | Lite Server服务器挂载磁盘 |
| `AttachDynamicStorage` | POST | `/v1/{project_id}/notebooks/{instance_id}/storage` | 动态挂载Notebook存储 |
| `AuthorizeApiToApigApps` | POST | `/v1/{project_id}/services/{service_id}/app-auth-apis/{api_id}/app-auth-api` | 授权API至APP |
| `BatchBindInferApiKeys` | POST | `/v2/{project_id}/services/{service_id}/api-keys/batch-bind` | 批量绑定应用密钥 |
| `BatchBindPoolNodes` | POST | `/v2/{project_id}/pools/{pool_name}/nodes/batch-bind` | 批量为节点绑定逻辑子池 |
| `BatchCreatePoolTags` | POST | `/v1/{project_id}/pools/{pool_name}/tags/create` | 批量创建资源池标签 |
| `BatchCreateServiceTags` | POST | `/v1/{project_id}/services/{resource_id}/tags/create` | 添加资源标签 |
| `BatchDeleteInferIntranetConnections` | POST | `/v2/{project_id}/intranet-connection/delete` | 批量删除内网接入 |
| `BatchDeleteInferServices` | POST | `/v2/{project_id}/services/delete` | 删除指定服务列表 |
| `BatchDeletePoolNodes` | POST | `/v2/{project_id}/pools/{pool_name}/nodes/batch-delete` | 批量删除节点 |
| `BatchDeletePoolTags` | DELETE | `/v1/{project_id}/pools/{pool_name}/tags/delete` | 批量删除资源池标签 |
| `BatchDeleteServiceTags` | DELETE | `/v1/{project_id}/services/{resource_id}/tags/delete` | 删除资源标签 |
| `BatchDevServersAction` | POST | `/v1/{project_id}/dev-servers/action` | 批量操作Lite Server实例 |
| `BatchLockPoolNodes` | POST | `/v2/{project_id}/pools/{pool_name}/nodes/batch-lock` | 批量对节点功能上锁 |
| `BatchMigratePoolNodes` | POST | `/v2/{project_id}/pools/{pool_name}/nodes/batch-migrate` | 批量迁移节点 |
| `BatchRebootPoolNodes` | POST | `/v2/{project_id}/pools/{pool_name}/nodes/batch-reboot` | 批量重启节点 |
| `BatchResetPoolNodes` | POST | `/v2/{project_id}/pools/{pool_name}/nodes/batch-reset` | 重置节点 |
| `BatchResizePoolNodes` | POST | `/v2/{project_id}/pools/{pool_name}/nodes/batch-resize` | 节点规格变更 |
| `BatchUnbindInferApiKeys` | POST | `/v2/{project_id}/services/{service_id}/api-keys/batch-unbind` | 批量解绑应用密钥 |
| `BatchUnlockPoolNodes` | POST | `/v2/{project_id}/pools/{pool_name}/nodes/batch-unlock` | 批量对节点功能解锁 |
| `BatchUpdatePoolNodes` | POST | `/v2/{project_id}/pools/{pool_name}/nodes/batch-update` | 批量更新节点 |
| `BindDevServerPublicIP` | POST | `/v1/{project_id}/dev-servers/{id}/publicips` | Lite Server服务器绑定EIP |
| `BindInferApiKey` | POST | `/v2/{project_id}/services/{service_id}/api-keys/{key_id}/bind` | 绑定应用密钥 |
| `CancelInferDeployment` | POST | `/v2/{project_id}/services/{service_id}/deployments/{deployment_id}/interrupt` | 中断服务部署 |
| `ChangeAlgorithm` | PUT | `/v2/{project_id}/algorithms/{algorithm_id}` | 更新算法 |
| `ChangeDevServerOS` | POST | `/v1/{project_id}/dev-servers/{id}/changeos` | 切换Lite Server服务器操作系统镜像 |
| `ChangeHyperinstanceOS` | POST | `/v1/{project_id}/dev-servers/hyperinstance/{id}/changeos` | 切换Lite Server超节点服务器操作系统镜像 |
| `ChangeTrainingExperiment` | PUT | `/v2/{project_id}/training-experiments/{experiment_id}` | 更新训练实验信息 |

... and 285 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
