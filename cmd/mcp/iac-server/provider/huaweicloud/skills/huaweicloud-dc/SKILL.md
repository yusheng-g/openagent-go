---
name: huaweicloud-dc
description: HuaweiCloud DC API guide. 58 APIs covering GEIP操作管理, 专线关联连接, 专线接入点管理, 互联网关, 全域接入网关.
---

# HuaweiCloud DC API Guide

58 APIs. Tags: GEIP操作管理, 专线关联连接, 专线接入点管理, 互联网关, 全域接入网关, 全域接入网关路由表, 标签管理, 物理连接, 虚拟接口, 虚拟接口对等体连通性探测, 虚拟网关, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateResourceTags` | POST | `/v3/{project_id}/{resource_type}/{resource_id}/tags/action` | 批量添加删除资源标签 |
| `BindGlobalEips` | POST | `/v3/{project_id}/dcaas/connect-gateways/{connect_gateway_id}/binding-global-eips` | 绑定GEIP操作 |
| `CreateConnectGateway` | POST | `/v3/{project_id}/dcaas/connect-gateways` | 创建互联网关 |
| `CreateGlobalDcGateway` | POST | `/v3/{project_id}/dcaas/global-dc-gateways` | 创建专线全域接入网关 |
| `CreateHostedDirectConnect` | POST | `/v3/{project_id}/dcaas/hosted-connects` | 创建托管专线连接 |
| `CreatePeerLink` | POST | `/v3/{project_id}/dcaas/global-dc-gateways/{global_dc_gateway_id}/peer-links` | 创建专线关联连接 |
| `CreateResourceTag` | POST | `/v3/{project_id}/{resource_type}/{resource_id}/tags` | 添加资源标签 |
| `CreateVifPeer` | POST | `/v3/{project_id}/dcaas/vif-peers` | 创建虚拟接口对等体 |
| `CreateVifPeerDetection` | POST | `/v3/{project_id}/dcaas/vif-peer-detections` | 创建虚拟接口对等体连通性探测实例 |
| `CreateVirtualGateway` | POST | `/v3/{project_id}/dcaas/virtual-gateways` | 创建虚拟网关 |
| `CreateVirtualInterface` | POST | `/v3/{project_id}/dcaas/virtual-interfaces` | 创建虚拟接口 |
| `DeleteConnectGateway` | DELETE | `/v3/{project_id}/dcaas/connect-gateways/{connect_gateway_id}` | 删除互联网关 |
| `DeleteDirectConnect` | DELETE | `/v3/{project_id}/dcaas/direct-connects/{direct_connect_id}` | 删除物理连接 |
| `DeleteGlobalDcGateway` | DELETE | `/v3/{project_id}/dcaas/global-dc-gateways/{global_dc_gateway_id}` | 删除专线全域接入网关 |
| `DeleteHostedDirectConnect` | DELETE | `/v3/{project_id}/dcaas/hosted-connects/{hosted_connect_id}` | 删除托管专线连接 |
| `DeletePeerLink` | DELETE | `/v3/{project_id}/dcaas/global-dc-gateways/{global_dc_gateway_id}/peer-links/{peer_link_id}` | 删除专线关联连接 |
| `DeleteResourceTag` | DELETE | `/v3/{project_id}/{resource_type}/{resource_id}/tags/{key}` | 删除资源标签 |
| `DeleteVifPeer` | DELETE | `/v3/{project_id}/dcaas/vif-peers/{vif_peer_id}` | 删除虚拟接口对应的对等体 |
| `DeleteVifPeerDetection` | DELETE | `/v3/{project_id}/dcaas/vif-peer-detections/{vif_peer_id}` | 删除虚拟接口对等体连通性探测实例 |
| `DeleteVirtualGateway` | DELETE | `/v3/{project_id}/dcaas/virtual-gateways/{virtual_gateway_id}` | 删除虚拟网关 |
| `DeleteVirtualInterface` | DELETE | `/v3/{project_id}/dcaas/virtual-interfaces/{virtual_interface_id}` | 删除虚拟接口 |
| `ListConnectGateways` | GET | `/v3/{project_id}/dcaas/connect-gateways` | 查询互联网关列表信息 |
| `ListDirectConnectLocations` | GET | `/v3/{project_id}/dcaas/direct-connect-locations` | 查询专线接入点位置列表 |
| `ListDirectConnects` | GET | `/v3/{project_id}/dcaas/direct-connects` | 查询物理连接列表 |
| `ListGdgwRouteTables` | GET | `/v3/{project_id}/dcaas/gdgw/{gdgw_id}/routetables` | 查询全域接入网关路由表 |
| `ListGlobalDcGateways` | GET | `/v3/{project_id}/dcaas/global-dc-gateways` | 查询专线全域接入网关列表 |
| `ListGlobalEips` | GET | `/v3/{project_id}/dcaas/connect-gateways/{connect_gateway_id}/binding-global-eips` | 查询已经绑定的GEIP列表 |
| `ListHostedDirectConnects` | GET | `/v3/{project_id}/dcaas/hosted-connects` | 查询租户的托管专线列表 |
| `ListPeerLinks` | GET | `/v3/{project_id}/dcaas/global-dc-gateways/{global_dc_gateway_id}/peer-links` | 查询专线关联连接列表 |
| `ListProjectTags` | GET | `/v3/{project_id}/{resource_type}/tags` | 查询项目标签 |

... and 28 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
