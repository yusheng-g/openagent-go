---
name: huaweicloud-mapds
description: HuaweiCloud MapDS API guide. 5 APIs covering 凭证管理, 获取地图瓦片.
---

# HuaweiCloud MapDS API Guide

5 APIs. Tags: 凭证管理, 获取地图瓦片

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateCredential` | POST | `/v1/map/credentials` | 创建凭证 |
| `CreateSasToken` | POST | `/v1/map/sastoken` | 创建SAS Token |
| `DeleteCedential` | DELETE | `/v1/map/credentials/{clientid}` | 删除凭证 |
| `ShowCredential` | GET | `/v1/map/credentials` | 查询凭证 |
| `ShowMapTile` | GET | `/v1/map/tile/{z}/{x}/{y}` | 获取地图瓦片 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
