---
name: huaweicloud-pangulargemodels
description: HuaweiCloud PanguLargeModels API guide. 2 APIs covering Completions.
---

# HuaweiCloud PanguLargeModels API Guide

2 APIs. Tags: Completions

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ExecuteChatCompletion` | POST | `/v2/{project_id}/pools/{pool_id}/deployments/{deployment_id}/chat/completions` | 对话问答 |
| `ExecuteTextCompletion` | POST | `/v1/{project_id}/deployments/{deployment_id}/text/completions` | 通用文本 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
