---
name: huaweicloud-gaussdbforopengauss
description: HuaweiCloud GaussDBforopenGauss API guide. 257 APIs covering ASP报告, LTS日志, SQL PATCH, SQL执行计划, SQL限流.
---

# HuaweiCloud GaussDBforopenGauss API Guide

257 APIs. Tags: ASP报告, LTS日志, SQL PATCH, SQL执行计划, SQL限流, Top SQL, WDR报告, 事件管理, 任务管理, 会话管理, 全量SQL, 历史API, 参数配置, 回收站, 备份恢复管理, 备份管理, 实例管理, 容灾管理, 引擎版本和规格, 慢SQL, 指标管理, 插件管理, 数据库磁盘类型, 日志管理, 权限管理, 标签管理, 版本升级, 版本升级路径, 磁盘管理, 空间分析, 管理数据库和用户, 诊断优化, 配额管理, 首页总览

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddHbaConfs` | POST | `/v3/{project_id}/instances/{instance_id}/hba-info` | 新增客户端接入认证配置 |
| `AddInstanceTags` | POST | `/v3/{project_id}/instances/{instance_id}/tags` | 添加实例标签。 |
| `AllowDbPrivileges` | POST | `/v3/{project_id}/instances/{instance_id}/db-privilege` | 授权数据库帐号 |
| `AllowDbRolePrivileges` | POST | `/v3.1/{project_id}/instances/{instance_id}/db-privilege` | 授权数据库角色 |
| `AttachEip` | POST | `/v3/{project_id}/instances/{instance_id}/nodes/{node_id}/public-ip` | 绑定/解绑弹性公网IP |
| `AuthorizeBackupDownload` | POST | `/v3/{project_id}/backups/{backup_id}/download/authorization` | 授权备份文件下载 |
| `BatchDeleteInstanceTag` | DELETE | `/v3/{project_id}/instances/{instance_id}/tags` | 批量删除实例标签 |
| `BatchExecuteEvents` | POST | `/v3/{project_id}/schedule-events` | 操作EG事件中心通知事件 |
| `BatchSetBackupPolicy` | PUT | `/v3/{project_id}/backups/policy` | 批量设置自动备份策略 |
| `BatchShowUpgradeCandidateVersions` | POST | `/v3.1/{project_id}/instances/db-upgrade/candidate-versions` | 查询批量实例可升级的版本和升级类型 |
| `BindDNat` | PUT | `/v3/{project_id}/instances/{instance_id}/dnat` | 绑定/解绑NAT网关 |
| `BindLtsConfig` | POST | `/v3/{project_id}/instances/logs/lts-config` | 关联LTS日志流 |
| `CancelScheduleTask` | PUT | `/v3/{project_id}/instances/schedule-task/{task_id}/cancel` | 取消定时任务 |
| `ChangeDemand2Period` | PUT | `/v3/{project_id}/instances/change-charge-mode` | 按需转包周期 |
| `CollectAsp` | POST | `/v3/{project_id}/instances/{instance_id}/asp/collect` | 采集ASP报告 |
| `CollectWdrSnapshot` | POST | `/v3/{project_id}/instances/{instance_id}/wdr-snapshots/collect` | 采集WDR快照报告 |
| `ConfirmRestoredData` | POST | `/v3/{project_id}/instances/{instance_id}/confirm-restore-data` | 备份恢复到目标实例数据后执行数据确认 |
| `CopyConfiguration` | POST | `/v3/{project_id}/configurations/{config_id}/copy` | 复制参数模板 |
| `CreateConfigurationTemplate` | POST | `/v3/{project_id}/configurations` | 创建参数模板 |
| `CreateCrossCloudConstructDisaster` | POST | `/v3.5/{project_id}/instances/{instance_id}/disaster-recovery/construct` | 搭建容灾关系 |
| `CreateDatabase` | POST | `/v3/{project_id}/instances/{instance_id}/database` | 创建数据库 |
| `CreateDatabaseInstance` | POST | `/v3.2/{project_id}/instances` | 创建数据库实例 |
| `CreateDatabaseSchemas` | POST | `/v3/{project_id}/instances/{instance_id}/schema` | 创建数据库SCHEMA |
| `CreateDbInstance` | POST | `/v3.1/{project_id}/instances` | 创建数据库实例 |
| `CreateDbRole` | POST | `/v3.1/{project_id}/instances/{instance_id}/db-role` | 创建数据库角色 |
| `CreateDbUser` | POST | `/v3/{project_id}/instances/{instance_id}/db-user` | 创建数据库用户 |
| `CreateGaussDbInstance` | POST | `/v5/{project_id}/instances` | 创建数据库实例 |
| `CreateInstance` | POST | `/v3/{project_id}/instances` | 创建数据库实例 |
| `CreateLimitTask` | POST | `/v3/{project_id}/instances/{instance_id}/limit-task` | 创建限流任务 |
| `CreateManualBackup` | POST | `/v3/{project_id}/backups` | 创建手动备份 |

... and 227 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
