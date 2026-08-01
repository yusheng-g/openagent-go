---
name: huaweicloud-ces
description: HuaweiCloud CES API guide. 72 APIs covering Agent任务相关接口, 一键告警, 事件监控, 告警模板, 告警模板关联告警规则.
---

# HuaweiCloud CES API Guide

72 APIs. Tags: Agent任务相关接口, 一键告警, 事件监控, 告警模板, 告警模板关联告警规则, 告警策略, 告警规则, 告警规则管理, 告警记录, 告警资源, 告警通知, 告警通知屏蔽, 指标管理, 插件状态查询, 监控数据管理, 监控看板, 监控视图, 资源分组, 资源分组关联资源, 资源分组管理, 资源标签管理, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddAlarmRuleResources` | POST | `/v2/{project_id}/alarms/{alarm_id}/resources/batch-create` | 批量增加告警规则资源 |
| `BatchCreateAgentInvocations` | POST | `/v3/{project_id}/agent-invocations/batch-create` | 批量创建Agent任务 |
| `BatchCreateResources` | POST | `/v2/{project_id}/resource-groups/{group_id}/resources/batch-create` | 自定义资源分组批量增加关联资源 |
| `BatchDeleteAlarmRules` | POST | `/v2/{project_id}/alarms/batch-delete` | 批量删除告警规则 |
| `BatchDeleteAlarmTemplates` | POST | `/v2/{project_id}/alarm-templates/batch-delete` | 批量删除自定义告警模板 |
| `BatchDeleteNotificationMasks` | POST | `/v2/{project_id}/notification-masks/batch-delete` | 批量删除告警通知屏蔽规则 |
| `BatchDeleteOneClickAlarms` | POST | `/v2/{project_id}/one-click-alarms/batch-delete` | 批量删除一键告警 |
| `BatchDeleteResourceGroups` | POST | `/v2/{project_id}/resource-groups/batch-delete` | 批量删除资源分组 |
| `BatchDeleteResources` | POST | `/v2/{project_id}/resource-groups/{group_id}/resources/batch-delete` | 自定义资源分组批量删除关联资源 |
| `BatchEnableAlarmRules` | POST | `/v2/{project_id}/alarms/action` | 批量启停告警规则 |
| `BatchListMetricData` | POST | `/V1.0/{project_id}/batch-query-metric-data` | 批量查询监控数据 |
| `BatchListSpecifiedMetricData` | POST | `/v2/{project_id}/batch-query-metric-data` | 批量查询指标数据 |
| `BatchUpdateNotificationMasks` | PUT | `/v2/{project_id}/notification-masks` | 批量设置告警通知屏蔽规则 |
| `BatchUpdateNotificationMaskTime` | POST | `/v2/{project_id}/notification-masks/batch-update` | 批量修改告警通知屏蔽规则的屏蔽时间 |
| `BatchUpdateOneClickAlarmPoliciesEnabledState` | PUT | `/v2/{project_id}/one-click-alarms/{one_click_alarm_id}/alarms/{alarm_id}/policies/action` | 批量修改一键告警关联告警规则策略的启用状态 |
| `BatchUpdateOneClickAlarmsEnabledState` | PUT | `/v2/{project_id}/one-click-alarms/{one_click_alarm_id}/alarm-rules/action` | 批量修改一键告警关联告警规则的启用状态 |
| `BatchUpdateWidgets` | POST | `/v2/{project_id}/widgets/batch-update` | 批量更新监控视图 |
| `CreateAlarm` | POST | `/V1.0/{project_id}/alarms` | 创建告警规则(V1) |
| `CreateAlarmRules` | POST | `/v2/{project_id}/alarms` | 创建告警规则(推荐) |
| `CreateAlarmTemplate` | POST | `/V1.0/{project_id}/alarm-template` | 创建自定义告警模板 |
| `CreateDashboardWidgets` | POST | `/v2/{project_id}/dashboards/{dashboard_id}/widgets` | 创建/复制/批量创建监控视图到指定的监控看板 |
| `CreateEvents` | POST | `/V1.0/{project_id}/events` | 上报事件 |
| `CreateMetricData` | POST | `/V1.0/{project_id}/metric-data` | 添加监控数据 |
| `CreateOneClickAlarm` | POST | `/v2/{project_id}/one-click-alarms` | 创建一键告警 |
| `CreateOneDashboard` | POST | `/v2/{project_id}/dashboards` | 创建/复制监控看板 |
| `CreateResourceGroup` | POST | `/V1.0/{project_id}/resource-groups` | 创建资源分组 |
| `DeleteAlarm` | DELETE | `/V1.0/{project_id}/alarms/{alarm_id}` | 删除告警规则 |
| `DeleteAlarmRuleResources` | POST | `/v2/{project_id}/alarms/{alarm_id}/resources/batch-delete` | 批量删除告警规则资源 |
| `DeleteAlarmTemplate` | DELETE | `/V1.0/{project_id}/alarm-template/{template_id}` | 删除自定义告警模板 |
| `DeleteDashboards` | POST | `/v2/{project_id}/dashboards/batch-delete` | 批量删除监控看板 |

... and 42 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
