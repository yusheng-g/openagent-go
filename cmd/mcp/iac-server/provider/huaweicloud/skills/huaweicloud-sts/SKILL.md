---
name: huaweicloud-sts
description: HuaweiCloud STS API guide. 5 APIs covering 临时安全凭证, 调用者信息查询, 鉴权结果查询.
---

# HuaweiCloud STS API Guide

5 APIs. Tags: 临时安全凭证, 调用者信息查询, 鉴权结果查询

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AssumeAgency` | POST | `/v5/agencies/assume` | 通过委托或者信任委托获取临时安全凭证 |
| `AssumeAgencyWithOIDC` | POST | `/v5/agencies/assume-with-oidc` | 通过使用OIDC协议SSO的信任委托获取临时安全凭证 |
| `AssumeAgencyWithSAML` | POST | `/v5/agencies/assume-with-saml` | 通过使用SAML协议SSO的信任委托获取临时安全凭证 |
| `DecodeAuthorizationMessage` | POST | `/v5/decode-authorization-message` | 解密鉴权失败的原因 |
| `GetCallerIdentity` | GET | `/v5/caller-identity` | 获取调用者身份信息 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
