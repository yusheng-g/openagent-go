---
name: huaweicloud-elb
description: HuaweiCloud ELB API guide. 137 APIs covering API版本信息, IP地址组, SSL证书管理, 主备后端服务器组, 云日志. Detailed swagger definitions in references/<APIName>.json.
---

# HuaweiCloud ELB API Guide

137 APIs. Tags: API版本信息, IP地址组, SSL证书管理, 主备后端服务器组, 云日志, 健康检查, 可用区, 后端云服务器, 后端云服务器组, 后端服务器, 后端服务器组, 回收站, 安全策略, 异步任务, 标签管理, 特性配置, 白名单, 监听器, 规格, 证书, 负载均衡器, 转发策略, 转发规则, 配额, 预占IP

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchAddAvailableZones` | POST | `/v3/{project_id}/elb/loadbalancers/{loadbalancer_id}/availability-zone/batch-add` | 新增负载均衡器可用区 |
| `BatchCreateListenerTags` | POST | `/v2.0/{project_id}/listeners/{listener_id}/tags/action` | 批量添加监听器标签 |
| `BatchCreateLoadBalancers` | POST | `/v3/{project_id}/elb/loadbalancers/batch-create` | 批量创建负载均衡器 |
| `BatchCreateLoadbalancerTags` | POST | `/v2.0/{project_id}/loadbalancers/{loadbalancer_id}/tags/action` | 批量添加负载均衡器标签 |
| `BatchCreateMembers` | POST | `/v3/{project_id}/elb/pools/{pool_id}/members/batch-add` | 批量创建后端服务器 |
| `BatchDeleteCertificates` | POST | `/v3/{project_id}/elb/certificates/batch-delete` | 批量删除证书 |
| `BatchDeleteIpList` | POST | `/v3/{project_id}/elb/ipgroups/{ipgroup_id}/iplist/batch-delete` | 删除IP地址组的IP列表项 |
| `BatchDeleteListeners` | POST | `/v3/{project_id}/elb/listeners/batch-delete` | 批量删监听器 |
| `BatchDeleteListenerTags` | POST | `/v2.0/{project_id}/listeners/{listener_id}/tags/action` | 批量删除监听器标签 |
| `BatchDeleteLoadbalancers` | POST | `/v3/{project_id}/elb/loadbalancers/batch-delete` | 批量删除负载均衡器 |
| `BatchDeleteLoadbalancerTags` | POST | `/v2.0/{project_id}/loadbalancers/{loadbalancer_id}/tags/action` | 批量删除负载均衡器标签 |
| `BatchDeleteMembers` | POST | `/v3/{project_id}/elb/pools/{pool_id}/members/batch-delete` | 批量删除后端服务器 |
| `BatchDeletePools` | POST | `/v3/{project_id}/elb/pools/batch-delete` | 批量删除后端服务器组 |
| `BatchDisableDomainIPs` | POST | `/v3/{project_id}/elb/loadbalancers/{loadbalancer_id}/dns/ips/batch-disable` | 批量将IP地址从ELB实例域名解析中移除 |
| `BatchEnableDomainIPs` | POST | `/v3/{project_id}/elb/loadbalancers/{loadbalancer_id}/dns/ips/batch-enable` | 批量将IP地址加入ELB实例域名解析中 |
| `BatchRemoveAvailableZones` | POST | `/v3/{project_id}/elb/loadbalancers/{loadbalancer_id}/availability-zone/batch-remove` | 移除负载均衡器可用区 |
| `BatchUpdateMembers` | POST | `/v3/{project_id}/elb/pools/{pool_id}/members/batch-update` | 批量更新后端服务器 |
| `BatchUpdatePoliciesPriority` | POST | `/v3/{project_id}/elb/l7policies/batch-update-priority` | 批量更新转发策略优先级 |
| `ChangeListenerTags` | POST | `/v3/{project_id}/listeners/{listener_id}/tags/action` | 变更监听器标签列表 |
| `ChangeLoadbalancerChargeMode` | POST | `/v3/{project_id}/elb/loadbalancers/change-charge-mode` | 变更负载均衡器计费模式 |
| `ChangeLoadbalancerTags` | POST | `/v3/{project_id}/loadbalancers/{loadbalancer_id}/tags/action` | 变更负载均衡器标签列表 |
| `CloneListener` | POST | `/v3/{project_id}/elb/listeners/{listener_id}/clone` | 复制已有监听器 |
| `CloneLoadbalancer` | POST | `/v3/{project_id}/elb/loadbalancers/{loadbalancer_id}/clone` | 复制已有负载均衡器 |
| `CountPreoccupyIpNum` | GET | `/v3/{project_id}/elb/preoccupy-ip-num` | 计算预占IP数 |
| `CreateCertificate` | POST | `/v2/{project_id}/elb/certificates` | 创建SSL证书 |
| `CreateCertificatePrivateKeyEcho` | POST | `/v3/{project_id}/elb/certificates/settings/private-key-echo` | 修改证书私钥字段回显开关 |
| `CreateHealthmonitor` | POST | `/v2/{project_id}/elb/healthmonitors` | 创建健康检查 |
| `CreateIpGroup` | POST | `/v3/{project_id}/elb/ipgroups` | 创建IP地址组 |
| `CreateL7Policy` | POST | `/v3/{project_id}/elb/l7policies` | 创建转发策略 |
| `CreateL7rule` | POST | `/v2/{project_id}/elb/l7policies/{l7policy_id}/rules` | 创建转发规则 |

... and 107 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
