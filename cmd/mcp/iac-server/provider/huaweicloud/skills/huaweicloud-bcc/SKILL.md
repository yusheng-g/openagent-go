---
name: huaweicloud-bcc
description: HuaweiCloud BCC API guide. 59 APIs covering 任务管理, 副本, 合规规则, 告警管理, 存储库.
---

# HuaweiCloud BCC API Guide

59 APIs. Tags: 任务管理, 副本, 合规规则, 告警管理, 存储库, 报告管理, 报告配置, 支持区域列表, 模板, 用户管理, 监控, 策略, 组织策略, 资源定级, 资源管理, 配置保护

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BindResourceLevelCompliance` | PUT | `/v1/{domain_id}/resource-levels/{level_id}/compliance` | 绑定资源等级合规规则 |
| `ChangeResourcesLevel` | PUT | `/v1/{domain_id}/resources/resource-level` | 指定资源等级 |
| `CreateComplianceRule` | POST | `/v1/{domain_id}/backup/compliancerules` | 新建自定义合规规则 |
| `CreatePolicy` | POST | `/v1/{domain_id}/backup/policies` | 新建策略 |
| `CreateReportSetting` | POST | `/v1/{domain_id}/report/settings` | 创建报告配置 |
| `CreateResourcesLevel` | POST | `/v1/{domain_id}/resource-levels` | 新增资源分级 |
| `CreateTemplate` | POST | `/v1/{domain_id}/templates` | 创建模板 |
| `DeleteComplianceRule` | DELETE | `/v1/{domain_id}/backup/compliancerules/{compliance_id}` | 删除自定义合规规则 |
| `DeletePolicy` | DELETE | `/v1/{domain_id}/backup/policies/{policy_id}` | 删除指定策略 |
| `DeleteReport` | DELETE | `/v1/{domain_id}/reports/{report_id}` | 删除指定的报告 |
| `DeleteReportSetting` | DELETE | `/v1/{domain_id}/report/settings/{setting_id}` | 删除报告配置 |
| `DeleteTemplate` | DELETE | `/v1/{domain_id}/templates/{template_id}` | 删除模板 |
| `EnableDomain` | POST | `/v1/{domain_id}/enable` | 用户授权开启BCC |
| `ListAlarmRules` | GET | `/v1/{domain_id}/alarm-rules` | 查询告警规则列表 |
| `ListAlarms` | GET | `/v1/{domain_id}/alarms` | 查询告警列表 |
| `ListComplianceRule` | GET | `/v1/{domain_id}/backup/compliancerules` | 列举合规规则 |
| `ListEvents` | GET | `/v1/{domain_id}/events` | 查询事件数据 |
| `ListMetrics` | GET | `/v1/{domain_id}/metrics` | 查询监控数据 |
| `ListOrganizationPolicy` | GET | `/v1/{domain_id}/backup/organizationpolicies` | 列举策略 |
| `ListPolicy` | GET | `/v1/{domain_id}/backup/policies` | 列举策略 |
| `ListReports` | GET | `/v1/{domain_id}/reports` | 查询报告列表 |
| `ListReportSettings` | GET | `/v1/{domain_id}/report/settings` | 查询报告配置列表 |
| `ListResourceCopies` | GET | `/v1/{domain_id}/copies` | 查询副本列表 |
| `ListResources` | GET | `/v1/{domain_id}/resources` | 查询资源列表 |
| `ListResourcesLevel` | GET | `/v1/{domain_id}/resource-levels` | 列举资源分级 |
| `ListResourcesLevelTags` | GET | `/v1/{domain_id}/resource-levels/tags` | 列举资源分级已指定的标签 |
| `ListSupportedRegion` | GET | `/v1/bcc/regions` | 查询支持的region列表 |
| `ListTasks` | GET | `/v1/{domain_id}/tasks` | 查询任务列表 |
| `ListTemplates` | GET | `/v1/{domain_id}/templates` | 查询模板列表 |
| `ListVault` | GET | `/v1/{domain_id}/backup/vaults` | 列举存储库 |

... and 29 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
