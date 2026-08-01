---
name: huaweicloud-cts
description: HuaweiCloud CTS API guide. 16 APIs covering 事件管理, 关键操作通知管理, 其它接口, 标签管理, 追踪器管理.
---

# HuaweiCloud CTS API Guide

16 APIs. Tags: 事件管理, 关键操作通知管理, 其它接口, 标签管理, 追踪器管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateResourceTags` | POST | `/v3/{project_id}/{resource_type}/{resource_id}/tags/create` | 批量添加CTS资源标签 |
| `BatchDeleteResourceTags` | DELETE | `/v3/{project_id}/{resource_type}/{resource_id}/tags/delete` | 批量删除CTS资源标签 |
| `CheckObsBuckets` | POST | `/v3/{domain_id}/checkbucket` | 检查已经配置OBS桶是否可以成功转储 |
| `CreateNotification` | POST | `/v3/{project_id}/notifications` | 创建关键操作通知 |
| `CreateTracker` | POST | `/v3/{project_id}/tracker` | 创建追踪器 |
| `DeleteNotification` | DELETE | `/v3/{project_id}/notifications` | 删除关键操作通知 |
| `DeleteTracker` | DELETE | `/v3/{project_id}/trackers` | 删除追踪器 |
| `ListNotifications` | GET | `/v3/{project_id}/notifications/{notification_type}` | 查询关键操作通知 |
| `ListOperations` | GET | `/v3/{project_id}/operations` | 查询云服务的全量操作列表 |
| `ListQuotas` | GET | `/v3/{project_id}/quotas` | 查询租户追踪器配额信息 |
| `ListTraceResources` | GET | `/v3/{domain_id}/resources` | 查询事件的资源类型列表 |
| `ListTraces` | GET | `/v3/{project_id}/traces` | 查询事件列表 |
| `ListTrackers` | GET | `/v3/{project_id}/trackers` | 查询追踪器 |
| `ListUserResources` | GET | `/v3/{project_id}/user-resources` | 查询30天事件的操作用户列表 |
| `UpdateNotification` | PUT | `/v3/{project_id}/notifications` | 修改关键操作通知 |
| `UpdateTracker` | PUT | `/v3/{project_id}/tracker` | 修改追踪器 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
