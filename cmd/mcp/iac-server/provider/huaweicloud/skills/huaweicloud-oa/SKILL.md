---
name: huaweicloud-oa
description: HuaweiCloud OA API guide. 3 APIs covering AvailabilityCheckOpenApi.
---

# HuaweiCloud OA API Guide

3 APIs. Tags: AvailabilityCheckOpenApi

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ShowCheckItemList` | GET | `/v3/optimization-advisor/check-items` | 获取可用性检查项列表 |
| `ShowCheckItemResult` | GET | `/v3/optimization-advisor/check-item-result` | 获取可用性检查项详情 |
| `StartItemCheck` | POST | `/v3/optimization-advisor/item-check/start` | 启动单检查项检查 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
