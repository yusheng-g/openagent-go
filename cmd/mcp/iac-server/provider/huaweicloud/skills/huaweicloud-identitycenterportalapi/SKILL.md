---
name: huaweicloud-identitycenterportalapi
description: HuaweiCloud IdentityCenterPortalAPI API guide. 4 APIs covering 令牌管理, 凭证管理, 委托管理, 账户管理.
---

# HuaweiCloud IdentityCenterPortalAPI API Guide

4 APIs. Tags: 令牌管理, 凭证管理, 委托管理, 账户管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `GetAgencyCredentials` | GET | `/v1/credentials` | 获取委托凭证 |
| `ListAccountAgencies` | GET | `/v1/assigned-agencies` | 列出账号委托 |
| `ListAccounts` | GET | `/v1/assigned-accounts` | 列出账号 |
| `Logout` | POST | `/v1/logout` | 登出用户 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
