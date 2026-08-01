---
name: huaweicloud-hcseip
description: HuaweiCloud HCSEIP API guide. 31 APIs covering 浮动IP(社区兼容), 组合接口.
---

# HuaweiCloud HCSEIP API Guide

31 APIs. Tags: 浮动IP(社区兼容), 组合接口

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddBandwith` | POST | `/v2.0/{tenant_id}/bandwidths/{bandwidth_id}/insert` | 共享带宽插入弹性IP |
| `BatchCreateBandwiths` | POST | `/v2.0/{tenant_id}/batch-bandwidths` | 批量创建共享带宽 |
| `CreateBandwiths` | POST | `/v2.0/{tenant_id}/bandwidths` | 创建共享带宽 |
| `CreateEip` | POST | `/v1/{tenant_id}/publicips` | 申请弹性IP |
| `CreateEipEiii` | POST | `/v3/{tenant_id}/eip/publicips` | 申请弹性IP_V3 |
| `CreateFloatingips` | POST | `/v2.0/floatingips` | 创建浮动IP_社区兼容 |
| `CreateFloatingipsEii` | POST | `/v2.1/floatingips` | 创建浮动IP |
| `DeleteBandwith` | DELETE | `/v2.0/{tenant_id}/bandwidths/{bandwidth_id}` | 删除共享带宽 |
| `DeleteEip` | DELETE | `/v1/{tenant_id}/publicips/{publicip_id}` | 删除弹性IP |
| `DeleteFloatingip` | DELETE | `/v2.0/floatingips/{floatingip_id}` | 删除浮动IP_社区兼容 |
| `DeleteFloatingipsEiiById` | DELETE | `/v2.1/floatingips/{floatingip_id}` | 删除浮动IP |
| `ListBandwidthInternal` | GET | `/v2.0/{tenant_id}/bandwidths` | 查询带宽列表_v2 |
| `ListBandwiths` | GET | `/v1/{tenant_id}/bandwidths` | 查询带宽列表_v1 |
| `ListEip` | GET | `/v3/{tenant_id}/eip/publicips` | 查询弹性IP列表_V3 |
| `ListFloatingip` | GET | `/v2.0/floatingips` | 查询浮动IP列表_社区兼容 |
| `ListPublicIp` | GET | `/v1/{tenant_id}/publicips` | 查询弹性IP列表 |
| `RemoveBandwith` | POST | `/v2.0/{tenant_id}/bandwidths/{bandwidth_id}/remove` | 共享带宽移除弹性IP |
| `ShowBandwidthDetailInternal` | GET | `/v2.0/{tenant_id}/bandwidths/{bandwidth_id}` | 查询带宽详情_v2 |
| `ShowBandwith` | GET | `/v1/{tenant_id}/bandwidths/{bandwidth_id}` | 查询带宽详情_v1 |
| `ShowEipByTypes` | GET | `/v1/{tenant_id}/publicip_types/{vpc_id}` | 查询弹性IP类型分组列表 |
| `ShowEipExt` | GET | `/v1/{tenant_id}/publicips/ext/{vpc_id}` | 查询VPC能使用的弹性IP |
| `ShowEipListEiii` | GET | `/v1/{tenant_id}/publicip_types` | 查询弹性IP类型列表 |
| `ShowFloatingipDetails` | GET | `/v2.0/floatingips/{floatingip_id}` | 查询浮动IP详情_社区兼容 |
| `ShowFloatingipEiiById` | GET | `/v2.1/floatingips/{floatingip_id}` | 查询浮动IP |
| `ShowFloatingipsEii` | GET | `/v2/{tenant_id}/floatingips` | 查询浮动IP列表V2 |
| `ShowFloatingipsListEii` | GET | `/v2.1/floatingips` | 查询浮动IP列表V2.1 |
| `ShowPublicIp` | GET | `/v1/{tenant_id}/publicips/{publicip_id}` | 查询弹性IP |
| `UpdateBandwidthInternal` | PUT | `/v2.0/{tenant_id}/bandwidths/{bandwidth_id}` | 更新带宽_v2 |
| `UpdateEip` | PUT | `/v1/{tenant_id}/publicips/{publicip_id}` | 更新弹性IP |
| `UpdateFloatingip` | PUT | `/v2.0/floatingips/{floatingip_id}` | 更新浮动IP_社区兼容 |

... and 1 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
