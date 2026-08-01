---
name: huaweicloud-oroas
description: HuaweiCloud OROAS API guide. 4 APIs covering OROAS Service.
---

# HuaweiCloud OROAS API Guide

4 APIs. Tags: OROAS Service

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateTask` | POST | `/v1/{project_id}/oroas/{service_group}/{service_type}/tasks` | 创建任务 |
| `DeleteTask` | DELETE | `/v1/{project_id}/oroas/{service_group}/{service_type}/tasks/{task_id}` | 删除任务 |
| `ListTask` | GET | `/v1/{project_id}/oroas/{service_group}/{service_type}/tasks` | 查询任务列表 |
| `ShowTask` | GET | `/v1/{project_id}/oroas/{service_group}/{service_type}/tasks/{task_id}` | 获取任务详情 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
