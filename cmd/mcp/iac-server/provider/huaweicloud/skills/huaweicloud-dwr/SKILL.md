---
name: huaweicloud-dwr
description: HuaweiCloud DWR API guide. 46 APIs covering Workflow, 三方算子, 任务查询接口, 向量entity接口, 工作流权限.
---

# HuaweiCloud DWR API Guide

46 APIs. Tags: Workflow, 三方算子, 任务查询接口, 向量entity接口, 工作流权限, 执行工作流, 知识仓接口, 系统算子, 索引接口, 集合接口

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptServiceContract` | POST | `/v3/{project_id}/workflow-agreements` | 同意服务协议 |
| `AsyncInvokeApiStartWorkflow` | POST | `/v3/{project_id}/workflows/{graph_name}/execute` | API异步启动工作流 |
| `CheckWorkflowAuthentication` | GET | `/v3/{project_id}/workflow-authorization` | 查询授权 |
| `CreateCollection` | POST | `/v1/collections/create` | 创建collection |
| `CreateIndex` | POST | `/v1/indexes/create` | 创建索引 |
| `CreateMyActionTemplate` | POST | `/v3/{project_id}/myactiontemplates/{template_name}` | 创建第三方算子模板 |
| `CreateStore` | POST | `/v1/stores/create` | 创建知识仓实例 |
| `CreateWorkflow` | POST | `/v3/{project_id}/workflows/{graph_name}` | 创建工作流 |
| `CreateWorkflowAuthentication` | POST | `/v3/{project_id}/workflow-authorization` | 开通授权 |
| `DeleteCollection` | POST | `/v1/collections/delete` | 删除collection |
| `DeleteEntities` | POST | `/v1/entities/delete` | 删除向量 |
| `DeleteIndex` | POST | `/v1/indexes/delete` | 删除索引 |
| `DeleteMyActionTemplate` | DELETE | `/v3/{project_id}/myactiontemplates/{template_name}` | 删除第三方算子模板 |
| `DeleteStore` | POST | `/v1/stores/delete` | 删除知识仓实例 |
| `DeleteWorkflow` | DELETE | `/v3/{project_id}/workflows/{graph_name}` | 删除工作流 |
| `DescribeCollection` | POST | `/v1/collections/describe` | 查询collection |
| `DescribeIndex` | POST | `/v1/indexes/describe` | 查询索引 |
| `DescribeJob` | POST | `/v1/jobs/describe` | 获取指定ID任务信息 |
| `DescribeStore` | POST | `/v1/stores/describe` | 查询知识仓实例 |
| `GetProgress` | POST | `/v1/indexes/get-progress` | 查询索引构建进度 |
| `HybridSearch` | POST | `/v1/entities/hybrid-search` | 混合搜索 |
| `InsertEntities` | POST | `/v1/entities/insert` | 插入向量 |
| `ListCollections` | POST | `/v1/collections/list` | 列举collection |
| `ListJobs` | POST | `/v1/jobs/list` | 查询任务列表 |
| `ListMyActionTemplate` | GET | `/v3/{project_id}/myactiontemplates` | 查询第三方算子列表 |
| `ListStores` | POST | `/v1/stores/list` | 列举知识仓实例 |
| `ListSystemTemplates` | GET | `/v3/{project_id}/actiontemplates` | 查询华为云内置算子列表 |
| `ListWorkflowInstance` | GET | `/v3/{project_id}/workflowexecutions` | 本接口用于查询用户工作流的实例列表 |
| `ListWorkflows` | GET | `/v3/{project_id}/workflows` | 查询工作流列表 |
| `LoadCollection` | POST | `/v1/collections/load` | 加载collection |

... and 16 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
