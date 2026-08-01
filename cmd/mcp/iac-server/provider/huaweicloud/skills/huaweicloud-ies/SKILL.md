---
name: huaweicloud-ies
description: HuaweiCloud IES API guide. 12 APIs covering 区域, 存储池, 机柜, 边缘小站, 边缘小站监控.
---

# HuaweiCloud IES API Guide

12 APIs. Tags: 区域, 存储池, 机柜, 边缘小站, 边缘小站监控, 配额

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateEdgeSite` | POST | `/v1/{domain_id}/edge-sites` | 创建边缘小站 |
| `DeleteEdgeSite` | DELETE | `/v1/{domain_id}/edge-sites/{site_id}` | 删除边缘小站 |
| `ListEdgeSiteMetrics` | GET | `/v1/{domain_id}/edge-sites/{site_id}/metric-data` | 查看站点容量信息 |
| `ListEdgeSites` | GET | `/v1/{domain_id}/edge-sites` | 查询边缘小站列表 |
| `ListQuotas` | GET | `/v1/{domain_id}/quotas` | 查询配额 |
| `ListRacks` | GET | `/v1/{domain_id}/racks` | 查询机柜列表 |
| `ListStoragePools` | GET | `/v1/{domain_id}/storage-pools` | 查询存储池列表 |
| `ListSupportedRegions` | GET | `/v1/{domain_id}/regions` | 查询支持的区域列表 |
| `ShowEdgeSite` | GET | `/v1/{domain_id}/edge-sites/{site_id}` | 查询边缘小站详情 |
| `ShowRack` | GET | `/v1/{domain_id}/racks/{id}` | 查询机柜详情 |
| `ShowStoragePool` | GET | `/v1/{domain_id}/storage-pools/{id}` | 查询存储池详情 |
| `UpdateEdgeSite` | PUT | `/v1/{domain_id}/edge-sites/{site_id}` | 更新边缘小站 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
