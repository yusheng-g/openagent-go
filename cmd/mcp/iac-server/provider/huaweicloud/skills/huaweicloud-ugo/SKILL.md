---
name: huaweicloud-ugo
description: HuaweiCloud UGO API guide. 22 APIs covering API版本管理, SQL语句转换, 评估项目, 迁移项目, 配额管理.
---

# HuaweiCloud UGO API Guide

22 APIs. Tags: API版本管理, SQL语句转换, 评估项目, 迁移项目, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CheckPermission` | POST | `/v1/{project_id}/migration-projects/{migration_project_id}/permission-check` | 目标库权限检查。 |
| `CommitSyntaxConversion` | POST | `/v1/{project_id}/migration-projects/{migration_project_id}/syntax-conversion` | 提交语法转换 |
| `CommitVerification` | POST | `/v1/{project_id}/migration-projects/{migration_project_id}/verification` | 提交验证。 |
| `ConfirmTargetDbType` | POST | `/v1/{project_id}/evaluation-projects/target-confirmation` | 评估项目确认目标数据库类型。 |
| `CreateEvaluationProject` | POST | `/v1/{project_id}/evaluation-projects` | 创建评估项目。 |
| `CreateMigrationProject` | POST | `/v1/{project_id}/migration-projects` | 创建迁移项目。 |
| `DeleteEvaluationProject` | DELETE | `/v1/{project_id}/evaluation-projects/{evaluation_project_id}` | 删除评估项目。 |
| `DeleteMigrationProject` | DELETE | `/v1/{project_id}/migration-projects/{migration_project_id}` | 删除迁移项目 |
| `DownloadFailureReport` | GET | `/v1/{project_id}/migration-projects/{migration_project_id}/download-failure-report` | 下载迁移错误报告。 |
| `ListApiVersions` | GET | `/` | 查询API版本信息列表。 |
| `ListEvaluationProjects` | GET | `/v1/{project_id}/evaluation-projects` | 查询评估项目列表。 |
| `ListMigrationProjects` | GET | `/v1/{project_id}/migration-projects` | 查询迁移项目列表。 |
| `ListPermissionCheckResult` | GET | `/v1/{project_id}/migration-projects/{migration_project_id}/permission-result` | 查询权限检查结果。 |
| `ListQuotas` | GET | `/v1/{project_id}/quotas` | 查询配额。 |
| `ListSyntaxConversionProgress` | GET | `/v1/{project_id}/migration-projects/{migration_project_id}/syntax-conversion-progress` | 查询语法转换的进度。 |
| `ListVerificationProgress` | GET | `/v1/{project_id}/migration-projects/{migration_project_id}/verification-progress` | 查询验证进度。 |
| `RunSqlConversion` | POST | `/v1/{project_id}/sql-conversion` | SQL语句转换。 |
| `ShowApiVersionInfo` | GET | `/{api_version}` | 查询指定版本号的API版本信息。 |
| `ShowEvaluationProjectDetail` | GET | `/v1/{project_id}/evaluation-projects/{evaluation_project_id}/detail` | 查询评估项目详情。 |
| `ShowEvaluationProjectStatus` | GET | `/v1/{project_id}/evaluation-projects/{evaluation_project_id}/status` | 查询评估项目状态。 |
| `ShowMigrationProjectDetail` | GET | `/v1/{project_id}/migration-projects/{migration_project_id}/detail` | 查询迁移项目详情。 |
| `ShowMigrationProjectStatus` | GET | `/v1/{project_id}/migration-projects/{migration_project_id}/status` | 查询迁移项目状态。 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
