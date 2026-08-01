---
name: huaweicloud-identitycenteroidc
description: HuaweiCloud IdentityCenterOIDC API guide. 3 APIs covering 令牌管理, 客户端管理, 设备授权管理.
---

# HuaweiCloud IdentityCenterOIDC API Guide

3 APIs. Tags: 令牌管理, 客户端管理, 设备授权管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateToken` | POST | `/v1/tokens` | 创建令牌 |
| `RegisterClient` | POST | `/v1/clients` | 注册客户端 |
| `StartDeviceAuthorization` | POST | `/v1/device/authorize` | 请求设备授权 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
