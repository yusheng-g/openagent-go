---
name: huaweicloud-dds
description: HuaweiCloud DDS API guide. 130 APIs covering 任务管理, 参数配置, 备份与恢复, 实例管理, 引擎版本和规格.
---

# HuaweiCloud DDS API Guide

130 APIs. Tags: 任务管理, 参数配置, 备份与恢复, 实例管理, 引擎版本和规格, 数据库运维, 查询API版本, 标签管理, 管理数据库和用户, 获取日志信息, 连接管理, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddReadonlyNode` | POST | `/v3/{project_id}/instances/{instance_id}/readonly-node` | 实例新增只读节点 |
| `AddShardingNode` | POST | `/v3/{project_id}/instances/{instance_id}/enlarge` | 扩容集群实例的节点数量 |
| `AttachEip` | POST | `/v3/{project_id}/nodes/{node_id}/bind-eip` | 绑定弹性公网IP |
| `AttachInternalIp` | POST | `/v3/{project_id}/instances/{instance_id}/modify-internal-ip` | 修改实例内网地址 |
| `BatchDeleteBackup` | DELETE | `/v3/{project_id}/instances/backups` | 批量删除手动备份 |
| `BatchDeleteShards` | POST | `/v3/{project_id}/instances/{instance_id}/shards/batch-delete` | 删除分片 |
| `BatchTagAction` | POST | `/v3/{project_id}/instances/{instance_id}/tags/action` | 批量添加或删除资源标签 |
| `BatchUpgradeDatabaseVersion` | POST | `/v3/{project_id}/instances/db-upgrade` | 批量数据库补丁升级 |
| `BindPublicGateway` | POST | `/v3/{project_id}/instances/{instance_id}/nodes/{node_id}/public-gateway` | 绑定公网网关 |
| `CancelEip` | POST | `/v3/{project_id}/nodes/{node_id}/unbind-eip` | 解绑弹性公网IP |
| `CancelScheduledTask` | DELETE | `/v3/{project_id}/scheduled-jobs/{job_id}` | 取消定时任务 |
| `ChangeOpsWindow` | PUT | `/v3/{project_id}/instances/{instance_id}/maintenance-window` | 设置可维护时间段 |
| `CheckPassword` | POST | `/v3/{project_id}/instances/{instance_id}/check-password` | 检查数据库密码 |
| `CheckWeakPassword` | POST | `/v3/{project_id}/weak-password-verification` | 检查弱密码 |
| `CompareConfiguration` | POST | `/v3/{project_id}/configurations/comparison` | 参数模板比较 |
| `CopyConfiguration` | POST | `/v3/{project_id}/configurations/{config_id}/copy` | 复制参数模板 |
| `CreateConfiguration` | POST | `/v3/{project_id}/configurations` | 创建参数模板 |
| `CreateDatabaseRole` | POST | `/v3/{project_id}/instances/{instance_id}/db-role` | 创建数据库角色 |
| `CreateDatabaseUser` | POST | `/v3/{project_id}/instances/{instance_id}/db-user` | 创建数据库用户 |
| `CreateInstance` | POST | `/v3/{project_id}/instances` | 创建实例 |
| `CreateIp` | POST | `/v3/{project_id}/instances/{instance_id}/create-ip` | 创建集群的Shard/Config IP |
| `CreateKillOpRule` | POST | `/v3/{project_id}/instances/{instance_id}/kill-op-rule` | 创建killOp规则 |
| `CreateManualBackup` | POST | `/v3/{project_id}/backups` | 创建手动备份 |
| `DeleteAuditLog` | DELETE | `/v3/{project_id}/instances/{instance_id}/auditlog` | 删除审计日志 |
| `DeleteConfiguration` | DELETE | `/v3/{project_id}/configurations/{config_id}` | 删除参数模板 |
| `DeleteDatabaseRole` | DELETE | `/v3/{project_id}/instances/{instance_id}/db-role` | 删除数据库角色 |
| `DeleteDatabaseUser` | DELETE | `/v3/{project_id}/instances/{instance_id}/db-user` | 删除数据库用户 |
| `DeleteInstance` | DELETE | `/v3/{project_id}/instances/{instance_id}` | 删除实例 |
| `DeleteIp` | DELETE | `/v3/{project_id}/instances/{instance_id}/ip` | 删除集群的Shard/Config IP |
| `DeleteKillOpRuleList` | DELETE | `/v3/{project_id}/instances/{instance_id}/kill-op-rule` | 删除killOp规则 |

... and 100 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
