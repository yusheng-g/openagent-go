---
name: huaweicloud-hilens
description: HuaweiCloud HiLens API guide. 58 APIs covering 作业管理, 固件管理, 密钥管理, 工作空间管理, 应用部署管理.
---

# HuaweiCloud HiLens API Guide

58 APIs. Tags: 作业管理, 固件管理, 密钥管理, 工作空间管理, 应用部署管理, 技能市场, 标签管理, 设备告警管理, 设备管理, 设备管理v1, 配置项管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddDeploymentNodes` | PUT | `/v3/{project_id}/ai-mgr/deployments/{deployment_id}/nodes` | 批量部署 |
| `BatchCreateNodeTags` | POST | `/v3/{project_id}/tag-mgr/node-tags` | 批量添加节点标签 |
| `CreateConfigMap` | POST | `/v3/{project_id}/ai-mgr/configmaps` | 创建配置项 |
| `CreateDeployment` | POST | `/v3/{project_id}/ai-mgr/deployments` | 创建应用部署 |
| `CreateNode` | POST | `/v3/{project_id}/ai-mgr/nodes` | 注册设备 |
| `CreateOrderForm` | POST | `/v1/{project_id}/skill-market/skill-order` | 创建免费技能订单 |
| `CreateResourceTags` | POST | `/v3/{project_id}/{resource_type}/{resource_id}/tags` | 添加资源标签 |
| `CreateSecret` | POST | `/v3/{project_id}/ai-mgr/secrets` | 创建密钥 |
| `CreateTask` | POST | `/v3/{project_id}/ai-mgr/deployments/{deployment_id}/tasks` | 创建作业 |
| `CreateWorkSpace` | POST | `/v3/{project_id}/ai-mgr/workspaces` | 创建工作空间 |
| `DeleteConfigMap` | DELETE | `/v3/{project_id}/ai-mgr/configmaps/{config_map_id}` | 删除配置项 |
| `DeleteDeployment` | DELETE | `/v3/{project_id}/ai-mgr/deployments/{deployment_id}` | 删除应用部署 |
| `DeleteNode` | DELETE | `/v3/{project_id}/ai-mgr/nodes/{node_id}` | 删除设备 |
| `DeletePod` | DELETE | `/v3/{project_id}/ai-mgr/deployments/{deployment_id}/{pod_id}` | 删除应用实例 |
| `DeleteResourceTag` | DELETE | `/v3/{project_id}/{resource_type}/{resource_id}/tags/{key}` | 删除资源标签 |
| `DeleteSecret` | DELETE | `/v3/{project_id}/ai-mgr/secrets/{secret_id}` | 删除密钥 |
| `DeleteTask` | DELETE | `/v3/{project_id}/ai-mgr/deployments/{deployment_id}/tasks/{task_id}` | 删除作业 |
| `DeleteWorkSpace` | DELETE | `/v3/{project_id}/ai-mgr/workspaces/{workspace_id}` | 删除工作空间 |
| `FreezeNode` | POST | `/v3/{project_id}/ai-mgr/nodes/{node_id}/deactivate` | 将激活订单与设备解绑 |
| `ListConfigMaps` | GET | `/v3/{project_id}/ai-mgr/configmaps` | 查询配置项列表 |
| `ListDeviceAlarms` | GET | `/v1/{project_id}/alarm-manager/alarms` | 获取设备告警列表 |
| `ListDevices` | GET | `/v2/{project_id}/device-manager/devices` | 获取基础版设备列表 |
| `ListFirmwares` | GET | `/v3/ai-mgr/firmwares` | 查询固件列表 |
| `ListPlatformManager` | GET | `/v1/{project_id}/platform-manager/orders` | 获取运行服务费订单列表 |
| `ListResourceTags` | GET | `/v3/{project_id}/tag-mgr/{resource_type}/tags` | 查询某资源类型的标签 |
| `ListSecrets` | GET | `/v3/{project_id}/ai-mgr/secrets` | 查询密钥列表 |
| `ListTasks` | GET | `/v3/{project_id}/ai-mgr/deployments/{deployment_id}/tasks` | 查询作业列表 |
| `ListWorkSpaces` | GET | `/v3/{project_id}/ai-mgr/workspaces` | 获取工作空间列表 |
| `SetDefaultOrderForm` | PUT | `/v1/{project_id}/skill-market/skill-order/{order_id}/default` | 设置默认订单 |
| `ShowConfigMap` | GET | `/v3/{project_id}/ai-mgr/configmaps/{config_map_id}` | 查询配置项详情 |

... and 28 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
