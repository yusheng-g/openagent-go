---
name: huaweicloud-csms
description: HuaweiCloud CSMS API guide. 45 APIs covering 事件管理, 凭据标签管理, 凭据检测管理, 凭据版本状态管理, 凭据版本管理.
---

# HuaweiCloud CSMS API Guide

45 APIs. Tags: 事件管理, 凭据标签管理, 凭据检测管理, 凭据版本状态管理, 凭据版本管理, 凭据轮转管理, 密码管理, 授权管理, 生命周期管理, 用户管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateOrDeleteTags` | POST | `/v1/{project_id}/csms/{secret_id}/tags/action` | 批量添加或删除凭据标签 |
| `BatchImportSecrets` | POST | `/v1/{project_id}/secrets/batch-import` | 批量导入凭据 |
| `CheckSecrets` | POST | `/v1/{project_id}/secrets/checker/check` | 检测传入凭据的凭据强度 |
| `CreateAgency` | POST | `/v1/csms/agencies` | 创建服务委托 |
| `CreateGrants` | POST | `/v1/{project_id}/csms/grants` | 授权操作 |
| `CreateSecret` | POST | `/v1/{project_id}/secrets` | 创建凭据 |
| `CreateSecretEvent` | POST | `/v1/{project_id}/csms/events` | 创建事件 |
| `CreateSecretTag` | POST | `/v1/{project_id}/csms/{secret_id}/tags` | 添加凭据标签 |
| `CreateSecretVersion` | POST | `/v1/{project_id}/secrets/{secret_name}/versions` | 创建凭据版本 |
| `DeleteGrant` | DELETE | `/v1/{project_id}/csms/grants` | 删除授权 |
| `DeleteSecret` | DELETE | `/v1/{project_id}/secrets/{secret_name}` | 立即删除凭据 |
| `DeleteSecretEvent` | DELETE | `/v1/{project_id}/csms/events/{event_name}` | 立即删除事件 |
| `DeleteSecretForSchedule` | POST | `/v1/{project_id}/secrets/{secret_name}/scheduled-deleted-tasks/create` | 创建凭据的定时删除任务 |
| `DeleteSecretStage` | DELETE | `/v1/{project_id}/secrets/{secret_name}/stages/{stage_name}` | 删除凭据的版本状态 |
| `DeleteSecretTag` | DELETE | `/v1/{project_id}/csms/{secret_id}/tags/{key}` | 删除凭据标签 |
| `DownloadSecretBlob` | POST | `/v1/{project_id}/secrets/{secret_name}/backup` | 下载凭据备份 |
| `GenerateRandomPassword` | POST | `/v1/{project_id}/csms/generate-password` | None |
| `ListGrants` | GET | `/v1/{project_id}/csms/grants` | 授权列表 |
| `ListNotificationRecords` | GET | `/v1/{project_id}/csms/notification-records` | 查询已触发的事件通知记录 |
| `ListProjectSecretsTags` | GET | `/v1/{project_id}/csms/tags` | 查询项目标签 |
| `ListResourceInstances` | POST | `/v1/{project_id}/csms/{resource_instances}/action` | 查询凭据实例 |
| `ListSecretEvents` | GET | `/v1/{project_id}/csms/events` | 查询事件列表 |
| `ListSecrets` | GET | `/v1/{project_id}/secrets` | 查询凭据列表 |
| `ListSecretTags` | GET | `/v1/{project_id}/csms/{secret_id}/tags` | 查询凭据标签 |
| `ListSecretTask` | GET | `/v1/{project_id}/csms/tasks` | 查询任务列表 |
| `ListSecretVersions` | GET | `/v1/{project_id}/secrets/{secret_name}/versions` | 查询凭据的版本列表 |
| `ListUsers` | GET | `/v1/csms/users` | 查询用户列表 |
| `RestoreSecret` | POST | `/v1/{project_id}/secrets/{secret_name}/scheduled-deleted-tasks/cancel` | 取消凭据的定时删除任务 |
| `RotateSecret` | POST | `/v1/{project_id}/secrets/{secret_name}/rotate` | 轮转凭据 |
| `ShowAgency` | GET | `/v1/csms/agencies` | 查看是否有服务委托 |

... and 15 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
