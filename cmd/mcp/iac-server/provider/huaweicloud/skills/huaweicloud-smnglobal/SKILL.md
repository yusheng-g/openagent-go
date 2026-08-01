---
name: huaweicloud-smnglobal
description: HuaweiCloud SMNGLOBAL API guide. 4 APIs covering 订阅用户.
---

# HuaweiCloud SMNGLOBAL API Guide

4 APIs. Tags: 订阅用户

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateSubscriptionUser` | POST | `/v2/{domain_id}/subscription-users` | 添加订阅用户 |
| `DeleteSubscriptionUser` | DELETE | `/v2/{domain_id}/subscription-users/{id}` | 删除订阅用户 |
| `ListSubscriptionUser` | GET | `/v2/{domain_id}/subscription-users` | 查询订阅用户列表 |
| `UpdateSubscriptionUser` | PUT | `/v2/{domain_id}/subscription-users/{id}` | 更新订阅用户 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
