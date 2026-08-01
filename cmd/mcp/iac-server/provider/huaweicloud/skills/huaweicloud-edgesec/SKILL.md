---
name: huaweicloud-edgesec
description: HuaweiCloud EdgeSec API guide. 60 APIs covering Ddos攻击日志, Ddos统计, Http统计, Http防护策略管理, Http防护规则管理-CC.
---

# HuaweiCloud EdgeSec API Guide

60 APIs. Tags: Ddos攻击日志, Ddos统计, Http统计, Http防护策略管理, Http防护规则管理-CC, Http防护规则管理-IP黑白名单, Http防护规则管理-地理位置, Http防护规则管理-攻击惩罚, Http防护规则管理-精准防护, Http防护规则管理-误报屏蔽, Ip地址组管理, 安全总览, 引用表管理, 防护域名管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ApplyHttpPolicy` | POST | `/v1/edgesec/configuration/http/policies/{policy_id}/hosts` | 更新防护策略的域名 |
| `CreateDomain` | POST | `/v1/edgesec/configuration/domains` | 创建防护域名 |
| `CreateHttpAccessControlRule` | POST | `/v1/edgesec/configuration/http/policies/{policy_id}/access-control-rule` | 创建精准防护规则 |
| `CreateHttpBlockTrustIpRule` | POST | `/v1/edgesec/configuration/http/policies/{policy_id}/blocktrustip-rule` | 创建IP黑白名单规则 |
| `CreateHttpCcRule` | POST | `/v1/edgesec/configuration/http/policies/{policy_id}/cc-rule` | 创建cc规则 |
| `CreateHttpGeoIpRule` | POST | `/v1/edgesec/configuration/http/policies/{policy_id}/geoip-rule` | 创建地理位置规则 |
| `CreateHttpIgnoreRule` | POST | `/v1/edgesec/configuration/http/policies/{policy_id}/ignore-rule` | 创建误报屏蔽规则 |
| `CreateHttpIpGroup` | POST | `/v1/edgesec/configuration/http/ip-groups` | 创建IP地址组 |
| `CreateHttpPolicy` | POST | `/v1/edgesec/configuration/http/policies` | 创建防护策略 |
| `CreateHttpPunishmentRule` | POST | `/v1/edgesec/configuration/http/policies/{policy_id}/punishment-rule` | 创建攻击惩罚规则 |
| `CreateHttpReferenceTable` | POST | `/v1/edgesec/configuration/http/reference-table` | 创建引用表 |
| `DeleteDomain` | DELETE | `/v1/edgesec/configuration/domains/{domain_id}` | 删除防护域名 |
| `DeleteHttpAccessControlRule` | DELETE | `/v1/edgesec/configuration/http/policies/{policy_id}/access-control-rule/{rule_id}` | 删除精准防护规则 |
| `DeleteHttpBlockTrustIpRule` | DELETE | `/v1/edgesec/configuration/http/policies/{policy_id}/blocktrustip-rule/{rule_id}` | 删除IP黑白名单规则 |
| `DeleteHttpCcRule` | DELETE | `/v1/edgesec/configuration/http/policies/{policy_id}/cc-rule/{rule_id}` | 删除cc规则 |
| `DeleteHttpGeoIpRule` | DELETE | `/v1/edgesec/configuration/http/policies/{policy_id}/geoip-rule/{rule_id}` | 删除地理位置规则 |
| `DeleteHttpIgnoreRule` | DELETE | `/v1/edgesec/configuration/http/policies/{policy_id}/ignore-rule/{rule_id}` | 删除误报屏蔽规则 |
| `DeleteHttpIpGroup` | DELETE | `/v1/edgesec/configuration/http/ip-groups/{ip_group_id}` | 删除IP地址组 |
| `DeleteHttpPolicy` | DELETE | `/v1/edgesec/configuration/http/policies/{policy_id}` | 删除防护策略 |
| `DeleteHttpPunishmentRule` | DELETE | `/v1/edgesec/configuration/http/policies/{policy_id}/punishment-rule/{rule_id}` | 删除攻击惩罚规则 |
| `DeleteHttpReferenceTable` | DELETE | `/v1/edgesec/configuration/http/reference-table/{table_id}` | 删除引用表 |
| `DownloadDdosAttackLogs` | POST | `/v1/edgesec/log/ddos-attack-logs/download` | Ddos攻击日志下载 |
| `ResetHttpIgnoreRule` | POST | `/v1/edgesec/configuration/http/policies/{policy_id}/ignore-rule/{rule_id}/recount` | 重置误报屏蔽规则 |
| `ShowDdosAttackLogs` | GET | `/v1/edgesec/log/ddos-attack-logs` | 查询ddos攻击日志列表 |
| `ShowDdosAttackTimelineStats` | GET | `/v1/edgesec/stat/ddos-attack-timelines` | 查询DDoS攻击统计时间线数据 |
| `ShowDomainDetail` | GET | `/v1/edgesec/configuration/domains/{domain_id}` | 查询防护域名详情 |
| `ShowDomains` | GET | `/v1/edgesec/configuration/domains` | 查询防护域名列表 |
| `ShowHttpAccessControlRule` | GET | `/v1/edgesec/configuration/http/policies/{policy_id}/access-control-rule/{rule_id}` | 查询精准防护规则 |
| `ShowHttpAccessControlRules` | GET | `/v1/edgesec/configuration/http/policies/{policy_id}/access-control-rule` | 查询精准防护规则列表 |
| `ShowHttpAttackDistributionStats` | GET | `/v1/edgesec/stat/http-attack-distribution` | 查询HTTP攻击分布数据 |

... and 30 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
