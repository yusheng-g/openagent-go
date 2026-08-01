---
name: huaweicloud-lts
description: HuaweiCloud LTS API guide. 82 APIs covering AOM容器日志接入LTS, SQL告警规则, 主机组管理, 仪表盘管理, 关键词告警规则.
---

# HuaweiCloud LTS API Guide

82 APIs. Tags: AOM容器日志接入LTS, SQL告警规则, 主机组管理, 仪表盘管理, 关键词告警规则, 告警主题, 告警列表, 多账号日志汇聚, 快速查询, 日志接入, 日志流图表, 日志流管理, 日志管理, 日志组管理, 日志转储, 标签管理, 消息模板管理, 结构化配置, 超额采集

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateAccessConfig` | POST | `/v3/{project_id}/lts/access-config` | 创建日志接入 |
| `CreateAgencyAccess` | POST | `/v2.0/{project_id}/lts/createAgencyAccess` | 新建跨账号日志接入 |
| `CreateAomMappingRules` | POST | `/v2/{project_id}/lts/aom-mapping` | 创建接入规则 |
| `CreateDashBoard` | POST | `/v2/{project_id}/dashboard` | 创建仪表盘 |
| `CreateDashboardGroup` | POST | `/v2/{project_id}/lts/dashboard-group` | 创建仪表盘分组 |
| `Createfavorite` | POST | `/v1.0/{project_id}/lts/favorite` | 创建日志收藏 |
| `CreateHostGroup` | POST | `/v3/{project_id}/lts/host-group` | 创建主机组 |
| `CreateKeywordsAlarmRule` | POST | `/v2/{project_id}/lts/alarms/keywords-alarm-rule` | 创建关键词告警规则 |
| `CreateLogGroup` | POST | `/v2/{project_id}/groups` | 创建日志组 |
| `CreateLogStream` | POST | `/v2/{project_id}/groups/{log_group_id}/streams` | 创建日志流 |
| `CreateLogStreamIndex` | POST | `/v1.0/{project_id}/groups/{group_id}/stream/{stream_id}/index/config` | 向指定流创建索引 |
| `CreateNotificationTemplate` | POST | `/v2/{project_id}/{domain_id}/lts/events/notification/templates` | 创建消息模板 |
| `CreateSearchCriterias` | POST | `/v1.0/{project_id}/groups/{group_id}/topics/{topic_id}/search-criterias` | 添加快速查询 |
| `CreateSqlAlarmRule` | POST | `/v2/{project_id}/lts/alarms/sql-alarm-rule` | 创建SQL告警规则 |
| `CreateStructConfig` | POST | `/v3/{project_id}/lts/struct/template` | 通过结构化模板创建结构化配置(新) |
| `CreateStructTemplate` | POST | `/v2/{project_id}/lts/struct/template` | 创建结构化配置 |
| `CreateTags` | POST | `/v1/{project_id}/{resource_type}/{resource_id}/tags/action` | None |
| `CreateTransfer` | POST | `/v2/{project_id}/transfers` | 创建日志转储(新版) |
| `DeleteAccessConfig` | DELETE | `/v3/{project_id}/lts/access-config` | 删除日志接入 |
| `DeleteActiveAlarms` | POST | `/v2/{project_id}/{domain_id}/lts/alarms/sql-alarm/clear` | 删除活动告警 |
| `DeleteAomMappingRules` | DELETE | `/v2/{project_id}/lts/aom-mapping` | 删除接入规则 |
| `Deletefavorite` | DELETE | `/v1.0/{project_id}/lts/favorite/{fav_res_id}` | 取消收藏 |
| `DeleteHostGroup` | DELETE | `/v3/{project_id}/lts/host-group` | 删除主机组 |
| `DeleteKeywordsAlarmRule` | DELETE | `/v2/{project_id}/lts/alarms/keywords-alarm-rule/{keywords_alarm_rule_id}` | 删除关键词告警规则 |
| `DeleteLogGroup` | DELETE | `/v2/{project_id}/groups/{log_group_id}` | 删除日志组 |
| `DeleteLogStream` | DELETE | `/v2/{project_id}/groups/{log_group_id}/streams/{log_stream_id}` | 删除日志流 |
| `DeleteNotificationTemplate` | DELETE | `/v2/{project_id}/{domain_id}/lts/events/notification/templates` | 删除消息模板 |
| `DeleteSearchCriterias` | DELETE | `/v1.0/{project_id}/groups/{group_id}/topics/{topic_id}/search-criterias` | 删除快速查询 |
| `DeleteSqlAlarmRule` | DELETE | `/v2/{project_id}/lts/alarms/sql-alarm-rule/{sql_alarm_rule_id}` | 删除SQL告警规则 |
| `DeleteStructTemplate` | DELETE | `/v2/{project_id}/lts/struct/template` | 删除结构化配置 |

... and 52 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
