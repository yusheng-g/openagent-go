---
name: huaweicloud-gaussdbfornosql
description: HuaweiCloud GaussDBforNoSQL API guide. 151 APIs covering 任务管理, 企业项目管理, 参数模板管理, 备份与恢复, 实例管理.
---

# HuaweiCloud GaussDBforNoSQL API Guide

151 APIs. Tags: 任务管理, 企业项目管理, 参数模板管理, 备份与恢复, 实例管理, 实例负载均衡管理, 容灾管理, 数据库账号管理, 日志管理, 查询API版本, 查询专属资源列表, 查询所有实例规格信息, 查询数据库版本信息, 标签管理, 管理数据库和用户, 连接管理, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ApplyConfiguration` | PUT | `/v3/{project_id}/configurations/{config_id}/apply` | 应用参数模板 |
| `ApplyConfigurationToInstances` | PUT | `/v3.1/{project_id}/configurations/{config_id}/apply` | 应用参数模板 |
| `BatchDeleteManualBackup` | DELETE | `/v3/{project_id}/instances/backups` | 批量删除手动备份 |
| `BatchTagAction` | POST | `/v3/{project_id}/instances/{instance_id}/tags/action` | 批量添加或删除资源标签 |
| `BatchUpgradeDatabaseVersion` | POST | `/v3/{project_id}/instances/db-upgrade` | 批量数据库补丁升级 |
| `CancelInstanceScheduleWindow` | DELETE | `/v3/{project_id}/scheduled-jobs/{job_id}` | 取消定时任务 |
| `CheckDisasterRecoveryOperation` | POST | `/v3/{project_id}/instances/{instance_id}/disaster-recovery/precheck` | 校验实例是否可以与指定实例建立/解除容灾关系 |
| `CheckWeekPassword` | POST | `/v3/{project_id}/weak-password-verification` | 判断弱密码 |
| `ClearInstanceSessions` | DELETE | `/v3/{project_id}/instances/{instance_id}/sessions` | 关闭实例所有节点会话 |
| `CompareConfiguration` | POST | `/v3/{project_id}/configurations/comparison` | 参数模板比较 |
| `CopyConfiguration` | POST | `/v3/{project_id}/configurations/{config_id}/copy` | 复制参数模板 |
| `CreateBack` | POST | `/v3/{project_id}/instances/{instance_id}/backups` | 创建手动备份 |
| `CreateColdVolume` | POST | `/v3/{project_id}/instances/{instance_id}/cold-volume` | ‘创建冷数据存储’ |
| `CreateConfiguration` | POST | `/v3/{project_id}/configurations` | 创建参数模板 |
| `CreateDbCacheMapping` | POST | `/v3/{project_id}/dbcache/mapping` | 创建内存加速映射 |
| `CreateDbCacheRule` | POST | `/v3/{project_id}/dbcache/rule` | 创建内存加速规则 |
| `CreateDbUser` | POST | `/v3/{project_id}/redis/instances/{instance_id}/db-users` | 创建Redis数据库账号 |
| `CreateDisasterRecovery` | POST | `/v3/{project_id}/instances/{instance_id}/disaster-recovery/construction` | 搭建实例与特定实例的容灾关系 |
| `CreateGeminiDbDualActive` | POST | `/v3/{project_id}/instances/{instance_id}/dual-active-relationship` | 搭建双活 |
| `CreateInstance` | POST | `/v3/{project_id}/instances` | 创建实例 |
| `DeleteBackup` | DELETE | `/v3/{project_id}/backups/{backup_id}` | 删除手动备份 |
| `DeleteConfiguration` | DELETE | `/v3/{project_id}/configurations/{config_id}` | 删除参数模板 |
| `DeleteDbCacheMapping` | DELETE | `/v3/{project_id}/dbcache/mapping` | 解除内存加速映射 |
| `DeleteDbCacheRule` | DELETE | `/v3/{project_id}/dbcache/rule` | 删除内存加速规则 |
| `DeleteDbUser` | DELETE | `/v3/{project_id}/redis/instances/{instance_id}/db-users` | 删除Redis数据库账号 |
| `DeleteDisasterRecovery` | POST | `/v3/{project_id}/instances/{instance_id}/disaster-recovery/deconstruction` | 解除实例与特定实例的容灾关系 |
| `DeleteEnlargeFailNode` | DELETE | `/v3/{project_id}/instances/{instance_id}/enlarge-failed-nodes` | 删除扩容失败的节点 |
| `DeleteGeminiDbDualActive` | DELETE | `/v3/{project_id}/instances/{instance_id}/dual-active-relationship` | 解除双活 |
| `DeleteInstance` | DELETE | `/v3/{project_id}/instances/{instance_id}` | 删除实例 |
| `DeleteInstancesSession` | DELETE | `/v3/{project_id}/redis/nodes/{node_id}/sessions` | 关闭实例节点会话 |

... and 121 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
