---
name: huaweicloud-orgid
description: HuaweiCloud OrgID API guide. 3 APIs covering Cas30Service, OAUTH.
---

# HuaweiCloud OrgID API Guide

3 APIs. Tags: Cas30Service, OAUTH

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ShowOauth2Token` | POST | `/oauth2/token` | 用户级Token获取 |
| `ShowOauth2UserInfo` | GET | `/oauth2/userinfo` | 用户信息获取 |
| `ValidateService` | GET | `/cas/p3/serviceValidate` | 验证票据接口 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
