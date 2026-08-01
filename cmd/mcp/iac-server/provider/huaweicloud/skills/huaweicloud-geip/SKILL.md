---
name: huaweicloud-geip
description: HuaweiCloud GEIP API guide. 66 APIs covering Job相关接口, Region限制, 免责条款签署, 全域公网带宽, 全域公网带宽标签.
---

# HuaweiCloud GEIP API Guide

66 APIs. Tags: Job相关接口, Region限制, 免责条款签署, 全域公网带宽, 全域公网带宽标签, 全域公网带宽限制, 全域弹性公网IP, 全域弹性公网IP标签, 全域弹性公网IP段, 全域弹性公网IP段标签, 全域弹性公网IP池, 接入点, 掩码限制, 配额

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddGeipSegmentTags` | POST | `/v3/global-eip-segment/{resource_id}/tags` | 添加全域弹性公网IP段标签 |
| `AddGlobalEipTags` | POST | `/v3/global-eip/{resource_id}/tags` | 添加全域弹性公网IP标签 |
| `AddInternetBandwidthTags` | POST | `/v3/internet-bandwidth/{resource_id}/tags` | 添加全域公网带宽标签 |
| `AssociateGeipSegmentInstance` | POST | `/v3/{domain_id}/global-eip-segments/{global_eip_segment_id}/associate-instance` | 全域弹性公网IP段绑定后端实例 |
| `AssociateInstance` | POST | `/v3/{domain_id}/global-eips/{global_eip_id}/associate-instance` | 绑定后端实例 |
| `AttachInternetBandwidth` | POST | `/v3/{domain_id}/global-eips/{global_eip_id}/attach-internet-bandwidth` | 绑定全域公网带宽 |
| `BatchAttachGeipSegmentInternetBandwidth` | POST | `/v3/{domain_id}/global-eip-segments/batch-attach-internet-bandwidths` | 全域弹性公网IP段批量绑定全域公网带宽 |
| `BatchAttachInternetBandwidth` | POST | `/v3/{domain_id}/global-eips/batch-attach-internet-bandwidths` | 批量绑定全域公网带宽 |
| `BatchCreateGeipSegmentTags` | POST | `/v3/global-eip-segment/{resource_id}/tags/create` | 批量添加全域弹性公网IP段标签 |
| `BatchCreateGlobalEip` | POST | `/v3/{domain_id}/global-eips/batch-create` | 批量创建全域弹性公网IP |
| `BatchCreateGlobalEipTags` | POST | `/v3/global-eip/{resource_id}/tags/create` | 批量添加全域弹性公网IP标签 |
| `BatchCreateInternetBandwidth` | POST | `/v3/{domain_id}/geip/internet-bandwidths/batch-create` | 批量创建全域公网带宽 |
| `BatchCreateInternetBandwidthTags` | POST | `/v3/internet-bandwidth/{resource_id}/tags/create` | 批量添加全域公网带宽标签 |
| `BatchDeleteGeipSegmentTags` | POST | `/v3/global-eip-segment/{resource_id}/tags/delete` | 批量删除全域弹性公网IP段标签 |
| `BatchDeleteGlobalEipTags` | POST | `/v3/global-eip/{resource_id}/tags/delete` | 批量删除全域弹性公网IP标签 |
| `BatchDeleteInternetBandwidthTags` | POST | `/v3/internet-bandwidth/{resource_id}/tags/delete` | 批量删除全域公网带宽标签 |
| `BatchDetachGeipSegmentInternetBandwidth` | POST | `/v3/{domain_id}/global-eip-segments/batch-detach-internet-bandwidths` | 全域弹性公网IP段批量解绑全域公网带宽 |
| `BatchDetachInternetBandwidth` | POST | `/v3/{domain_id}/global-eips/batch-detach-internet-bandwidths` | 批量解绑全域公网带宽 |
| `CountGlobalEips` | GET | `/v3/{domain_id}/global-eips/count` | 查询全域弹性公网IP个数 |
| `CountGlobalEipSegment` | GET | `/v3/{domain_id}/global-eip-segments/count` | 查询全域弹性公网IP段个数 |
| `CountInternetBandwidth` | GET | `/v3/{domain_id}/geip/internet-bandwidths/count` | 查询全域公网带宽个数 |
| `CreateGlobalEip` | POST | `/v3/{domain_id}/global-eips` | 创建全域弹性公网IP |
| `CreateGlobalEipSegment` | POST | `/v3/{domain_id}/global-eip-segments` | 创建全域弹性公网IP段 |
| `CreateInternetBandwidth` | POST | `/v3/{domain_id}/geip/internet-bandwidths` | 创建全域公网带宽 |
| `CreateUserDisclaimer` | POST | `/v3/{domain_id}/geip/user-disclaimer-records` | 创建租户签署免责条款 |
| `DeleteGeipSegmentTag` | DELETE | `/v3/global-eip-segment/{resource_id}/tags/{tag_key}` | 删除全域弹性公网IP段标签 |
| `DeleteGlobalEip` | DELETE | `/v3/{domain_id}/global-eips/{global_eip_id}` | 删除全域弹性公网IP |
| `DeleteGlobalEipSegment` | DELETE | `/v3/{domain_id}/global-eip-segments/{global_eip_segment_id}` | 删除全域弹性公网IP段 |
| `DeleteGlobalEipTag` | DELETE | `/v3/global-eip/{resource_id}/tags/{tag_key}` | 删除全域弹性公网IP标签 |
| `DeleteInternetBandwidth` | DELETE | `/v3/{domain_id}/geip/internet-bandwidths/{internet_bandwidth_id}` | 删除全域公网带宽 |

... and 36 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
