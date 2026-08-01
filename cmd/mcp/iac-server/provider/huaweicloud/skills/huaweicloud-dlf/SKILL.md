---
name: huaweicloud-dlf
description: HuaweiCloud DLF API guide. 39 APIs covering 作业相关的API, 目录树相关的API, 脚本相关的API, 资源相关的API, 连接相关的API.
---

# HuaweiCloud DLF API Guide

39 APIs. Tags: 作业相关的API, 目录树相关的API, 脚本相关的API, 资源相关的API, 连接相关的API

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CancelScript` | POST | `/v1/{project_id}/scripts/{script_name}/instances/{instance_id}/stop` | 停止脚本实例的执行 |
| `CreateConnection` | POST | `/v1/{project_id}/connections` | 创建连接 |
| `CreateJob` | POST | `/v1/{project_id}/jobs` | 创建作业 |
| `CreateResource` | POST | `/v1/{project_id}/resources` | 创建资源 |
| `CreateScript` | POST | `/v1/{project_id}/scripts` | 创建脚本 |
| `DeleteConnction` | DELETE | `/v1/{project_id}/connections/{connection_name}` | 删除连接 |
| `DeleteJob` | DELETE | `/v1/{project_id}/jobs/{job_name}` | 删除作业 |
| `DeleteResource` | DELETE | `/v1/{project_id}/resources/{resource_id}` | 删除资源 |
| `DeleteScript` | DELETE | `/v1/{project_id}/scripts/{script_name}` | 删除脚本 |
| `ExecuteScript` | POST | `/v1/{project_id}/scripts/{script_name}/execute` | 执行脚本 |
| `ExportConnections` | POST | `/v1/{project_id}/connections/export` | 导出连接 |
| `ExportJob` | POST | `/v1/{project_id}/jobs/{job_name}/export` | 导出作业 |
| `ExportJobList` | POST | `/v1/{project_id}/jobs/batch-export` | 批量导出作业 |
| `ImportConnections` | POST | `/v1/{project_id}/connections/import` | 导入连接 |
| `ImportJob` | POST | `/v1/{project_id}/jobs/import` | 导入作业 |
| `ListConnections` | GET | `/v1/{project_id}/connections` | 查询连接列表 |
| `ListJobInstances` | GET | `/v1/{project_id}/jobs/instances/detail` | 查询作业实例列表 |
| `ListJobs` | GET | `/v1/{project_id}/jobs` | 查询作业列表 |
| `ListResources` | GET | `/v1/{project_id}/resources` | 查询资源列表 |
| `ListScriptResults` | GET | `/v1/{project_id}/scripts/{script_name}/instances/{instance_id}` | 查询脚本实例执行结果 |
| `ListScripts` | GET | `/v1/{project_id}/scripts` | 查询脚本列表 |
| `ListSystemTasks` | GET | `/v1/{project_id}/system-tasks/{task_id}` | 查询系统任务详情 |
| `RestoreJobInstance` | POST | `/v1/{project_id}/jobs/{job_name}/instances/{instance_id}/restart` | 重跑作业实例 |
| `RunOnce` | POST | `/v1/{project_id}/jobs/{job_name}/run-immediate` | 单次执行作业 |
| `ShowConnection` | GET | `/v1/{project_id}/connections/{connection_name}` | 查询连接详情 |
| `ShowDirectoryTree` | GET | `/v1.0/{project_id}/directory/tree` | 查询目录树 |
| `ShowFileInfo` | POST | `/v1/{project_id}/jobs/check-file` | 检查导入作业文件中的作业和脚本 |
| `ShowJob` | GET | `/v1/{project_id}/jobs/{job_name}` | 查询作业详情 |
| `ShowJobInstance` | GET | `/v1/{project_id}/jobs/{job_name}/instances/{instance_id}` | 查询作业实例详情 |
| `ShowJobStatus` | GET | `/v1/{project_id}/jobs/{job_name}/status` | 查询实时作业的运行状态 |

... and 9 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
