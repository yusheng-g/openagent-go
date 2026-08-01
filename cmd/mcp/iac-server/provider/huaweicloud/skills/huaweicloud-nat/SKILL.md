---
name: huaweicloud-nat
description: HuaweiCloud NAT API guide. 67 APIs covering 中转IP标签管理, 中转子网, 中转子网标签管理, 公网DNAT规则, 公网NAT网关. Detailed swagger definitions in references/<APIName>.json.
---

# HuaweiCloud NAT API Guide

67 APIs. Tags: 中转IP标签管理, 中转子网, 中转子网标签管理, 公网DNAT规则, 公网NAT网关, 公网NAT网关标签, 公网SNAT规则, 私网DNAT规则, 私网NAT中转IP, 私网NAT网关, 私网NAT网关标签管理, 私网SNAT规则

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateDeleteNatGatewayTag` | POST | `/v3/{project_id}/nat_gateways/{nat_gateway_id}/tags/action` | 批量添加/删除公网NAT网关资源标签 |
| `BatchCreateDeletePrivateNatTags` | POST | `/v3/{project_id}/private-nat-gateways/{resource_id}/tags/action` | 批量添加删除私网NAT网关标签 |
| `BatchCreateDeleteTransitIpTags` | POST | `/v3/{project_id}/transit-ips/{resource_id}/tags/action` | 批量添加删除中转IP标签 |
| `BatchCreateDeleteTransitSubnetTags` | POST | `/v3/{project_id}/transit-subnets/{resource_id}/tags/action` | 批量添加删除中转子网标签 |
| `BatchCreateNatGatewayDnatRules` | POST | `/v2/{project_id}/dnat_rules/batch` | 批量创建DNAT规则 |
| `CreateNatGateway` | POST | `/v2/{project_id}/nat_gateways` | 创建公网NAT网关 |
| `CreateNatGatewayDnatRule` | POST | `/v2/{project_id}/dnat_rules` | 创建DNAT规则 |
| `CreateNatGatewaySnatRule` | POST | `/v2/{project_id}/snat_rules` | 创建SNAT规则 |
| `CreateNatGatewayTag` | POST | `/v3/{project_id}/nat_gateways/{nat_gateway_id}/tags` | 添加公网NAT网关资源标签 |
| `CreatePrivateDnat` | POST | `/v3/{project_id}/private-nat/dnat-rules` | 创建DNAT规则 |
| `CreatePrivateNat` | POST | `/v3/{project_id}/private-nat/gateways` | 创建私网NAT网关 |
| `CreatePrivateNatTag` | POST | `/v3/{project_id}/private-nat-gateways/{resource_id}/tags` | 添加私网NAT网关标签 |
| `CreatePrivateSnat` | POST | `/v3/{project_id}/private-nat/snat-rules` | 创建SNAT规则 |
| `CreateTransitIp` | POST | `/v3/{project_id}/private-nat/transit-ips` | 创建中转IP |
| `CreateTransitIpTag` | POST | `/v3/{project_id}/transit-ips/{resource_id}/tags` | 添加中转IP标签 |
| `CreateTransitSubnet` | POST | `/v3/{project_id}/private-nat/transit-subnets` | 创建中转子网 |
| `CreateTransitSubnetTag` | POST | `/v3/{project_id}/transit-subnets/{resource_id}/tags` | 添加中转子网标签 |
| `DeleteNatGateway` | DELETE | `/v2/{project_id}/nat_gateways/{nat_gateway_id}` | 删除公网NAT网关 |
| `DeleteNatGatewayDnatRule` | DELETE | `/v2/{project_id}/nat_gateways/{nat_gateway_id}/dnat_rules/{dnat_rule_id}` | 删除DNAT规则 |
| `DeleteNatGatewaySnatRule` | DELETE | `/v2/{project_id}/nat_gateways/{nat_gateway_id}/snat_rules/{snat_rule_id}` | 删除SNAT规则 |
| `DeleteNatGatewayTag` | DELETE | `/v3/{project_id}/nat_gateways/{nat_gateway_id}/tags/{key}` | 删除公网NAT网关资源标签 |
| `DeletePrivateDnat` | DELETE | `/v3/{project_id}/private-nat/dnat-rules/{dnat_rule_id}` | 删除DNAT规则 |
| `DeletePrivateNat` | DELETE | `/v3/{project_id}/private-nat/gateways/{gateway_id}` | 删除私网NAT网关 |
| `DeletePrivateNatTag` | DELETE | `/v3/{project_id}/private-nat-gateways/{resource_id}/tags/{key}` | 删除私网NAT网关标签 |
| `DeletePrivateSnat` | DELETE | `/v3/{project_id}/private-nat/snat-rules/{snat_rule_id}` | 删除SNAT规则 |
| `DeleteTransitIp` | DELETE | `/v3/{project_id}/private-nat/transit-ips/{transit_ip_id}` | 删除中转IP |
| `DeleteTransitIpTag` | DELETE | `/v3/{project_id}/transit-ips/{resource_id}/tags/{key}` | 删除中转IP标签 |
| `DeleteTransitSubnet` | DELETE | `/v3/{project_id}/private-nat/transit-subnets/{transit_subnet_id}` | 删除中转子网 |
| `DeleteTransitSubnetTag` | DELETE | `/v3/{project_id}/transit-subnets/{resource_id}/tags/{key}` | 删除中转子网标签 |
| `ListNatGatewayByTag` | POST | `/v3/{project_id}/nat_gateways/resource_instances/action` | 查询公网NAT网关资源实例 |

... and 37 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
