---
name: huaweicloud-vias
description: HuaweiCloud VIAS API guide. 29 APIs covering 任务中心, 算法中心, 视频中心, 运维中心.
---

# HuaweiCloud VIAS API Guide

29 APIs. Tags: 任务中心, 算法中心, 视频中心, 运维中心

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchStartStopTask` | PUT | `/v2/{project_id}/batch-tasks/{id}/action/{command}` | 启动/停止批量配置任务 |
| `CreateBatchTask` | POST | `/v2/{project_id}/batch-tasks` | 新增批量任务 |
| `CreateEdgePool` | POST | `/v2/{project_id}/edge-pools` | 创建边缘资源池 |
| `CreateTask` | POST | `/v2/{project_id}/services/{service_name}/tasks` | 创建任务 |
| `CreateVideoGroup` | POST | `/v2/{project_id}/video-group` | 创建视频源分组 |
| `CreateVideoSource` | POST | `/v2/{project_id}/source` | 创建视频源 |
| `DeleteBatchTask` | DELETE | `/v2/{project_id}/batch-tasks/{id}` | 删除批量配置任务 |
| `DeleteEdgePool` | DELETE | `/v2/{project_id}/edge-pools/{id}` | 删除边缘资源池 |
| `DeleteTask` | DELETE | `/v2/{project_id}/tasks/{task_id}` | 删除单个任务 |
| `DeleteVideoGroup` | DELETE | `/v2/{project_id}/video-group/{video_group_id}` | 删除视频源分组 |
| `DeleteVideoSource` | DELETE | `/v2/{project_id}/source/{video_source_id}` | 删除视频源 |
| `DeployAlgorithm` | POST | `/v2/{project_id}/algorithm/{alg_id}/deploy` | 部署算法 |
| `ListBatchTasks` | GET | `/v2/{project_id}/batch-tasks` | 获取批量配置任务列表 |
| `ListEdgePools` | GET | `/v2/{project_id}/edge-pools` | 查询边缘资源池列表 |
| `ListTasks` | GET | `/v2/{project_id}/tasks` | 获取任务列表 |
| `ListUserServices` | GET | `/v2/{project_id}/algorithm/services/user` | 我的算法服务列表 |
| `ListVideoGroups` | GET | `/v2/{project_id}/video-group/groups` | 获取视频源分组列表 |
| `ListVideoSources` | GET | `/v2/{project_id}/source/sources` | 获取视频源列表 |
| `ShowEdgePoolInfo` | GET | `/v2/{project_id}/edge-pools/{id}` | 查询边缘资源池详情 |
| `ShowServiceDetail` | GET | `/v2/{project_id}/algorithm/services/{service_id}` | 获取服务详情 |
| `ShowTaskInfo` | GET | `/v2/{project_id}/tasks/{task_id}` | 获取任务详情 |
| `ShowVideoGroupDetail` | GET | `/v2/{project_id}/video-group/{video_group_id}` | 获取视频源分组详情 |
| `ShowVideoSourceDetail` | GET | `/v2/{project_id}/source/sources/{video_source_id}` | 获取视频源详情 |
| `StartStopTask` | PUT | `/v2/{project_id}/tasks/{task_id}/action/{command}` | 任务启停 |
| `StopDeployAlgorithm` | PUT | `/v2/{project_id}/algorithm/{alg_id}/deploy/stop` | 停止算法部署 |
| `UpdateBatchTask` | PUT | `/v2/{project_id}/batch-tasks/{id}` | 修改批量配置任务 |
| `UpdateTask` | PUT | `/v2/{project_id}/tasks/{task_id}` | 修改任务 |
| `UpdateVideoGroup` | PUT | `/v2/{project_id}/video-group/{video_group_id}` | 更新视频源分组 |
| `UpdateVideoSource` | PUT | `/v2/{project_id}/source/{video_source_id}` | 修改视频源 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
