---
name: huaweicloud-koophone
description: HuaweiCloud KooPhone API guide. 21 APIs covering 实例使用, 实例管理, 实例订购, 应用管理.
---

# HuaweiCloud KooPhone API Guide

21 APIs. Tags: 实例使用, 实例管理, 实例订购, 应用管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AsyncInvokeInstance` | POST | `/instances/async-command` | 实例执行异步命令 |
| `BatchBackupInstances` | POST | `/instances/backup` | 实例备份 |
| `BatchPrepareInstances` | POST | `/instances/prepare` | 实例批量准备 |
| `BatchRebootInstance` | POST | `/instances/reboot` | 实例重启 |
| `BatchResetInstance` | POST | `/instances/batch-reset` | 实例批量重置 |
| `BatchShowInstance` | POST | `/instances/batch-query-status` | 实例状态批量查询 |
| `BatchShowSku` | GET | `/instances/sku` | 可售实例sku批量查询 |
| `CancelInstance` | POST | `/instances/unassign` | 实例取消分配 |
| `CreateInstance` | POST | `/instances/create` | 实例开通接口 |
| `DeleteInstance` | POST | `/instances/delete` | 实例删除 |
| `ExecuteInstanceAuthToken` | POST | `/instances/{instance_id}/auth` | 租户实例串流前获取设备的device_token |
| `ExecuteJob` | GET | `/instances/tasks/{task_id}` | 实例执行任务查询 |
| `InstallApp` | POST | `/instances/app/install` | 实例安装app |
| `ListInstanceApp` | GET | `/instances/{instance_id}/app-list` | 应用列表查询 |
| `ListInstances` | GET | `/instances/batch-query` | 实例批量查询 |
| `ProvisionInstance` | POST | `/instances/assign` | 实例分配 |
| `SetVideo` | PUT | `/instances/video-setting` | 实例视频设置 |
| `ShowProgress` | POST | `/instances/prepare-progress` | 实例准备进度 |
| `StopInstancesSession` | POST | `/instances/session/release` | 实例释放会话 |
| `StopInstancesStreaming` | POST | `/instances/streaming/stop` | 实例停止串流 |
| `SyncInvokeInstance` | POST | `/instances/sync-command` | 实例执行同步命令 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
