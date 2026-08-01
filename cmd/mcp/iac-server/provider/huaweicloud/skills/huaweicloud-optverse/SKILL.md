---
name: huaweicloud-optverse
description: HuaweiCloud OptVerse API guide. 4 APIs covering OptVerse Service.
---

# HuaweiCloud OptVerse API Guide

4 APIs. Tags: OptVerse Service

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateTask` | POST | `/v2/{project_id}/optverse/{service_type}/tasks` | 创建任务 |
| `DeleteTask` | DELETE | `/v2/{project_id}/optverse/{service_type}/tasks/{task_id}` | 删除任务 |
| `ListTask` | GET | `/v2/{project_id}/optverse/{service_type}/tasks` | 查询任务列表 |
| `ShowTask` | GET | `/v2/{project_id}/optverse/{service_type}/tasks/{task_id}` | 获取任务详情 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
