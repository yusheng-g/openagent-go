---
name: huaweicloud-res
description: HuaweiCloud RES API guide. 33 APIs covering 在线服务, 场景, 工作空间, 数据源, 查询规格.
---

# HuaweiCloud RES API Guide

33 APIs. Tags: 在线服务, 场景, 工作空间, 数据源, 查询规格, 训练作业, 调度

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateResDatasource` | POST | `/v2.0/{project_id}/workspaces/{workspace_id}/data-sources` | 创建数据源 |
| `CreateResIntelligentScene` | POST | `/v2.0/{project_id}/workspaces/{workspace_id}/intelligent-scenes` | 创建智能场景 |
| `CreateResJob` | POST | `/v2.0/{project_id}/workspaces/{workspace_id}/resources/{resource_id}/job-instance` | 新建训练作业 |
| `CreateResJobs` | POST | `/v2.0/{project_id}/workspaces/{workspace_id}/resources/{resource_id}/job-instances` | 新建多个训练作业 |
| `CreateResOnlineInstance` | POST | `/v2.0/{project_id}/workspaces/{workspace_id}/resources/{resource_id}/service-instance` | 新建在线服务 |
| `CreateResScene` | POST | `/v2.0/{project_id}/workspaces/{workspace_id}/scenes` | 创建自定义场景 |
| `CreateResWorkspace` | POST | `/v2.0/{project_id}/workspaces` | 创建工作空间 |
| `DeleteResDatasource` | DELETE | `/v2.0/{project_id}/workspaces/{workspace_id}/data-sources/{datasource_id}` | 删除数据源 |
| `DeleteResJob` | DELETE | `/v2.0/{project_id}/workspaces/{workspace_id}/resources/{resource_id}/job-instance/{job_id}` | 删除训练作业 |
| `DeleteResOnlineInstance` | DELETE | `/v2.0/{project_id}/workspaces/{workspace_id}/resources/{resource_id}/service-instance/{job_id}` | 删除在线服务 |
| `DeleteResScene` | DELETE | `/v2.0/{project_id}/workspaces/{workspace_id}/scenes/{scene_id}` | 删除场景 |
| `DeleteResWorkspace` | DELETE | `/v2.0/{project_id}/workspaces/{workspace_id}` | 删除工作空间 |
| `ListResDatasources` | GET | `/v2.0/{project_id}/workspaces/{workspace_id}/data-sources` | 查询数据源列表 |
| `ListResEnterprises` | GET | `/v2.0/{project_id}/enterprise-projects` | 查询企业项目列表 |
| `ListResOnlineServiceDetails` | GET | `/v2.0/{project_id}/workspaces/{workspace_id}/resources/{resource_id}/service-instance` | 查询在线服务详情 |
| `ListResResourceSpec` | GET | `/v2.0/{project_id}/resource-specs` | 查询训练规格 |
| `ListResScenes` | GET | `/v2.0/{project_id}/workspaces/{workspace_id}/scenes` | 查询场景列表 |
| `ListResWorkspaces` | GET | `/v2.0/{project_id}/workspaces` | 查询工作空间列表 |
| `ShowResDatasource` | GET | `/v2.0/{project_id}/workspaces/{workspace_id}/data-sources/{datasource_id}` | 查询数据源详情 |
| `ShowResDatasourceWorkDetail` | GET | `/v2.0/{project_id}/workspaces/{workspace_id}/data-sources/{resource_id}/detail` | 查询数据源任务结果 |
| `ShowResJob` | GET | `/v2.0/{project_id}/workspaces/{workspace_id}/resources/{resource_id}/job-instance` | 查询训练作业 |
| `ShowResRecallSet` | GET | `/v2.0/{project_id}/workspaces/{workspace_id}/resources/{resource_id}/result-set` | 查询训练作业候选集 |
| `ShowResScene` | GET | `/v2.0/{project_id}/workspaces/{workspace_id}/scenes/{scene_id}` | 查询场景详情 |
| `ShowResWrokspace` | GET | `/v2.0/{project_id}/workspaces/{workspace_id}` | 查询工作空间详情 |
| `StartResJob` | POST | `/v2.0/{project_id}/workspaces/{workspace_id}/resources/{resource_id}/jobs/{job_id}/schedule-job` | 执行作业 |
| `StartResSceneJobs` | POST | `/v2.0/{project_id}/workspaces/{workspace_id}/resources/{resource_id}/schedule-scene` | 执行场景 |
| `UpdateResDatasource` | PUT | `/v2.0/{project_id}/workspaces/{workspace_id}/data-sources/{datasource_id}` | 修改数据源内容 |
| `UpdateResDatastruct` | PUT | `/v2.0/{project_id}/workspaces/{workspace_id}/data-sources/{datasource_id}/data-struct` | 修改数据源特征 |
| `UpdateResIntelligentScene` | PUT | `/v2.0/{project_id}/workspaces/{workspace_id}/intelligent-scenes/{scene_id}` | 更新智能场景内容 |
| `UpdateResJob` | PUT | `/v2.0/{project_id}/workspaces/{workspace_id}/resources/{resource_id}/job-instance/{job_id}` | 修改训练作业参数 |

... and 3 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
