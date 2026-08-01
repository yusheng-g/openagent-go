---
name: huaweicloud-dbss
description: HuaweiCloud DBSS API guide. 173 APIs covering SQL白名单功能, TMS标签, 加密增强管理, 告警信息, 备份管理.
---

# HuaweiCloud DBSS API Guide

173 APIs. Tags: SQL白名单功能, TMS标签, 加密增强管理, 告警信息, 备份管理, 审计Agent, 审计实例, 审计数据库, 审计规则-SQL注入, 审计规则-审计范围, 审计规则-隐私数据保护, 审计规则-风险操作, 待下线接口, 总览, 报表管理, 数据报表, 管理侧查询, 运维增强管理, 风险导出

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddAuditAgent` | POST | `/v2/{project_id}/{instance_id}/audit/agents` | 添加审计数据库Agent[待下线] |
| `AddAuditAgentNew` | POST | `/v3/{project_id}/audit/{instance_id}/agents` | 添加审计数据库Agent |
| `AddAuditDatabase` | POST | `/v1/{project_id}/{instance_id}/audit/databases` | 添加自建数据库[待下线] |
| `AddAuditDatabaseNew` | POST | `/v2/{project_id}/audit/{instance_id}/databases` | 添加自建数据库 |
| `AddDatabaseSslKey` | POST | `/v2/{project_id}/audit/{instance_id}/databases/{db_id}/sslkey` | 上传/更新数据库私钥 |
| `AddRdsDatabase` | POST | `/v2/{project_id}/{instance_id}/audit/databases/rds` | 添加RDS数据库[待下线] |
| `AddRdsDatabaseNew` | POST | `/v3/{project_id}/audit/{instance_id}/databases/rds` | 添加RDS数据库 |
| `AddRdsNoAgentDatabase` | POST | `/v1/{project_id}/{instance_id}/dbss/audit/databases/rds` | 添加RDS数据库[待下线] |
| `BatchAddAuditWhitelist` | POST | `/v1/{project_id}/audit/{instance_id}/whitelists` | 批量添加白名单 |
| `BatchAddResourceTag` | POST | `/v1/{project_id}/{resource_type}/{resource_id}/tags/create` | 批量添加资源标签 |
| `BatchDeleteAuditScope` | POST | `/v1/{project_id}/audit/{instance_id}/rule/scopes/batch-delete` | 审计范围规则操作-删除策略 |
| `BatchDeleteResourceTag` | DELETE | `/v1/{project_id}/{resource_type}/{resource_id}/tags/delete` | 批量删除资源标签 |
| `BatchSetAuditAlarmLogStatus` | POST | `/v1/{project_id}/audit/{instance_id}/alarm-log/mark` | 批量标记 |
| `BindDbEncryptEip` | POST | `/v1/{project_id}/db-encrypt/{instance_id}/eip/bind` | 绑定数据库加密实例的EIP |
| `BindDbOmEip` | POST | `/v1/{project_id}/db-om/{instance_id}/eip/bind` | 绑定数据库运维实例的EIP |
| `ChangeDbEncryptSecurityGroup` | PUT | `/v1/{project_id}/db-encrypt/{instance_id}/security-group` | 更改数据库加密实例的安全组 |
| `ChangeDbOmSecurityGroup` | PUT | `/v1/{project_id}/db-om/{instance_id}/security-group` | 更改数据库运维实例的安全组 |
| `ConfirmUpgradeAudit` | POST | `/v1/{project_id}/audit/{resource_id}/upgrade` | 触发审计实例升级 |
| `CountDbAccountSession` | POST | `/v1/{project_id}/audit/{instance_id}/statistics/session/db-account` | 查询数据库用户会话分布 |
| `CountDbClientSession` | POST | `/v1/{project_id}/audit/{instance_id}/statistics/session/db-client` | 查询客户端会话分布 |
| `CountInjectionStatistics` | POST | `/v1/{project_id}/audit/{instance_id}/statistics/trend/sql-injection` | 获取指定时间段内的sql注入分布统计 |
| `CountOperationStatistics` | POST | `/v1/{project_id}/audit/{instance_id}/statistics/trend/risk-operation` | 获取指定时间段内的风险操作数量分布统计 |
| `CountResourceInstanceByTag` | POST | `/v1/{project_id}/{resource_type}/resource-instances/count` | 根据标签查询资源实例数量 |
| `CountRiskTrendStatistics` | POST | `/v1/{project_id}/audit/{instance_id}/statistics/trend/risk-level` | 获取指定时间段内的风险分布统计 |
| `CountSessionStatistics` | POST | `/v1/{project_id}/audit/{instance_id}/statistics/trend/session` | 获取指定时间段内的查询会话统计 |
| `CountSqlStatistics` | POST | `/v1/{project_id}/audit/{instance_id}/statistics/trend/sql-type` | 获取指定时间段内的SQL分布统计 |
| `CountSqlTrendStatistics` | POST | `/v1/{project_id}/audit/{instance_id}/statistics/trend/sql-count` | 获取指定时间段内的sql数量分布统计 |
| `CreateAuditDbAgent` | POST | `/v2/{project_id}/audit/{instance_id}/agents/{agent_id}` | 指定agent_id方式添加agent |
| `CreateAuditRiskRule` | POST | `/v1/{project_id}/audit/{instance_id}/rule/risk` | 添加风险规则 |
| `CreateAuditScopeRule` | POST | `/v1/{project_id}/audit/{instance_id}/rule/scopes` | 添加审计范围策略 |

... and 143 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
