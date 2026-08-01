---
name: huaweicloud-iotanalytics
description: HuaweiCloud IoTAnalytics API guide. 66 APIs covering 存储标签查询, 存储管理, 存储组管理, 实时作业操作, 实时作业管理.
---

# HuaweiCloud IoTAnalytics API Guide

66 APIs. Tags: 存储标签查询, 存储管理, 存储组管理, 实时作业操作, 实时作业管理, 批计算资源管理, 数据源配置, 离线分析作业管理, 离线分析作业运行管理, 离线分析作业运行结果, 离线数据表管理, 管道作业管理, 设备存储数据查询, 设备数据上报, 资产, 资产属性, 资产模型

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddDevData` | POST | `/v1/{project_id}/datasources/{datasource_id}/dev-data` | 通过API数据源上报设备数据 |
| `AddPipelineJob` | POST | `/v1/{project_id}/pipelines` | 新建管道作业 |
| `CreateAssetModel` | POST | `/v1/{project_id}/asset-models` | 创建资产模型 |
| `CreateAssetNew` | POST | `/v1/{project_id}/assets` | 创建资产 |
| `CreateBatchJob` | POST | `/v1/{project_id}/batch/jobs` | 创建离线作业 |
| `CreateComputingResource` | POST | `/v1/{project_id}/batch-computing-resources` | 创建批计算资源 |
| `CreateDatasource` | POST | `/v1/{project_id}/datasources` | 创建数据源 |
| `CreateGroup` | POST | `/v1/{project_id}/data-store-groups` | 创建存储组 |
| `CreateRun` | POST | `/v1/{project_id}/batch/jobs/{job_id}/runs` | 执行离线作业 |
| `CreateStreamingJob` | POST | `/v1/{project_id}/streaming/jobs` | 新建实时作业 |
| `CreateTable` | POST | `/v1/{project_id}/tables` | 创建离线数据表 |
| `DeleteAssetModel` | DELETE | `/v1/{project_id}/asset-models/{model_id}` | 删除资产模型 |
| `DeleteAssetNew` | DELETE | `/v1/{project_id}/assets/{asset_id}` | 删除资产 |
| `DeleteBatchJob` | DELETE | `/v1/{project_id}/batch/jobs/{job_id}` | 删除离线作业 |
| `DeleteComputingResource` | DELETE | `/v1/{project_id}/batch-computing-resources/{computing_resource_id}` | 删除批计算资源 |
| `DeleteDatasource` | DELETE | `/v1/{project_id}/datasources/{datasource_id}` | 删除数据源 |
| `DeleteDataStore` | DELETE | `/v1/{project_id}/data-stores/{data_store_id}` | 删除存储 |
| `DeleteGroup` | DELETE | `/v1/{project_id}/data-store-groups/{group_id}` | 删除存储组 |
| `DeletePipelineJob` | DELETE | `/v1/{project_id}/pipelines/{pipeline_id}` | 删除管道作业 |
| `DeleteRun` | DELETE | `/v1/{project_id}/batch/jobs/{job_id}/runs/{run_id}` | 停止离线作业 |
| `DeleteStreamingJobById` | DELETE | `/v1/{project_id}/streaming/jobs/{job_id}` | 删除实时作业 |
| `DeleteTable` | DELETE | `/v1/{project_id}/tables/{table_id}` | 删除离线数据表 |
| `ExportDataset` | POST | `/v1/{project_id}/batch/jobs/{job_id}/runs/{run_id}/export-dataset` | 下载离线作业结果 |
| `ImportData` | POST | `/v1/{project_id}/batch/jobs/import/runs` | 执行导入数据离线作业 |
| `ListAssetModels` | GET | `/v1/{project_id}/asset-models` | 获取资产模型列表 |
| `ListAssetsNew` | GET | `/v1/{project_id}/assets` | 获取资产列表 |
| `ListBatchJobs` | GET | `/v1/{project_id}/batch/jobs` | 查询离线作业列表 |
| `ListComputingResources` | GET | `/v1/{project_id}/batch-computing-resources` | 查询批计算资源列表 |
| `ListDataStores` | GET | `/v1/{project_id}/data-stores` | 查询存储列表 |
| `ListGroups` | GET | `/v1/{project_id}/data-store-groups` | 查询存储组列表 |

... and 36 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
