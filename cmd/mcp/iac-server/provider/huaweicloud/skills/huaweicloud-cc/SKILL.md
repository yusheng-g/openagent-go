---
name: huaweicloud-cc
description: HuaweiCloud CC API guide. 111 APIs covering 中心网络的附件管理, 中心网络管理, 中心网络能力, 中心网络连接, 中心网络配额.
---

# HuaweiCloud CC API Guide

111 APIs. Tags: 中心网络的附件管理, 中心网络管理, 中心网络能力, 中心网络连接, 中心网络配额, 云连接实例, 云连接能力, 云连接路由, 云连接配额, 全域互联带宽, 全域互联带宽标签管理, 分支网络管理, 分支网络能力, 分支网络配额, 分支连接管理, 域间带宽, 带宽包, 授权管理, 标签管理, 网络实例, 规格管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ApplyCentralNetworkPolicy` | POST | `/v3/{domain_id}/gcn/central-network/{central_network_id}/policies/{policy_id}/apply` | 应用中心网络策略 |
| `AssociateBandwidthPackage` | POST | `/v3/{domain_id}/ccaas/bandwidth-packages/{id}/associate` | 将带宽包实例绑定到云连接实例 |
| `AssociateGlobalConnectionBandwidthInstance` | POST | `/v3/{domain_id}/gcb/gcbandwidths/{id}/associate-instance` | 全域互联带宽绑定实例 |
| `AssociateSiteNetworkBandwidth` | POST | `/v3/{domain_id}/dcaas/site-network/{site_network_id}/connections/{site_connection_id}/associate` | 关联分支连接带宽 |
| `BatchCreateDeleteTags` | POST | `/v3/{domain_id}/ccaas/{resource_type}/{resource_id}/tags/action` | 批量创建和删除资源标签 |
| `BatchCreateGcbResourceTags` | POST | `/v3/gcb/{resource_id}/tags/create` | 批量添加账户全域互联带宽资源标签 |
| `BatchDeleteGcbResourceTags` | POST | `/v3/gcb/{resource_id}/tags/delete` | 批量删除账户全域互联带宽资源标签 |
| `CountGcbResourceByTag` | POST | `/v3/gcb/resource-instances/count` | 查询账户全域互联带宽资源标签数量 |
| `CreateAuthorisation` | POST | `/v3/{domain_id}/ccaas/authorisations` | 创建授权 |
| `CreateBandwidthPackage` | POST | `/v3/{domain_id}/ccaas/bandwidth-packages` | 创建带宽包实例 |
| `CreateCentralNetwork` | POST | `/v3/{domain_id}/gcn/central-networks` | 创建中心网络 |
| `CreateCentralNetworkErRouteTableAttachment` | POST | `/v3/{domain_id}/gcn/central-network/{central_network_id}/er-route-table-attachments` | 创建中心网络ER路由表附件 |
| `CreateCentralNetworkGdgwAttachment` | POST | `/v3/{domain_id}/gcn/central-network/{central_network_id}/gdgw-attachments` | 创建中心网络GDGW附件 |
| `CreateCentralNetworkPolicy` | POST | `/v3/{domain_id}/gcn/central-network/{central_network_id}/policies` | 创建一个新版本的中心网络策略 |
| `CreateCloudConnection` | POST | `/v3/{domain_id}/ccaas/cloud-connections` | 创建云连接实例 |
| `CreateGcbResourceTag` | POST | `/v3/gcb/{resource_id}/tags` | 添加账户全域互联带宽资源标签 |
| `CreateGlobalConnectionBandwidth` | POST | `/v3/{domain_id}/gcb/gcbandwidths` | 创建全域互联带宽 |
| `CreateInterRegionBandwidth` | POST | `/v3/{domain_id}/ccaas/inter-region-bandwidths` | 创建域间带宽实例 |
| `CreateNetworkInstance` | POST | `/v3/{domain_id}/ccaas/network-instances` | 创建网络实例 |
| `CreateP2PSiteNetwork` | POST | `/v3/{domain_id}/dcaas/p2p-site-networks` | 创建P2P类型的分支网络 |
| `CreateTag` | POST | `/v3/{domain_id}/ccaas/{resource_type}/{resource_id}/tags` | 添加资源标签 |
| `DeleteAuthorisation` | DELETE | `/v3/{domain_id}/ccaas/authorisations/{id}` | 删除授权 |
| `DeleteBandwidthPackage` | DELETE | `/v3/{domain_id}/ccaas/bandwidth-packages/{id}` | 删除带宽包实例 |
| `DeleteCentralNetwork` | DELETE | `/v3/{domain_id}/gcn/central-networks/{central_network_id}` | 删除中心网络 |
| `DeleteCentralNetworkAttachment` | DELETE | `/v3/{domain_id}/gcn/central-network/{central_network_id}/attachments/{attachment_id}` | 删除中心网络附件 |
| `DeleteCentralNetworkPolicy` | DELETE | `/v3/{domain_id}/gcn/central-network/{central_network_id}/policies/{policy_id}` | 删除中心网络策略版本 |
| `DeleteCloudConnection` | DELETE | `/v3/{domain_id}/ccaas/cloud-connections/{id}` | 删除云连接实例 |
| `DeleteGcbResourceTag` | DELETE | `/v3/gcb/{resource_id}/tags/{tag_key}` | 删除账户全域互联带宽资源标签 |
| `DeleteGlobalConnectionBandwidth` | DELETE | `/v3/{domain_id}/gcb/gcbandwidths/{id}` | 删除全域互联带宽 |
| `DeleteInterRegionBandwidth` | DELETE | `/v3/{domain_id}/ccaas/inter-region-bandwidths/{id}` | 删除域间带宽实例 |

... and 81 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
