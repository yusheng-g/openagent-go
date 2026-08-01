---
name: huaweicloud-aom
description: HuaweiCloud AOM API guide. 99 APIs covering Prometheus实例, UniAgent管理, prometheus监控, 仪表盘, 告警.
---

# HuaweiCloud AOM API Guide

99 APIs. Tags: Prometheus实例, UniAgent管理, prometheus监控, 仪表盘, 告警, 应用资源管理(即将下线), 日志, 监控, 自动化运维(即将下线), 配置管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddActionRule` | POST | `/v2/{project_id}/alert/action-rules` | 新增告警行动规则 |
| `AddAlarmRule` | POST | `/v2/{project_id}/alarm-rules` | 添加阈值规则 |
| `AddEvent2alarmRule` | POST | `/v2/{project_id}/event2alarm-rule` | 新增一条事件类告警规则 |
| `AddMetricData` | POST | `/v1/{project_id}/ams/report/metricdata` | 添加监控数据 |
| `AddMuteRules` | POST | `/v2/{project_id}/alert/mute-rules` | 新增静默规则 |
| `AddOrUpdateMetricOrEventAlarmRule` | POST | `/v4/{project_id}/alarm-rules` | 添加或修改指标类或事件类告警规则 |
| `AddOrUpdateServiceDiscoveryRules` | PUT | `/v1/{project_id}/inv/servicediscoveryrules` | 添加或修改服务发现规则 |
| `BatchImportAgent` | POST | `/v1/{project_id}/uniagent-console/mainview/batch-import` | 下发批量安装UniAgent任务 |
| `BatchUpdateAgent` | POST | `/v1/{project_id}/uniagent-console/upgrade/batch-upgrade` | 下发批量升级UniAgent任务 |
| `BatchUpdateAlarmRule` | PUT | `/v4/{project_id}/alarm-rules/batch-update` | 批量更新Prometheus监控告警规则 |
| `CountEvents` | POST | `/v2/{project_id}/events/statistic` | 统计事件告警信息 |
| `CreateApp` | POST | `/v1/applications` | 新增应用 |
| `CreateComponent` | POST | `/v1/components` | 新增组件 |
| `CreateEnv` | POST | `/v1/environments` | 创建环境 |
| `CreateNotificationTemplate` | POST | `/v2/{project_id}/events/notification/templates` | 新增消息通知模板 |
| `CreatePromInstance` | POST | `/v1/{project_id}/aom/prometheus` | 新增Prometheus实例 |
| `CreateRecordingRule` | POST | `/v1/{project_id}/{prometheus_instance}/aom/api/v1/rules` | 创建Prometheus实例的预聚合规则 |
| `CreateSubApp` | POST | `/v1/sub-applications` | 新增子应用 |
| `CreateWorkflow` | POST | `/{project_id}/cms/workflow` | 创建任务 |
| `DeleteActionRule` | DELETE | `/v2/{project_id}/alert/action-rules` | 删除告警行动规则 |
| `DeleteAlarmRule` | DELETE | `/v2/{project_id}/alarm-rules/{alarm_rule_id}` | 删除阈值规则 |
| `DeleteAlarmRules` | POST | `/v2/{project_id}/alarm-rules/delete` | 批量删除阈值规则 |
| `DeleteAlarmRuleTemplate` | DELETE | `/v4/{project_id}/alarm-rules-template` | 删除告警模板 |
| `DeleteApp` | DELETE | `/v1/applications/{application_id}` | 删除应用 |
| `DeleteComponent` | DELETE | `/v1/components/{component_id}` | 删除组件 |
| `DeleteDashboard` | DELETE | `/v2/{project_id}/aom/dashboards/{dashboard_id}` | 删除仪表盘 |
| `DeleteDashboardsFolder` | DELETE | `/v2/{project_id}/aom/dashboards-folder/{folder_id}` | 删除仪表盘分组 |
| `DeleteEnv` | DELETE | `/v1/environments/{environment_id}` | 删除环境 |
| `DeleteEvent2alarmRule` | DELETE | `/v2/{project_id}/event2alarm-rule` | 删除事件类告警规则 |
| `DeleteMetricOrEventAlarmRule` | DELETE | `/v4/{project_id}/alarm-rules` | 删除指标类或事件类告警规则 |

... and 69 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
