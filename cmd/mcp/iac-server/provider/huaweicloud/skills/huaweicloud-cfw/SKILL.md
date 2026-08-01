---
name: huaweicloud-cfw
description: HuaweiCloud CFW API guide. 168 APIs covering EIP管理, IPS白名单管理, IPS管理, VPC间防火墙管理, 反病毒管理.
---

# HuaweiCloud CFW API Guide

168 APIs. Tags: EIP管理, IPS白名单管理, IPS管理, VPC间防火墙管理, 反病毒管理, 告警配置管理, 地址组管理, 域名解析及域名组管理, 多账号管理, 安全报告, 抓包管理, 日志分析, 日志管理, 时间表管理, 服务组管理, 标签管理, 流量过滤, 访问控制规则管理, 防火墙管理, 黑白名单管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddAclRule` | POST | `/v1/{project_id}/acl-rule` | 创建ACL规则 |
| `AddAddressItem` | POST | `/v1/{project_id}/address-items` | 添加地址组成员 |
| `AddAddressSet` | POST | `/v1/{project_id}/address-set` | 添加地址组 |
| `AddBlackWhiteList` | POST | `/v1/{project_id}/black-white-list` | 创建黑白名单规则 |
| `AddCustomDnsServer` | POST | `/v1/{project_id}/dns/server/{server_ip}/status` | 添加指定DNS服务器 |
| `AddDomains` | POST | `/v1/{project_id}/domain-set/domains/{set_id}` | 添加域名列表 |
| `AddDomainSet` | POST | `/v1/{project_id}/domain-set` | 添加域名组 |
| `AddEipAlarmWhitelist` | POST | `/v1/{project_id}/eip/alarm-whitelist` | 添加EIP告警白名单 |
| `AddLogConfig` | POST | `/v1/{project_id}/cfw/logs/configuration` | 创建日志配置 |
| `AddServiceItems` | POST | `/v1/{project_id}/service-items` | 新建服务成员 |
| `AddServiceSet` | POST | `/v1/{project_id}/service-set` | 新建服务组 |
| `BatchAddAccounts` | POST | `/v1/{project_id}/system/multi-account/accounts` | 批量添加账号 |
| `BatchCreateBlackWhiteList` | POST | `/v1/{project_id}/black-white-lists` | 批量添加黑白名单列表 |
| `BatchCreateIpsWhitelist` | POST | `/v1/{project_id}/cfw/{fw_instance_id}/ips/whitelist/batch-create` | 批量添加IPS白名单 |
| `BatchCreatePrivateNetworkSegments` | POST | `/v2/{project_id}/firewall/{fw_instance_id}/east-west/private-network-segments/batch-create` | 创建私网网段 |
| `BatchDeleteAclRules` | DELETE | `/v1/{project_id}/acl-rule` | 批量删除Acl规则 |
| `BatchDeleteAddressItems` | DELETE | `/v1/{project_id}/address-items` | 批量删除地址组成员 |
| `BatchDeleteAddressSets` | POST | `/v1/{project_id}/address-sets/batch-delete` | 批量删除地址组 |
| `BatchDeleteBlackWhiteLists` | DELETE | `/v1/{project_id}/black-white-list` | 批量删除黑白名单列表 |
| `BatchDeleteCustomerIps` | POST | `/v1/{project_id}/ips/custom-rule/batch-delete` | 批量删除自定义IPS规则 |
| `BatchDeleteDomainSet` | POST | `/v1/{project_id}/domain-sets/batch-delete` | 批量删除域名组 |
| `BatchDeleteIpsWhitelist` | POST | `/v1/{project_id}/cfw/{fw_instance_id}/ips/whitelist/batch-delete` | 批量删除IPS白名单 |
| `BatchDeletePrivateNetworkSegments` | POST | `/v2/{project_id}/firewall/{fw_instance_id}/east-west/private-network-segments/batch-delete` | 删除私网网段信息 |
| `BatchDeleteSchedules` | POST | `/v1/{project_id}/schedules/batch-delete` | 批量删除时间表 |
| `BatchDeleteServiceItems` | DELETE | `/v1/{project_id}/service-items` | 批量删除服务组成员信息 |
| `BatchDeleteServiceSets` | POST | `/v1/{project_id}/service-sets/batch-delete` | 批量删除服务组 |
| `BatchRemoveAccounts` | POST | `/v1/{project_id}/system/multi-account/batch-delete` | 批量移除账号 |
| `BatchUpdateAclRuleActions` | PUT | `/v1/{project_id}/acl-rule/action` | 批量更新规则动作 |
| `BatchUpdateCustomerIpsAction` | POST | `/v1/{project_id}/ips/custom-rule/action` | 批量更新自定义IPS规则的动作 |
| `CancelCaptureTask` | POST | `/v1/{project_id}/capture-task/stop` | 取消抓包任务 |

... and 138 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
