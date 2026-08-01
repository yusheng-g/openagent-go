---
name: huaweicloud-eip
description: HuaweiCloud EIP API guide. 68 APIs covering GEIP与实例的绑定关系, OpenStack - 浮动IP, 公共池, 共享带宽类型, 带宽.
---

# HuaweiCloud EIP API Guide

68 APIs. Tags: GEIP与实例的绑定关系, OpenStack - 浮动IP, 公共池, 共享带宽类型, 带宽, 带宽加油包, 带宽规则, 弹性公网IP, 弹性公网IP标签管理, 弹性公网IP辅助接口, 批量操作弹性公网IP, 查询Job状态, 虚拟igw, 配额

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddPublicipsIntoSharedBandwidth` | POST | `/v2.0/{project_id}/bandwidths/{bandwidth_id}/insert` | 共享带宽插入弹性公网IP |
| `AssociatePublicips` | POST | `/v3/{project_id}/eip/publicips/{publicip_id}/associate-instance` | 绑定弹性公网IP |
| `AttachBatchPublicIp` | POST | `/v3/{project_id}/eip/publicips/attach-share-bandwidth` | 共享带宽批量加入弹性公网IP |
| `AttachShareBandwidth` | POST | `/v3/{project_id}/eip/publicips/{publicip_id}/attach-share-bandwidth` | 共享带宽加入弹性公网IP |
| `BatchCreatePublicips` | POST | `/v2/{project_id}/batchpublicips` | 批量创建弹性公网IP |
| `BatchCreatePublicipTags` | POST | `/v2.0/{project_id}/publicips/{publicip_id}/tags/action` | 批量创建弹性公网IP资源标签 |
| `BatchCreateSharedBandwidths` | POST | `/v2.0/{project_id}/batch-bandwidths` | 批量创建共享带宽 |
| `BatchDeletePublicIp` | DELETE | `/v2/{project_id}/batchpublicips` | 批量删除弹性公网IP |
| `BatchDeletePublicipTags` | POST | `/v2.0/{project_id}/publicips/{publicip_id}/tags/action` | 批量删除弹性公网IP资源标签 |
| `BatchDisassociatePublicips` | PATCH | `/v2/{project_id}/batchpublicips` | 批量解绑弹性公网IP |
| `BatchModifyBandwidth` | PUT | `/v2/{project_id}/batch-bandwidths/modify` | 批量更新带宽 |
| `ChangeBandwidthToPeriod` | POST | `/v2.0/{project_id}/bandwidths/change-to-period` | 按需转包API |
| `ChangePublicipToPeriod` | POST | `/v2.0/{project_id}/publicips/change-to-period` | 按需转包接口 |
| `CountEipAvailableResources` | POST | `/v3/{project_id}/eip/resources/available` | 查询弹性公网IP可用数 |
| `CountPublicIp` | GET | `/v2/{project_id}/elasticips` | 查询PublicIp数量 |
| `CountPublicIpInstance` | GET | `/v2/{project_id}/publicip/instances` | 查询PublicIp实例数 |
| `CreateBandwidthRuleV3` | POST | `/v3/{project_id}/eip/bandwidths/{bandwidth_id}/bandwidth-rules` | 创建带宽分组规则 |
| `CreatePrePaidPublicip` | POST | `/v2.0/{project_id}/publicips` | 申请包周期弹性公网IP |
| `CreatePublicip` | POST | `/v1/{project_id}/publicips` | 申请弹性公网IP |
| `CreatePublicipTag` | POST | `/v2.0/{project_id}/publicips/{publicip_id}/tags` | 创建弹性公网IP资源标签 |
| `CreateSharedBandwidth` | POST | `/v2.0/{project_id}/bandwidths` | 创建共享带宽 |
| `CreateTenantVpcIgw` | POST | `/v3/{project_id}/geip/vpc-igws` | 创建虚拟igw |
| `DeleteBandwidthRuleV3` | DELETE | `/v3/{project_id}/eip/bandwidths/{bandwidth_id}/bandwidth-rule/{bandwidth_rules_id}` | 删除带宽分组规则 |
| `DeletePublicip` | DELETE | `/v1/{project_id}/publicips/{publicip_id}` | 删除弹性公网IP |
| `DeletePublicipTag` | DELETE | `/v2.0/{project_id}/publicips/{publicip_id}/tags/{key}` | 删除弹性公网IP的标签 |
| `DeleteSharedBandwidth` | DELETE | `/v2.0/{project_id}/bandwidths/{bandwidth_id}` | 删除共享带宽 |
| `DeleteTenantVpcIgw` | DELETE | `/v3/{project_id}/geip/vpc-igws/{vpc_igw_id}` | 删除虚拟igw |
| `DetachBatchPublicIp` | POST | `/v3/{project_id}/eip/publicips/detach-share-bandwidth` | 共享带宽批量移出弹性公网IP |
| `DetachShareBandwidth` | POST | `/v3/{project_id}/eip/publicips/{publicip_id}/detach-share-bandwidth` | 共享带宽移出弹性公网IP |
| `DisableNat64` | POST | `/v3/{project_id}/eip/publicips/{publicip_id}/disable-nat64` | 弹性公网IP关闭NAT64 |

... and 38 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
