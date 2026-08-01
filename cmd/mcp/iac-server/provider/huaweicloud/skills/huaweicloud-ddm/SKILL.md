---
name: huaweicloud-ddm
description: HuaweiCloud DDM API guide. 124 APIs covering DDM会话管理, DDM备份管理, DDM实例管理, DDM实例管理V3, DDM帐号管理.
---

# HuaweiCloud DDM API Guide

124 APIs. Tags: DDM会话管理, DDM备份管理, DDM实例管理, DDM实例管理V3, DDM帐号管理, DDM慢查询, DDM监控管理, DDM账号管理, DDM逻辑库管理, DN管理, SQL限流, SQL限流V3, 参数配置, 查询API版本, 版本管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchDeleteNodes` | POST | `/v3/{project_id}/instances/{instance_id}/nodes/batch-delete` | 批量删除实例的节点 |
| `BatchModifyReadWriteStrategy` | PUT | `/v3/{project_id}/instances/{instance_id}/action/batch-modify-read-write-strategy` | 批量设置读策略V3 |
| `BindEip` | POST | `/v3/{project_id}/instances/{instance_id}/eip` | 绑定弹性公网IP |
| `CancelMigration` | PUT | `/v3/{project_id}/instances/{instance_id}/databases/{db_name}/migration/jobs/{job_id}/cancel` | 取消分片变更 |
| `ChangeDatabaseVersion` | POST | `/v3/{project_id}/instances/{instance_id}/database-version/change-version` | 变更内核版本 |
| `ChangeStrategy` | PUT | `/v3/{project_id}/instances/{instance_id}/databases/{db_name}/migration/jobs/{job_id}/route-switch-strategy` | 修改切换路由策略 |
| `CheckDataNodeConnection` | POST | `/v3/{project_id}/instances/{instance_id}/rds/connection` | rds连通性检查V3 |
| `CheckMigrateLogicDb` | POST | `/v3/{project_id}/instances/{instance_id}/databases/{db_name}/migration/precheck` | 分片变更预校验 |
| `CheckPreliminaryResults` | GET | `/v3/{project_id}/instances/{instance_id}/databases/{db_name}/migration/precheck/{job_id}` | 查询分片变更预校验异步结果 |
| `CleanMigration` | PUT | `/v3/{project_id}/instances/{instance_id}/databases/{db_name}/migration/jobs/{job_id}/clean` | 清理分片变更 |
| `CompareParameterGroups` | PUT | `/v3/{project_id}/configurations/diff` | 比较参数组V3 |
| `CopyConfiguration` | POST | `/v3/{project_id}/configurations/{config_id}/copy` | 复制参数组V3 |
| `CreateDatabase` | POST | `/v1/{project_id}/instances/{instance_id}/databases` | 创建DDM逻辑库 |
| `CreateDdmConfigurations` | POST | `/v3/{project_id}/configurations` | 创建参数组 |
| `CreateDdmDatabase` | POST | `/v3/{project_id}/instances/{instance_id}/databases` | 创建DDM逻辑库 |
| `CreateDdmInstance` | POST | `/v3/{project_id}/instances` | 购买创建DDM实例 |
| `CreateDdmUser` | POST | `/v3/{project_id}/instances/{instance_id}/users` | 创建账号 |
| `CreateGroup` | POST | `/v3/{project_id}/instances/{instance_id}/groups` | 创建组 |
| `CreateInstance` | POST | `/v1/{project_id}/instances` | 购买DDM实例 |
| `CreateUsers` | POST | `/v1/{project_id}/instances/{instance_id}/users` | 创建DDM帐号 |
| `DeleteBackup` | DELETE | `/v3/{project_id}/backups/{backup_id}` | 删除备份 |
| `DeleteConfiguration` | DELETE | `/v3/{project_id}/configurations/{config_id}` | 删除参数组 |
| `DeleteDatabase` | DELETE | `/v1/{project_id}/instances/{instance_id}/databases/{ddm_dbname}` | 删除DDM逻辑库 |
| `DeleteDdmDatabase` | DELETE | `/v3/{project_id}/instances/{instance_id}/databases/{database_name}` | 删除逻辑库 |
| `DeleteDdmInstance` | DELETE | `/v3/{project_id}/instances/{instance_id}` | 删除DDM实例 |
| `DeleteDdmUser` | DELETE | `/v3/{project_id}/instances/{instance_id}/users/{username}` | 删除账号 |
| `DeleteGroup` | DELETE | `/v3/{project_id}/instances/{instance_id}/groups/{group_id}` | 删除实例组 |
| `DeleteInstance` | DELETE | `/v1/{project_id}/instances/{instance_id}` | 删除DDM实例 |
| `DeleteNodes` | DELETE | `/v3/{project_id}/instances/{instance_id}/nodes` | 删除实例的节点 |
| `DeleteUser` | DELETE | `/v1/{project_id}/instances/{instance_id}/users/{username}` | 删除DDM帐号 |

... and 94 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
