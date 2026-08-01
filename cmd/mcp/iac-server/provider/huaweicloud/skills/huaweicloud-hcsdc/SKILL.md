---
name: huaweicloud-hcsdc
description: HuaweiCloud HCSDC API guide. 32 APIs covering 专线端口组, 物理专线, 虚拟接口, 虚拟网关, 逃生通道(只涉及增强型云专线(L3GW)).
---

# HuaweiCloud HCSDC API Guide

32 APIs. Tags: 专线端口组, 物理专线, 虚拟接口, 虚拟网关, 逃生通道(只涉及增强型云专线(L3GW)), 预配置, 高可用组

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateDcEndpointGroup` | POST | `/v2.0/dcaas/dc-endpoint-groups` | 创建专线端口组 |
| `CreateDirectConnect` | POST | `/v2.0/dcaas/direct-connects` | 创建物理专线 |
| `CreateEmergencyChannels` | POST | `/v2.0/dcaas/dc-emergency-channels` | 创建专线逃生通道 |
| `CreateHighAvailableGroup` | POST | `/v2.0/dcaas/high-available-groups` | 创建高可用组 |
| `CreateVirtualGateway` | POST | `/v2.0/dcaas/virtual-gateways` | 创建虚拟网关 |
| `CreateVirtualInterface` | POST | `/v2.0/dcaas/virtual-interfaces` | 创建虚拟接口 |
| `ListDcEndpointGroup` | GET | `/v2.0/dcaas/dc-endpoint-groups` | 查询专线端口组列表 |
| `ListDirectConnects` | GET | `/v2.0/dcaas/direct-connects` | 查询物理专线列表 |
| `ListEmergencyChannels` | GET | `/v2.0/dcaas/dc-emergency-channels` | 查询专线逃生通道列表 |
| `ListHighAvailableGroups` | GET | `/v2.0/dcaas/high-available-groups` | 查询高可用组列表 |
| `ListHostingDcPreConfigs` | GET | `/v2.0/dcaas/hosting-dc-pre-configs` | 查询预配置列表 |
| `ListVirtualGateways` | GET | `/v2.0/dcaas/virtual-gateways` | 查询虚拟网关列表 |
| `ListVirtualInterfaces` | GET | `/v2.0/dcaas/virtual-interfaces` | 查询虚拟接口列表 |
| `RemoveDcEndpointGroup` | DELETE | `/v2.0/dcaas/dc-endpoint-groups/{endpoint_group_id}` | 删除专线端口组 |
| `RemoveDirectConnect` | DELETE | `/v2.0/dcaas/direct-connects/{direct_connect_id}` | 删除物理专线 |
| `RemoveEmergencyChannel` | DELETE | `/v2.0/dcaas/dc-emergency-channels/{emergency_channel_id}` | 删除专线逃生通道 |
| `RemoveHighAvailableGroup` | DELETE | `/v2.0/dcaas/high-available-groups/{high_available_group_id}` | 删除高可用组 |
| `RemoveVirtualGateway` | DELETE | `/v2.0/dcaas/virtual-gateways/{virtual_gateway_id}` | 删除虚拟网关 |
| `RemoveVirtualInterface` | DELETE | `/v2.0/dcaas/virtual-interfaces/{virtual_interface_id}` | 删除虚拟接口 |
| `ShowDcEndpointGroup` | GET | `/v2.0/dcaas/dc-endpoint-groups/{endpoint_group_id}` | 查询专线端口组 |
| `ShowDirectConnect` | GET | `/v2.0/dcaas/direct-connects/{direct_connect_id}` | 查询物理专线 |
| `ShowEmergencyChannel` | GET | `/v2.0/dcaas/dc-emergency-channels/{emergency_channel_id}` | 查询专线逃生通道 |
| `ShowHighAvailableGroup` | GET | `/v2.0/dcaas/high-available-groups/{high_available_group_id}` | 查询高可用组 |
| `ShowHostingDcPreconfig` | GET | `/v2.0/dcaas/hosting-dc-pre-configs/{hosting_dc_pre_config_id}` | 查询预配置 |
| `ShowVirtualGateway` | GET | `/v2.0/dcaas/virtual-gateways/{virtual_gateway_id}` | 查询虚拟网关 |
| `ShowVirtualInterface` | GET | `/v2.0/dcaas/virtual-interfaces/{virtual_interface_id}` | 查询虚拟接口 |
| `UpdateDcEndpointGroup` | PUT | `/v2.0/dcaas/dc-endpoint-groups/{endpoint_group_id}` | 更新专线端口组 |
| `UpdateDirectConnect` | PUT | `/v2.0/dcaas/direct-connects/{direct_connect_id}` | 更新物理专线 |
| `UpdateEmergencyChannel` | PUT | `/v2.0/dcaas/dc-emergency-channels/{emergency_channel_id}` | 更新专线逃生通道 |
| `UpdateHighAvailableGroup` | PUT | `/v2.0/dcaas/high-available-groups/{high_available_group_id}` | 更新高可用组 |

... and 2 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
