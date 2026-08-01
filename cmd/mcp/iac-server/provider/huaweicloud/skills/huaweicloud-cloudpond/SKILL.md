---
name: huaweicloud-cloudpond
description: HuaweiCloud CloudPond API guide. 22 APIs covering 区域, 地区, 存储档位, 存储池, 存储类型.
---

# HuaweiCloud CloudPond API Guide

22 APIs. Tags: 区域, 地区, 存储档位, 存储池, 存储类型, 服务器, 机柜, 网络设备, 边缘小站, 边缘小站监控, 配额, 销售周期, 销售商品

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateEdgeSite` | POST | `/v1/{domain_id}/edge-sites` | 创建边缘小站 |
| `DeleteEdgeSite` | DELETE | `/v1/{domain_id}/edge-sites/{site_id}` | 删除边缘小站 |
| `ListEdgeSiteMetrics` | GET | `/v1/{domain_id}/edge-sites/{site_id}/metric-data` | 查看站点容量信息 |
| `ListEdgeSites` | GET | `/v1/{domain_id}/edge-sites` | 查询边缘小站列表 |
| `ListNetworkDeviceOfferings` | GET | `/v2/{domain_id}/network-device-offerings` | 查询网络设备商品列表 |
| `ListNetworkDevices` | GET | `/v2/{domain_id}/network-devices` | 查询网络设备列表 |
| `ListQuotas` | GET | `/v1/{domain_id}/quotas` | 查询配额 |
| `ListRacks` | GET | `/v1/{domain_id}/racks` | 查询机柜列表 |
| `ListSaleCycles` | GET | `/v2/{domain_id}/sale-cycles` | 查询可购买的销售周期 |
| `ListServerOfferings` | GET | `/v2/{domain_id}/server-offerings` | 查询服务器商品列表 |
| `ListServers` | GET | `/v2/{domain_id}/servers` | 查询服务器列表 |
| `ListStorageGears` | GET | `/v2/{domain_id}/storage-gears` | 查询存储计费档位 |
| `ListStoragePools` | GET | `/v1/{domain_id}/storage-pools` | 查询存储池列表 |
| `ListStorageTypes` | GET | `/v2/{domain_id}/storage-types` | 查询存储类型列表 |
| `ListSupportedRegions` | GET | `/v1/{domain_id}/regions` | 查询支持的区域列表 |
| `ListSupportedZones` | GET | `/v1/{domain_id}/zones` | 查询支持的地区列表 |
| `ShowEdgeSite` | GET | `/v1/{domain_id}/edge-sites/{site_id}` | 查询边缘小站详情 |
| `ShowNetworkDevice` | GET | `/v2/{domain_id}/network-devices/{network_device_id}` | 查询网络设备详情 |
| `ShowRack` | GET | `/v1/{domain_id}/racks/{id}` | 查询机柜详情 |
| `ShowServer` | GET | `/v2/{domain_id}/servers/{server_id}` | 查询服务器详情 |
| `ShowStoragePool` | GET | `/v1/{domain_id}/storage-pools/{id}` | 查询存储池详情 |
| `UpdateEdgeSite` | PUT | `/v1/{domain_id}/edge-sites/{site_id}` | 更新边缘小站 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
