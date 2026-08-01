---
name: huaweicloud-gaussdb
description: HuaweiCloud GaussDB API guide. 244 APIs covering HTAP-标准版, 任务中心, 参数模板管理, 备份管理, 多租特性.
---

# HuaweiCloud GaussDB API Guide

244 APIs. Tags: HTAP-标准版, 任务中心, 参数模板管理, 备份管理, 多租特性, 实例管理, 数据库代理, 数据库用户管理, 数据库管理, 日志管理, 智能诊断, 查询数据库引擎的版本, 查询数据库规格, 标签管理, 流量管理, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddDatabasePermission` | POST | `/v3/{project_id}/instances/{instance_id}/db-users/privilege` | 授予数据库用户数据库权限 |
| `BatchChangeInstanceSpecification` | POST | `/v3/{project_id}/instances/batch/flavor` | 批量变更实例规格 |
| `BatchDeleteBackup` | DELETE | `/v3/{project_id}/backups` | 批量删除手动备份 |
| `BatchTagAction` | POST | `/v3/{project_id}/instances/{instance_id}/tags/action` | 批量添加或删除标签 |
| `BatchUpgradeDatabases` | POST | `/v3/{project_id}/instances/database-version/upgrade` | 批量实例小版本升级 |
| `CancelGaussMySqlInstanceEip` | PUT | `/v3/{project_id}/instances/{instance_id}/public-ips/unbind` | 解绑弹性公网IP |
| `CancelScheduleTask` | DELETE | `/v3/{project_id}/scheduled-jobs` | 取消定时任务 |
| `ChangeGaussMySqlInstanceSpecification` | POST | `/v3/{project_id}/instances/{instance_id}/action` | 变更实例规格 |
| `ChangeGaussMySqlProxySpecification` | PUT | `/v3/{project_id}/instances/{instance_id}/proxy/{proxy_id}/flavor` | 数据库代理规格变更 |
| `CheckDataBaseConfig` | POST | `/v3/{project_id}/instances/{instance_id}/starrocks/databases/replication/database-config-check` | HTAP数据同步库配置校验 |
| `CheckResource` | POST | `/v3/{project_id}/resource-check` | 资源预校验 |
| `CheckScheduleTaskExist` | POST | `/v3/{project_id}/instances/{instance_id}/schedule-tasks/exist` | 查询实例是否存在相同定时任务类型 |
| `CheckStarrocksParams` | POST | `/v3/{project_id}/configurations/starrocks/comparison` | 参数对比 |
| `CheckStarRocksResource` | POST | `/v3/{project_id}/starrocks/resource-check` | StarRocks资源检查 |
| `CheckTableConfig` | POST | `/v3/{project_id}/instances/{instance_id}/starrocks/databases/replication/table-config-check` | HTAP数据同步表配置校验 |
| `CollectRealtimeSession` | POST | `/v3/{project_id}/instances/{instance_id}/nodes/{node_id}/realtime-session` | 收集全部实时会话信息 |
| `CopyConfigurations` | POST | `/v3/{project_id}/configurations/{configuration_id}/copy` | 复制参数组 |
| `CopyInstanceConfigurations` | POST | `/v3/{project_id}/instances/{instance_id}/configurations/{configuration_id}/copy` | 复制实例参数组 |
| `CreateAccessControl` | POST | `/v3/{project_id}/instances/{instance_id}/proxy/{proxy_id}/access-control` | 设置访问控制规则 |
| `CreateBackupResourcePackage` | POST | `/v3/{project_id}/backups/resource-package` | 创建备份资源包 |
| `CreateGaussMySqlBackup` | POST | `/v3/{project_id}/backups/create` | 创建手动备份 |
| `CreateGaussMySqlConfiguration` | POST | `/v3/{project_id}/configurations` | 创建参数模板 |
| `CreateGaussMySqlDatabase` | POST | `/v3/{project_id}/instances/{instance_id}/databases` | 创建数据库 |
| `CreateGaussMySqlDatabaseUser` | POST | `/v3/{project_id}/instances/{instance_id}/db-users` | 创建数据库用户 |
| `CreateGaussMysqlDns` | POST | `/v3/{project_id}/instances/{instance_id}/dns` | 申请内网域名 |
| `CreateGaussMySqlInstance` | POST | `/v3/{project_id}/instances` | 创建数据库实例 |
| `CreateGaussMySqlProxy` | POST | `/v3/{project_id}/instances/{instance_id}/proxy` | 开启数据库代理 |
| `CreateGaussMySqlReadonlyNode` | POST | `/v3/{project_id}/instances/{instance_id}/nodes/enlarge` | 创建只读节点 |
| `CreateLtsConfigs` | POST | `/v3/{project_id}/logs/lts-configs` | 批量创建LTS日志配置 |
| `CreateProxyDnsName` | POST | `/v3/{project_id}/instances/{instance_id}/proxy/{proxy_id}/dns` | 开启proxy内网DNS |

... and 214 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
