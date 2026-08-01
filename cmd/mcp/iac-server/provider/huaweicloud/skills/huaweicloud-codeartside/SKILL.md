---
name: huaweicloud-codeartside
description: HuaweiCloud CodeArtsIDE API guide. 2 APIs covering 发布版本查询, 白名单管理.
---

# HuaweiCloud CodeArtsIDE API Guide

2 APIs. Tags: 发布版本查询, 白名单管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ShowLatestUpgradableRelease` | GET | `/v1/release-info/latest` | 查询升级版本 |
| `ValidateWhitelistUser` | POST | `/v1/config/users/check` | 是否白名单用户 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
