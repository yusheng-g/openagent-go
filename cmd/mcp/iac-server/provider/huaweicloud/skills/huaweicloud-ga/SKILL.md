---
name: huaweicloud-ga
description: HuaweiCloud GA API guide. 49 APIs covering IP地址组, 云日志, 健康检查, 全球加速实例, 区域.
---

# HuaweiCloud GA API Guide

49 APIs. Tags: IP地址组, 云日志, 健康检查, 全球加速实例, 区域, 接入点, 标签, 监听器, 终端节点, 终端节点组, 自带IP地址池, 配额

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddIpGroupIp` | POST | `/v1/ip-groups/{ip_group_id}/add-ips` | 添加IP地址组中的IP网段 |
| `AssociateListener` | POST | `/v1/ip-groups/{ip_group_id}/associate-listener` | 绑定IP地址组与监听器 |
| `CountResourcesByTag` | POST | `/v1/{resource_type}/resource-instances/count` | 通过标签查询资源实例数量 |
| `CreateAccelerator` | POST | `/v1/accelerators` | 创建全球加速器 |
| `CreateEndpoint` | POST | `/v1/endpoint-groups/{endpoint_group_id}/endpoints` | 创建终端节点 |
| `CreateEndpointGroup` | POST | `/v1/endpoint-groups` | 创建终端节点组 |
| `CreateHealthCheck` | POST | `/v1/health-checks` | 创建健康检查 |
| `CreateIpGroup` | POST | `/v1/ip-groups` | 创建IP地址组 |
| `CreateListener` | POST | `/v1/listeners` | 创建监听器 |
| `CreateLogtank` | POST | `/v1/logtanks` | 创建云日志 |
| `CreateTags` | POST | `/v1/{resource_type}/{resource_id}/tags/create` | 创建资源标签 |
| `DeleteAccelerator` | DELETE | `/v1/accelerators/{accelerator_id}` | 删除全球加速器 |
| `DeleteEndpoint` | DELETE | `/v1/endpoint-groups/{endpoint_group_id}/endpoints/{endpoint_id}` | 删除终端节点 |
| `DeleteEndpointGroup` | DELETE | `/v1/endpoint-groups/{endpoint_group_id}` | 删除终端节点组 |
| `DeleteHealthCheck` | DELETE | `/v1/health-checks/{health_check_id}` | 删除健康检查 |
| `DeleteIpGroup` | DELETE | `/v1/ip-groups/{ip_group_id}` | 删除IP地址组 |
| `DeleteListener` | DELETE | `/v1/listeners/{listener_id}` | 删除监听器 |
| `DeleteLogtank` | DELETE | `/v1/logtanks/{logtank_id}` | 删除云日志 |
| `DeleteTags` | DELETE | `/v1/{resource_type}/{resource_id}/tags/delete` | 删除资源标签 |
| `DisassociateListener` | POST | `/v1/ip-groups/{ip_group_id}/disassociate-listener` | 解绑IP地址组与监听器 |
| `ListAccelerators` | GET | `/v1/accelerators` | 查询全球加速器列表 |
| `ListAllPops` | GET | `/v1/pops` | 查询pop列表 |
| `ListByoipPools` | GET | `/v1/byoip-pools` | 查询自带IP地址池列表 |
| `ListEndpointGroups` | GET | `/v1/endpoint-groups` | 查询终端节点组列表 |
| `ListEndpoints` | GET | `/v1/endpoint-groups/{endpoint_group_id}/endpoints` | 查询终端节点组下终端节点列表 |
| `ListHealthChecks` | GET | `/v1/health-checks` | 查询健康检查列表 |
| `ListIpGroups` | GET | `/v1/ip-groups` | 查询IP地址组列表 |
| `ListListeners` | GET | `/v1/listeners` | 查询监听器列表 |
| `ListLogtanks` | GET | `/v1/logtanks` | 查询云日志列表 |
| `ListRegions` | GET | `/v1/regions` | 查询区域列表 |

... and 19 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
