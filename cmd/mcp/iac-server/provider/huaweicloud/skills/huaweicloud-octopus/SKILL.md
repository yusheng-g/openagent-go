---
name: huaweicloud-octopus
description: HuaweiCloud Octopus API guide. 60 APIs covering 仿真任务, 仿真任务配置, 仿真场景, 仿真子任务, 仿真算法.
---

# HuaweiCloud Octopus API Guide

60 APIs. Tags: 仿真任务, 仿真任务配置, 仿真场景, 仿真子任务, 仿真算法, 仿真算法镜像, 作业管理, 作业队列, 内部作业, 场景地图, 扩展文件, 数据包, 数据导入, 算子管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchDeleteJob` | DELETE | `/v1.0/{project_id}/data/jobs` | 批量删除作业 |
| `BatchUpdateJob` | PUT | `/v1.0/{project_id}/data/jobs` | 批量更新作业 |
| `BatchUpdateQueueJob` | PUT | `/v1.0/{project_id}/data/queuing-jobs/batch-update` | 批量更新队列优先级 |
| `CreateCollectionRecord` | POST | `/v1.0/{project_id}/data/import-records` | 创建数据包导入任务 |
| `CreateJob` | POST | `/v1.0/{project_id}/data/jobs` | 创建作业 |
| `CreateProcessor` | POST | `/v1.0/{project_id}/data/processors` | 创建算子 |
| `CreateSimAlgorithmImages` | POST | `/v2/{project_id}/sim/pm/algorithm-images` | 创建算法镜像 |
| `CreateSimAlgorithms` | POST | `/v2/{project_id}/sim/pm/algorithms` | 创建算法 |
| `CreateSimBatchConfigs` | POST | `/v2/{project_id}/sim/pm/batch-configs` | 创建仿真任务配置 |
| `CreateSimBatches` | POST | `/v2/{project_id}/sim/pm/batches` | 创建仿真任务 |
| `CreateSimExtensions` | POST | `/v2/{project_id}/sim/pm/extensions` | 创建扩展文件 |
| `CreateSimSmMaps` | POST | `/v2/{project_id}/sim/sm/maps` | 创建场景地图 |
| `CreateSimSmScenarios` | POST | `/v2/{project_id}/sim/sm/scenarios` | 创建仿真场景 |
| `CreateSimSmScenariosFiles` | POST | `/v2/{project_id}/sim/sm/scenarios/{parent_lookup_id}/files` | 创建场景文件 |
| `CreateSystemJob` | POST | `/v1.0/{project_id}/data/system-jobs` | 创建内部作业 |
| `DeleteCollectionById` | DELETE | `/v1.0/{project_id}/data/import-records/{id}` | 删除导入任务 |
| `DeleteJob` | DELETE | `/v1.0/{project_id}/data/jobs/{id}` | 删除作业 |
| `DeletePackageById` | DELETE | `/v1.0/{project_id}/data/packages/{id}` | 永久删除数据包 |
| `DeleteProcessor` | DELETE | `/v1.0/{project_id}/data/processors/{id}` | 删除算子 |
| `DeleteSimAlgorithmImages` | DELETE | `/v2/{project_id}/sim/pm/algorithm-images/{id}` | 删除算法镜像 |
| `DeleteSimAlgorithms` | DELETE | `/v2/{project_id}/sim/pm/algorithms/{id}` | 删除算法 |
| `DeleteSimBatchConfigs` | DELETE | `/v2/{project_id}/sim/pm/batch-configs/{id}` | 删除仿真任务配置 |
| `DeleteSimBatches` | DELETE | `/v2/{project_id}/sim/pm/batches/{id}` | 删除仿真任务 |
| `DeleteSimExtensions` | DELETE | `/v2/{project_id}/sim/pm/extensions/{id}` | 删除扩展文件 |
| `GetCollectionById` | GET | `/v1.0/{project_id}/data/import-records/{id}` | 获取导入任务详情 |
| `GetPackageById` | GET | `/v1.0/{project_id}/data/packages/{id}` | 获取数据包详情 |
| `GetSystemJobDetail` | GET | `/v1.0/{project_id}/data/system-jobs/{id}` | 查询内部作业详情 |
| `GetSystemJobList` | GET | `/v1.0/{project_id}/data/system-jobs` | 查询内部作业列表 |
| `ListCollectionRecords` | GET | `/v1.0/{project_id}/data/import-records` | 获取导入任务列表 |
| `ListJobs` | GET | `/v1.0/{project_id}/data/jobs` | 查询作业列表 |

... and 30 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
