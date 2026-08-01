---
name: huaweicloud-vas
description: HuaweiCloud VAS API guide. 7 APIs covering 服务作业管理.
---

# HuaweiCloud VAS API Guide

7 APIs. Tags: 服务作业管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateTasks` | POST | `/{project_id}/services/{service_name}/tasks` | 创建服务作业 |
| `DeleteTask` | DELETE | `/{project_id}/services/{service_name}/tasks/{task_id}` | 删除服务作业 |
| `ListTasksDetails` | GET | `/{project_id}/services/{service_name}/tasks` | 获取服务作业列表 |
| `ShowTask` | GET | `/{project_id}/services/{service_name}/tasks/{task_id}` | 查询服务作业 |
| `StartTask` | POST | `/{project_id}/services/{service_name}/tasks/{task_id}/start` | 启动服务作业 |
| `StopTask` | POST | `/{project_id}/services/{service_name}/tasks/{task_id}/stop` | 停止服务作业 |
| `UpdateTask` | PUT | `/{project_id}/services/{service_name}/tasks/{task_id}` | 更新服务作业 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
