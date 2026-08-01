---
name: huaweicloud-koomap
description: HuaweiCloud KooMap API guide. 40 APIs covering 卫星影像任务管理, 卫星影像数据管理, 卫星影像用量统计, 实景三维任务管理, 实景三维刺点管理.
---

# HuaweiCloud KooMap API Guide

40 APIs. Tags: 卫星影像任务管理, 卫星影像数据管理, 卫星影像用量统计, 实景三维任务管理, 实景三维刺点管理, 实景三维数据管理, 实景三维用量统计, 实景三维精修后处理任务管理, 工作共享空间, 空间定位, 空间导航

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddSpurPoint` | POST | `/v1/real3d/spur` | 新增图片上的刺点 |
| `CreateCommonWorkspace` | POST | `/v1/workspaces` | 创建工作共享空间 |
| `CreateMarkerInfo` | POST | `/v1/real3d/spur/{task_id}/markerinfo` | 生成刺点文件 |
| `CreateReal3DSubTask` | POST | `/v1/real3d/{workspace_id}/tasks/{task_id}/subtasks` | 创建实景三维精修后处理任务 |
| `CreateReal3DTask` | POST | `/v1/real3d/{workspace_id}/tasks` | 创建实景三维建模任务 |
| `CreateTask` | POST | `/v1/kmp-control/tasks` | 新建任务 |
| `DeleteCommonWorkspace` | DELETE | `/v1/workspaces/{workspace_id}` | 删除工作共享空间 |
| `DeleteReal3DProduct` | DELETE | `/v1/real3d/products/{product_id}` | 删除实景三维成果影像 |
| `DeleteReal3DRefineProduct` | DELETE | `/v1/real3d/refineproducts/{refine_product_id}` | 删除实景三维精修后处理成果数据 |
| `DeleteReal3DSubTask` | DELETE | `/v1/real3d/{workspace_id}/tasks/{task_id}/subtasks/{subtask_id}` | 删除实景三维精修后处理任务 |
| `DeleteReal3DTask` | DELETE | `/v1/real3d/{workspace_id}/tasks/{task_id}` | 删除实景三维建模任务 |
| `DeleteSpurPoint` | DELETE | `/v1/real3d/spur` | 删除图片上的刺点 |
| `DeleteTask` | DELETE | `/v1/kmp-control/tasks` | 删除任务 |
| `ListCommonWorkspace` | GET | `/v1/workspaces` | 查询工作共享空间列表 |
| `ListFolder` | GET | `/v1/real3d/folders` | 查询当前租户的倾斜影像列表 |
| `ListImageBaseInfo` | POST | `/v1/kmp-data/imageinfo` | 查询卫星影像基本信息 |
| `ListReal3DProducts` | GET | `/v1/real3d/products` | 查询实景三维成果影像列表 |
| `ListReal3DRefineProducts` | GET | `/v1/real3d/refineproducts` | 查询实景三维精修后处理成果数据列表 |
| `ListReal3DSubTasks` | GET | `/v1/real3d/{workspace_id}/tasks/{task_id}/subtasks` | 分页查询实景三维精修后处理任务列表 |
| `ListSpurPoints` | POST | `/v1/real3d/spurs` | 获取单张图片里的所有刺点信息 |
| `ListTaskInfo` | GET | `/v1/kmp-control/tasks` | 查询任务 |
| `ListTasksInWorkspace` | GET | `/v1/real3d/{workspace_id}/tasks` | 分页查询工作共享空间内实景三维任务列表 |
| `ListUsageInfo` | GET | `/v1/kmp-control/usages` | 查询用量 |
| `ShowReal3DUsage` | GET | `/v1/real3d/usages` | 查询实景三维用量 |
| `ShowSpurCount` | POST | `/v1/real3d/spur/count` | 查询单个像控点的已刺点数量 |
| `ShowTaskOverview` | GET | `/v1/kmp-control/tasks/overview` | 查看任务概览 |
| `ShowTaskOverviewInWorkspace` | GET | `/v1/real3d/{workspace_id}/tasks/overview` | 展示工作共享空间内任务概览 |
| `StartNavi` | POST | `/v1/algo/navi` | AR导航 |
| `StartReal3DSubTask` | POST | `/v1/real3d/{workspace_id}/tasks/{task_id}/subtasks/{subtask_id}/start` | 启动实景三维精修后处理任务 |
| `StartReal3DTask` | POST | `/v1/real3d/{workspace_id}/tasks/{task_id}/start` | 启动实景三维建模任务 |

... and 10 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
