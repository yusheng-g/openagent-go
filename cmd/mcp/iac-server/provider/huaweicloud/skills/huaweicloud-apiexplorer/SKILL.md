---
name: huaweicloud-apiexplorer
description: HuaweiCloud APIExplorer API guide. 6 APIs covering API.
---

# HuaweiCloud APIExplorer API Guide

6 APIs. Tags: API

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ListApis` | GET | `/v2/apis` | 获取API列表 |
| `ListGroups` | GET | `/v1/groups` | 获取指定产品的分组信息 |
| `ListProductsV4` | GET | `/v4/products` | 查询云服务列表 |
| `ListRegionsV4` | GET | `/v4/regions` | 获取region列表 |
| `ShowApi` | GET | `/v3/apis/detail` | 获取API详情 |
| `ShowMockData` | GET | `/v1/mock/{product_short}/{api_name}` | 获取该api的模拟数据 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
