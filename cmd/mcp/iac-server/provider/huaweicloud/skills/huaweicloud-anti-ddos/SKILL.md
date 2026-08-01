---
name: huaweicloud-anti-ddos
description: HuaweiCloud Anti-DDoS API guide. 18 APIs covering DDoS任务管理, DDoS防护管理, 告警配置管理, 默认防护策略管理.
---

# HuaweiCloud Anti-DDoS API Guide

18 APIs. Tags: DDoS任务管理, DDoS防护管理, 告警配置管理, 默认防护策略管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateDefaultConfig` | POST | `/v1/{project_id}/antiddos/default-config` | 配置Anti-DDoS默认防护策略 |
| `DeleteDefaultConfig` | DELETE | `/v1/{project_id}/antiddos/default-config` | 删除Ani-DDoS默认防护策略 |
| `EnableDefensePolicy` | POST | `/v1/{project_id}/antiddos/{floating_ip_id}` | 开通DDoS服务 |
| `ListDailyLog` | GET | `/v1/{project_id}/antiddos/{floating_ip_id}/logs` | 查询指定EIP异常事件 |
| `ListDailyReport` | GET | `/v1/{project_id}/antiddos/{floating_ip_id}/daily` | 查询指定EIP防护流量 |
| `ListDDosStatus` | GET | `/v1/{project_id}/antiddos` | 查询EIP防护状态列表 |
| `ListNewConfigs` | GET | `/v2/{project_id}/antiddos/query-config-list` | 查询Anti-DDoS配置可选范围 |
| `ListQuota` | GET | `/v1/{project_id}/antiddos/quotas` | 查询配额 |
| `ListWeeklyReports` | GET | `/v1/{project_id}/antiddos/weekly` | 查询周防护统计情况 |
| `ShowAlertConfig` | GET | `/v2/{project_id}/warnalert/alertconfig/query` | 查询告警配置信息 |
| `ShowDDos` | GET | `/v1/{project_id}/antiddos/{floating_ip_id}` | 查询Anti-DDoS服务 |
| `ShowDDosStatus` | GET | `/v1/{project_id}/antiddos/{floating_ip_id}/status` | 查询指定EIP防护状态 |
| `ShowDefaultConfig` | GET | `/v1/{project_id}/antiddos/default-config` | 查询Ani-DDoS默认防护策略 |
| `ShowLogConfig` | GET | `/v1/{project_id}/antiddos/lts-config` | 查询全量日志设置 |
| `ShowNewTaskStatus` | GET | `/v2/{project_id}/query-task-status` | 查询Anti-DDoS任务 |
| `UpdateAlertConfig` | POST | `/v2/{project_id}/warnalert/alertconfig/update` | 更新告警配置信息 |
| `UpdateDDos` | PUT | `/v1/{project_id}/antiddos/{floating_ip_id}` | 更新Anti-DDoS服务 |
| `UpdateLogConfig` | PUT | `/v1/{project_id}/antiddos/lts-config` | 更新用户全量日志设置 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
