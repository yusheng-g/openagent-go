---
name: huaweicloud-rds
description: HuaweiCloud RDS API guide. 324 APIs covering Msdtc, RDS, sql统计, 事件中心, 任务管理. Detailed swagger definitions in references/<APIName>.json.
---

# HuaweiCloud RDS API Guide

324 APIs. Tags: Msdtc, RDS, sql统计, 事件中心, 任务管理, 冷热分离, 参数配置, 发布订阅(SQL Server), 回收站, 备份与恢复, 实例管理, 引擎版本和规格, 数据库代理, 数据库代理(PostgreSQL), 数据库安全性, 日志, 服务商(SQL Server), 查询API版本, 查询版本, 标签管理, 灾备实例, 管理数据库和用户(MySQL), 管理数据库和用户(PostgreSQL), 管理数据库和用户(SQL Server), 获取任务信息, 获取扩展日志下载信息, 获取扩展日志文件列表, 获取日志信息, 配置只读延迟库(PostgreSQL), 配额

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddPostgresqlHbaConf` | POST | `/v3/{project_id}/instances/{instance_id}/hba-info` | 在pg_hba.conf文件最后新增单个或多个配置 |
| `AllowDbPrivilege` | POST | `/v3/{project_id}/instances/{instance_id}/db_privilege` | 授权数据库帐号 |
| `AllowDbUserPrivilege` | POST | `/v3/{project_id}/instances/{instance_id}/db_privilege` | 授权数据库帐号 |
| `AllowSqlserverDbUserPrivilege` | POST | `/v3/{project_id}/instances/{instance_id}/db_privilege` | 授权数据库帐号 |
| `ApplyConfigurationAsync` | PUT | `/v3.1/{project_id}/configurations/{config_id}/apply` | 应用参数模板 |
| `AttachEip` | PUT | `/v3/{project_id}/instances/{instance_id}/public-ip` | 绑定和解绑弹性公网IP |
| `BatchAddMsdtcs` | POST | `/v3/{project_id}/instances/{instance_id}/msdtc/host` | 添加MSDTC |
| `BatchDeleteInstance` | POST | `/v3/{project_id}/instances/batch-delete` | 批量删除实例 |
| `BatchDeleteManualBackup` | POST | `/v3/{project_id}/backups/batch-delete` | 批量删除手动备份 |
| `BatchExecuteEvents` | POST | `/v3/{project_id}/schedule-events` | 操作EG事件中心通知的事件 |
| `BatchModifyPublication` | PUT | `/v3/{project_id}/instances/{instance_id}/replication/publications` | 批量修改发布 |
| `BatchModifySubscription` | PUT | `/v3/{project_id}/instances/{instance_id}/replication/subscriptions` | 批量修改订阅 |
| `BatchResizeFlavor` | POST | `/v3/{project_id}/instances/batch/resize` | 批量变更实例规格 |
| `BatchRestoreDatabase` | POST | `/v3/{project_id}/instances/batch/restore/databases` | 库级时间点恢复 |
| `BatchRestorePostgreSqlTables` | POST | `/v3/{project_id}/instances/batch/restore/tables` | 表级时间点恢复(PostgreSQL) |
| `BatchStopInstance` | POST | `/v3/{project_id}/instances/batch/action/shutdown` | 批量停止实例 |
| `BatchTagAddAction` | POST | `/v3/{project_id}/instances/{instance_id}/tags/action` | 批量添加标签 |
| `BatchTagDelAction` | POST | `/v3/{project_id}/instances/{instance_id}/tags/action` | 批量删除标签 |
| `ChangeBackupConfig` | PUT | `/v3/{project_id}/instances/{instance_id}/backups/config` | 切换实例备份方式(PostgreSQL) |
| `ChangeFailoverMode` | PUT | `/v3/{project_id}/instances/{instance_id}/failover/mode` | 更改主备实例的数据同步方式 |
| `ChangeFailoverStrategy` | PUT | `/v3/{project_id}/instances/{instance_id}/failover/strategy` | 切换主备实例的倒换策略 |
| `ChangeOpsWindow` | PUT | `/v3/{project_id}/instances/{instance_id}/ops-window` | 设置可维护时间段 |
| `ChangeProxyScale` | POST | `/v3/{project_id}/instances/{instance_id}/proxy/scale` | 数据库代理规格变更 |
| `ChangeTheDelayThreshold` | PUT | `/v3/{project_id}/instances/{instance_id}/proxy/delay-threshold` | 修改读写分离阈值 |
| `CheckInstanceForUpgrade` | PUT | `/v3/{project_id}/instances/{instance_id}/upgrade-version/precheck` | 大版本升级预检查 |
| `CheckWeakpwd` | POST | `/v3/{project_id}/weakpwd` | 弱密码校验 |
| `CollectPublicationMonitor` | GET | `/v3/{project_id}/instances/{instance_id}/replication/publications/{publication_id}/monitor` | 查询发布监控信息 |
| `CollectSubscriptionMonitor` | GET | `/v3/{project_id}/instances/{instance_id}/replication/subscriptions/{subscription_id}/monitor` | 查询订阅监控信息 |
| `CompareConfiguration` | PUT | `/v3/{project_id}/configurations/difference` | 比较参数模板 |
| `CopyConfiguration` | POST | `/v3/{project_id}/configurations/{config_id}/copy` | 复制参数模板 |

... and 294 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
