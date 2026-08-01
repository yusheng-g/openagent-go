---
name: huaweicloud-das
description: HuaweiCloud DAS API guide. 108 APIs covering 云DBA, 开发工具, 获取API版本.
---

# HuaweiCloud DAS API Guide

108 APIs. Tags: 云DBA, 开发工具, 获取API版本

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddEmailTemplate` | POST | `/v3/{project_id}/batch-inspection/email-template` | 新增邮件模板 |
| `AddFullSqlTask` | POST | `/v3/{project_id}/instances/{instance_id}/full-sql/add-task` | 创建全量SQL明细解析任务 |
| `AddInstanceGroup` | POST | `/v3/{project_id}/batch-inspection/instance-group` | 新增实例组 |
| `AddInstanceToGroup` | POST | `/v3/{project_id}/batch-inspection/add-instance-to-group` | 将实例添加到实例组 |
| `BatchSendEmail` | POST | `/v3/{project_id}/batch-inspection/batch-send-email` | 批量发送邮件 |
| `BatchSubscribeReport` | POST | `/v3/{project_id}/batch-inspection/batch-subscribe` | 批量订阅/取消订阅 |
| `CancelShareConnections` | DELETE | `/v3/{project_id}/connections/share` | 删除共享链接 |
| `ChangeChargeMode` | POST | `/v3/{project_id}/cloud-dba/change-payment-mode` | 设置付费模式 |
| `ChangeFullDeadLockSwitch` | POST | `/v3/{project_id}/instances/{instance_id}/set-fulldeadlock-switch` | 设置全量死锁开关 |
| `ChangeSqlLimitSwitchStatus` | POST | `/v3/{project_id}/instances/{instance_id}/sql-limit/switch` | 设置SQL限流开关状态 |
| `ChangeSqlSwitch` | POST | `/v3/{project_id}/instances/{instance_id}/sql/switch` | 开启/关闭全量SQL、慢SQL开关 |
| `ChangeTransactionSwitchStatus` | POST | `/v3/{project_id}/instances/{instance_id}/transaction/switch` | 开启/关闭历史事务开关 |
| `CheckCredential` | POST | `/v3/{project_id}/instances/{instance_id}/health-report/check-credential` | 测试AK/SK |
| `CheckCredentialForBatchInspection` | POST | `/v3/{project_id}/batch-inspection/check-credential` | 测试AK/SK |
| `CreateHealthReportTask` | POST | `/v3/{project_id}/instances/{instance_id}/create-instance-health-report-task` | 创建实例健康诊断任务 |
| `CreateHistoryTransactionExportTask` | POST | `/v3/{project_id}/transaction/{instance_id}/create-export-task` | 创建导出历史事务任务 |
| `CreateInstanceConnection` | POST | `/v3/{project_id}/instances/{instance_id}/create-connection` | 创建实例连接 |
| `CreateShareConnections` | POST | `/v3/{project_id}/connections/share` | 设置共享链接 |
| `CreateSnapshots` | POST | `/v3/{project_id}/connections/{connection_id}/instance/create-snapshot` | 创建快照 |
| `CreateSpaceAnalysisTask` | POST | `/v3/{project_id}/instances/{instance_id}/space-analysis` | 创建空间分析任务 |
| `CreateSqlLimitRules` | POST | `/v3/{project_id}/instances/{instance_id}/sql-limit/rules` | 创建SQL限流规则 |
| `CreateTuning` | POST | `/v3/{project_id}/connections/{connection_id}/tuning/create-tuning` | 执行SQL诊断 |
| `DeleteDbUser` | DELETE | `/v3/{project_id}/instances/{instance_id}/db-users/{db_user_id}` | 删除数据库用户 |
| `DeleteEmailTemplate` | DELETE | `/v3/{project_id}/batch-inspection/email-template` | 删除邮件模板 |
| `DeleteHistoryTransactionExportTask` | POST | `/v3/{project_id}/transaction/{instance_id}/delete-export-task` | 删除导出历史事务任务 |
| `DeleteInstanceGroup` | DELETE | `/v3/{project_id}/batch-inspection/instance-group` | 删除实例组 |
| `DeleteProcess` | DELETE | `/v3/{project_id}/instances/{instance_id}/process` | 查杀会话 |
| `DeleteSqlLimitRules` | DELETE | `/v3/{project_id}/instances/{instance_id}/sql-limit/rules` | 删除SQL限流规则 |
| `ExportFullSqlDetails` | GET | `/v3/{project_id}/instances/{instance_id}/full-sql-search` | 导出全量SQL明细 |
| `ExportSlowQueryLogs` | GET | `/v3/{project_id}/instances/{instance_id}/slow-query-logs` | 导出慢SQL数据 |

... and 78 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
