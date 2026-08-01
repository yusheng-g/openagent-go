---
name: huaweicloud-idme
description: HuaweiCloud iDME API guide. 9 APIs covering 应用管理, 运行服务管理.
---

# HuaweiCloud iDME API Guide

9 APIs. Tags: 应用管理, 运行服务管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateXdmApplication` | POST | `/v1/{project_id}/apps` | 创建应用 |
| `DeleteCloudService` | DELETE | `/v1/{project_id}/{service_type}/instances/{instance_id}` | 删除iDME实例 |
| `DeleteXdmApplication` | DELETE | `/v1/{project_id}/apps/{app_id}` | 删除应用 |
| `DeployApplication` | POST | `/v1/{project_id}/envs/{env_id}/apps/{app_id}/deploy` | 部署应用 |
| `ListApps` | GET | `/v1/{project_id}/apps` | 获取租户下的应用清单 |
| `ListEnvs` | GET | `/v1/{project_id}/envs` | 获取运行服务清单 |
| `ModifyApplication` | PUT | `/v1/{project_id}/apps/{app_id}` | 编辑应用 |
| `SubscribeCloudService` | POST | `/v1/{project_id}/{service_type}/instances` | 开通iDME实例 |
| `Uninstall` | DELETE | `/v1/{project_id}/envs/{env_id}/apps/{app_id}` | 卸载应用 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
