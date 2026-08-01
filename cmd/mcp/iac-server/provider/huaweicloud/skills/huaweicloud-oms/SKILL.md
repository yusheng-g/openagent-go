---
name: huaweicloud-oms
description: HuaweiCloud OMS API guide. 35 APIs covering 云服务, 区域, 同步任务, 查询API版本信息, 桶操作.
---

# HuaweiCloud OMS API Guide

35 APIs. Tags: 云服务, 区域, 同步任务, 查询API版本信息, 桶操作, 迁移任务管理, 迁移任务组管理, 隐私协议

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchUpdateTasks` | POST | `/v2/{project_id}/tasks/batch-update` | 批量更新任务 |
| `CheckPrefix` | POST | `/v2/{project_id}/objectstorage/buckets/prefix` | 检查前缀是否在源端桶中存在 |
| `CheckUrlSourceListFileFormat` | POST | `/v2/{project_id}/objectstorage/buckets/url-source-list-file` | 检查url来源列表文件格式 |
| `CreateSyncEvents` | POST | `/v2/{project_id}/sync-tasks/{sync_task_id}/events` | 创建同步事件 |
| `CreateSyncTask` | POST | `/v2/{project_id}/sync-tasks` | 创建同步任务 |
| `CreateTask` | POST | `/v2/{project_id}/tasks` | 创建迁移任务 |
| `CreateTaskGroup` | POST | `/v2/{project_id}/taskgroups` | 创建迁移任务组 |
| `DeleteSyncTask` | DELETE | `/v2/{project_id}/sync-tasks/{sync_task_id}` | 删除同步任务 |
| `DeleteTask` | DELETE | `/v2/{project_id}/tasks/{task_id}` | 删除迁移任务 |
| `DeleteTaskGroup` | DELETE | `/v2/{project_id}/taskgroups/{group_id}` | 删除指定id的迁移任务组 |
| `ListApiVersions` | GET | `/` | 查询API版本信息列表 |
| `ListSyncTasks` | GET | `/v2/{project_id}/sync-tasks` | 查询同步任务列表 |
| `ListSyncTaskStatistic` | GET | `/v2/{project_id}/sync-tasks/{sync_task_id}/statistics` | 查询指定ID的同步任务统计数据 |
| `ListTaskGroup` | GET | `/v2/{project_id}/taskgroups` | 查询迁移任务组列表 |
| `ListTasks` | GET | `/v2/{project_id}/tasks` | 查询迁移任务列表 |
| `RetryTaskGroup` | PUT | `/v2/{project_id}/taskgroups/{group_id}/retry` | 对已经失败的指定id迁移任务组进行重启 |
| `ShowApiInfo` | GET | `/{version}` | 查询指定API版本信息 |
| `ShowBucketList` | POST | `/v2/{project_id}/objectstorage/buckets` | 查询桶列表 |
| `ShowBucketObjects` | POST | `/v2/{project_id}/objectstorage/buckets/objects` | 查询桶对象列表 |
| `ShowBucketRegion` | POST | `/v2/{project_id}/objectstorage/buckets/regions` | 查询桶对应的region |
| `ShowCdnInfo` | POST | `/v2/{project_id}/objectstorage/buckets/cdn-info` | 查桶对应的CDN信息 |
| `ShowCloudType` | GET | `/v2/{project_id}/objectstorage/cloud-type` | 查询所有支持的云厂商 |
| `ShowRegionInfo` | GET | `/v2/{project_id}/objectstorage/data-center` | 查询云厂商支持的region |
| `ShowSyncTask` | GET | `/v2/{project_id}/sync-tasks/{sync_task_id}` | 查询指定ID的同步任务详情 |
| `ShowTask` | GET | `/v2/{project_id}/tasks/{task_id}` | 查询指定ID的任务详情 |
| `ShowTaskGroup` | GET | `/v2/{project_id}/taskgroups/{group_id}` | 获取指定id的taskgroup信息 |
| `StartSyncTask` | POST | `/v2/{project_id}/sync-tasks/{sync_task_id}/start` | 启动同步任务 |
| `StartTask` | POST | `/v2/{project_id}/tasks/{task_id}/start` | 启动迁移任务 |
| `StartTaskGroup` | PUT | `/v2/{project_id}/taskgroups/{group_id}/start` | 恢复指定id的迁移任务组 |
| `StopSyncTask` | POST | `/v2/{project_id}/sync-tasks/{sync_task_id}/stop` | 暂停同步任务 |

... and 5 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
