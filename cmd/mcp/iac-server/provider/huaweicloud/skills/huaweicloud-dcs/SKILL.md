---
name: huaweicloud-dcs
description: HuaweiCloud DCS API guide. 137 APIs covering IP白名单管理, 会话管理, 其他接口, 分片与副本, 参数管理.
---

# HuaweiCloud DCS API Guide

137 APIs. Tags: IP白名单管理, 会话管理, 其他接口, 分片与副本, 参数管理, 后台任务管理, 备份与恢复, 实例管理, 实例诊断, 数据迁移, 日志管理, 标签管理, 模板管理, 生命周期管理, 离线全量key分析, 缓存分析, 网络安全, 账号管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateOrDeleteTags` | POST | `/v2/{project_id}/dcs/{instance_id}/tags/action` | 批量添加或删除标签 |
| `BatchDeleteInstances` | DELETE | `/v2/{project_id}/instances` | 批量删除实例 |
| `BatchRestartOnlineMigrationTasks` | POST | `/v2/{project_id}/migration-tasks/batch-restart` | 批量重启在线迁移任务 |
| `BatchShowNodesInformation` | GET | `/v2/{project_id}/instances-logical-nodes` | 批量查询实例节点信息 |
| `BatchStopMigrationTasks` | POST | `/v2/{project_id}/migration-task/batch-stop` | 批量停止数据迁移任务 |
| `ChangeMasterStandby` | POST | `/v2/{project_id}/instances/{instance_id}/swap` | 主备切换 |
| `ChangeMasterStandbyAsync` | PUT | `/v2/{project_id}/instances/{instance_id}/async-swap` | 异步交换实例主备节点 |
| `ChangeNodesStartStopStatus` | PUT | `/v2/{project_id}/instances/{instance_id}/nodes/status` | 指定实例节点启停开关 |
| `CopyInstance` | POST | `/v2/{project_id}/instances/{instance_id}/backups` | 备份指定实例 |
| `CreateAclAccount` | POST | `/v2/{project_id}/instances/{instance_id}/accounts` | 创建ACL账号 |
| `CreateAutoExpireScanTask` | POST | `/v2/{project_id}/instances/{instance_id}/scan-expire-keys-task` | 创建过期key扫描任务 |
| `CreateBigkeyScanTask` | POST | `/v2/{project_id}/instances/{instance_id}/bigkey-task` | 创建大key分析任务 |
| `CreateCustomTemplate` | POST | `/v2/{project_id}/config-templates` | 创建自定义模板 |
| `CreateDiagnosisTask` | POST | `/v2/{project_id}/instances/{instance_id}/diagnosis` | 创建实例诊断任务 |
| `CreateHotkeyScanTask` | POST | `/v2/{project_id}/instances/{instance_id}/hotkey-task` | 创建热key分析任务 |
| `CreateInstance` | POST | `/v2/{project_id}/instances` | 创建缓存实例 |
| `CreateMigrationTask` | POST | `/v2/{project_id}/migration-task` | 创建数据迁移任务 |
| `CreateOfflineKeyAnalysis` | POST | `/v2/{project_id}/instances/{instance_id}/offline/key-analysis` | 创建离线全量key分析任务 |
| `CreateOnlineMigrationTask` | POST | `/v2/{project_id}/migration/instance` | 创建在线数据迁移任务 |
| `CreateRedislog` | POST | `/v2/{project_id}/instances/{instance_id}/redislog` | 采集Redis运行日志 |
| `CreateRedislogDownloadLink` | POST | `/v2/{project_id}/instances/{instance_id}/redislog/{id}/links` | 获取日志下载链接 |
| `CreateResizeOrder` | POST | `/v2/{project_id}/orders/instances/{instance_id}/resize` | 包周期实例变更规格 |
| `DeleteAclAccount` | DELETE | `/v2/{project_id}/instances/{instance_id}/accounts/{account_id}` | 删除ACL账号 |
| `DeleteBackgroundTask` | DELETE | `/v2/{project_id}/instances/{instance_id}/tasks/{task_id}` | 删除后台任务 |
| `DeleteBackupFile` | DELETE | `/v2/{project_id}/instances/{instance_id}/backups/{backup_id}` | 删除备份文件 |
| `DeleteBigkeyScanTask` | DELETE | `/v2/{project_id}/instances/{instance_id}/bigkey-task/{bigkey_id}` | 删除大key分析记录 |
| `DeleteCenterTask` | DELETE | `/v2/{project_id}/tasks/{task_id}` | 删除任务中心任务 |
| `DeleteConfigTemplate` | DELETE | `/v2/{project_id}/config-templates/{template_id}` | 删除自定义模板 |
| `DeleteDiagnosisTask` | DELETE | `/v2/{project_id}/instances/{instance_id}/diagnosis` | 删除诊断记录 |
| `DeleteHotkeyScanTask` | DELETE | `/v2/{project_id}/instances/{instance_id}/hotkey-task/{hotkey_id}` | 删除热key分析任务 |

... and 107 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
