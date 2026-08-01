---
name: huaweicloud-apm
description: HuaweiCloud APM API guide. 45 APIs covering AKSK, ALARM, APM, CMDB, Profiling.
---

# HuaweiCloud APM API Guide

45 APIs. Tags: AKSK, ALARM, APM, CMDB, Profiling, REGION, TOPOLOGY, TRACING, TRANSACTION, VIEW

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ChangeAgentStatus` | POST | `/v1/apm2/openapi/apm-service/agent-mgr/change-status` | 更改实例的采集状态 |
| `CreateAkSk` | POST | `/v1/apm2/access-keys` | 创建aksk |
| `CreateBusiness` | POST | `/v1/apm2/openapi/tracing/business/create` | 创建链路追踪应用 |
| `DeleteAgent` | POST | `/v1/apm2/openapi/apm-service/agent-mgr/delete-agent` | 删除agent |
| `DeleteAkSk` | DELETE | `/v1/apm2/access-keys/{ak}` | 删除aksk |
| `DeleteApp` | DELETE | `/v1/apm2/openapi/cmdb/apps/delete-app/{application_id}` | 根据组件id删除指定的组件 |
| `ListAkSk` | GET | `/v1/apm2/openapi/systemmng/get-ak-sk-list` | 获取ak/sk |
| `ListAlarmData` | POST | `/v1/apm2/openapi/alarm/data/get-alarm-data-list` | 查询告警列表 |
| `ListAlarmNotify` | POST | `/v1/apm2/openapi/alarm/data/get-alarm-notify-list` | 查询告警消息列表 |
| `ListAppEnvs` | GET | `/v1/apm2/openapi/cmdb/envs/get-app-envs` | 获取组件下的环境列表 |
| `ListApps` | GET | `/v1/apm2/openapi/cmdb/apps/get-apps` | 获取组件列表 |
| `ListBusiness` | GET | `/v1/apm2/openapi/cmdb/business/get-business-list` | 查询应用列表 |
| `ListBusinessEnv` | POST | `/v1/apm2/openapi/transaction/business-env` | 查询URL跟踪Region环境列表 |
| `ListEnvInstances` | POST | `/v1/apm2/openapi/view/mainview/get-env-instance-list` | 获取实例信息列表 |
| `ListEnvMonitorItem` | POST | `/v1/apm2/openapi/apm-service/monitor-item-mgr/get-env-monitor-item-list` | 查询监控项列表 |
| `ListEnvTags` | POST | `/v1/apm2/openapi/cmdb/tag/get-env-tag-list` | 查询环境标签 |
| `ListOpenRegion` | GET | `/v1/apm2/openapi/region/get-opened-region` | 查询开通的region |
| `ListSupportedRegion` | GET | `/v1/apm2/openapi/region/get-all-supported-region` | 查询所有的支持的region |
| `SaveMonitorItemConfig` | POST | `/v1/apm2/openapi/apm-service/monitor-item-mgr/save-monitor-item-config` | 保存监控项 |
| `SearchAgent` | POST | `/v1/apm2/openapi/apm-service/agent-mgr/search` | 查询应用下所有探针 |
| `SearchApplication` | POST | `/v1/apm2/openapi/apm-service/app-mgr/search` | 对指定区域下的组件和环境及其探针情况进行搜索 |
| `SearchBusinessTopology` | POST | `/v1/apm2/openapi/topology/business-search` | 查询应用全局拓扑图 |
| `SearchEnvTopology` | POST | `/v1/apm2/openapi/topology/env-search` | 查询组件环境拓扑图 |
| `SearchTransaction` | POST | `/v1/apm2/openapi/transaction/search` | 查询URL跟踪视图列表 |
| `SearchTransactionConfig` | POST | `/v1/apm2/openapi/transaction/transaction-config-search` | 查询URL跟踪配置列表 |
| `ShowAccessPoint` | POST | `/v1/apm2/openapi/tracing/access/get-access-point/{business_id}` | 获取链路追踪应用接入地址 |
| `ShowAkSks` | GET | `/v1/apm2/access-keys` | 查询租户的aksk |
| `ShowBusinessDetail` | GET | `/v1/apm2/openapi/cmdb/business/get-business-detail/{business_id}` | 查询单个应用的详情 |
| `ShowClobDetail` | POST | `/v1/apm2/openapi/view/metric/get-clob-detail` | 获取原始数据详情 |
| `ShowEnvMonitorItems` | GET | `/v1/apm2/openapi/view/mainview/get-env-monitor-item-list` | 获取监控项信息 |

... and 15 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
