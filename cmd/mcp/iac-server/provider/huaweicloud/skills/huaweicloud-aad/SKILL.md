---
name: huaweicloud-aad
description: HuaweiCloud AAD API guide. 71 APIs covering DDoS原生高级防护-告警配置管理, DDoS原生高级防护-实例管理, DDoS原生高级防护-概览管理, DDoS原生高级防护-策略管理, DDoS原生高级防护-防护对象管理.
---

# HuaweiCloud AAD API Guide

71 APIs. Tags: DDoS原生高级防护-告警配置管理, DDoS原生高级防护-实例管理, DDoS原生高级防护-概览管理, DDoS原生高级防护-策略管理, DDoS原生高级防护-防护对象管理, DDoS高防-告警通知, DDoS高防-域名管理, DDoS高防-实例列表, DDoS高防-概览, DDoS高防-转发规则管理, DDoS高防-防护策略, 解封中心-解封管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddBlackWhiteIpList` | POST | `/v1/{project_id}/aad/external/bwlist` | 高防实例添加黑白名单 |
| `AddPolicyBlackAndWhiteIpList` | POST | `/v1/cnad/policies/{policy_id}/ip-list/add` | 策略添加黑白名单 |
| `AddWafWhiteIpRule` | POST | `/v2/aad/policies/waf/blackwhite-list` | 防护策略web-cc黑白名单-创建黑白名单规则 |
| `AssociateIpToPolicy` | POST | `/v1/cnad/policies/{policy_id}/bind` | 策略绑定防护对象 |
| `AssociateIpToPolicyAndPackage` | POST | `/v3/cnad/policies/{policy_id}/bind` | 策略和实例绑定防护对象 |
| `BatchCreateInstanceIpRule` | POST | `/v1/aad/instances/{instance_id}/{ip}/rules/batch-create` | 批量创建高防实例IP的转发规则 |
| `BatchDeleteInstanceIpRule` | POST | `/v1/aad/instances/{instance_id}/{ip}/rules/batch-delete` | 批量删除高防实例IP的转发规则 |
| `CreateDomain` | POST | `/v1/{project_id}/aad/external/domains` | 创建防护域名 |
| `CreatePolicy` | POST | `/v1/cnad/policies` | 创建策略 |
| `DeleteAlarmConfig` | DELETE | `/v1/cnad/alarm-config` | 删除告警配置 |
| `DeleteBlackWhiteIpList` | DELETE | `/v1/{project_id}/aad/external/bwlist` | 高防实例删除黑白名单 |
| `DeleteDomain` | DELETE | `/v2/aad/domains` | 删除防护域名 |
| `DeletePolicy` | DELETE | `/v1/cnad/policies/{policy_id}` | 删除策略 |
| `DeletePolicyBlackAndWhiteIpList` | POST | `/v1/cnad/policies/{policy_id}/ip-list/delete` | 策略删除黑白名单 |
| `DeleteWafWhiteIpRule` | DELETE | `/v2/aad/policies/waf/blackwhite-list` | 防护策略web-cc黑白名单-删除黑白名单规则 |
| `DisassociateIpFromPolicy` | POST | `/v1/cnad/policies/{policy_id}/unbind` | 策略解绑防护对象 |
| `DisassociateIpFromPolicyAndPackage` | POST | `/v3/cnad/policies/{policy_id}/unbind` | 策略和实例解绑防护对象 |
| `ExecuteUnblockIp` | POST | `/v1/unblockservice/{domain_id}/unblock` | 解封IP |
| `ListBlockIps` | GET | `/v1/unblockservice/{domain_id}/block-list` | 查询租户封堵列表 |
| `ListDDoSAttackEvent` | POST | `/v2/aad/instances/{instance_id}/ddos-info/attack/events` | 查询DDoS攻击事件列表 |
| `ListDDoSBlackHoleEvent` | GET | `/v2/aad/instances/ddos-info/attack/blackhole-event` | 黑洞事件列表 |
| `ListDDoSConnectionNumber` | GET | `/v2/aad/instances/{instance_id}/ddos-info/flow/connection-numbers` | 查询新建连接数和并发连接数 |
| `ListDDoSFlow` | GET | `/v2/aad/instances/{instance_id}/ddos-info/flow` | 查询DDoS攻击防护BPS/PPS流量 |
| `ListDomain` | GET | `/v1/aad/protected-domains` | 查询域名列表 |
| `ListFrequencyControlRule` | GET | `/v2/aad/policies/waf/frequency-control-rule` | 查询频率控制规则列表 |
| `ListGlobalConfig` | GET | `/v2/aad/domains/global-config` | 查询控制台WAF全局配置 |
| `ListInstance` | GET | `/v1/aad/instances` | 查询高防实例列表 |
| `ListInstanceDomains` | GET | `/v2/aad/instances/{instance_id}/domains` | 查询实例关联的域名信息 |
| `ListInstanceId` | GET | `/v1/aad/protected-domains/{domain_id}` | 查询域名关联的实例ID |
| `ListInstanceIpRule` | GET | `/v1/aad/instances/{instance_id}/{ip}/rules` | 查询高防实例IP的转发规则列表 |

... and 41 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
