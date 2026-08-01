---
name: huaweicloud-devstar
description: HuaweiCloud DevStar API guide. 26 APIs covering 代码生成, 应用管理, 模板管理.
---

# HuaweiCloud DevStar API Guide

26 APIs. Tags: 代码生成, 应用管理, 模板管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CheckRepositoryDuplicateName` | POST | `/v1/check-repository-duplicate-name` | 检查仓库名称是否重名 |
| `ConfirmDeploymentJob` | POST | `/v1/applications/{application_id}/environments/{environment_tag}/confirm` | 部署任务执行变更人工审核 |
| `CreateDeploymentJobs` | POST | `/v1/applications/{application_id}/environments/{environment_tag}/deployment-jobs` | 创建部署任务 |
| `CreateTemplateViewHistories` | POST | `/v1/templates/view-histories` | 同步模板浏览记录 |
| `DeleteApplicationV4` | DELETE | `/v4/applications/{application_id}` | 删除应用信息 |
| `DownloadApplicationCode` | GET | `/v1/application-codes` | 下载模板产物 |
| `ListApplicationsV6` | GET | `/v6/applications` | 获取应用列表 |
| `ListPipelineTemplates` | GET | `/v1/pipeline-templates` | 流水线模板列表查询 |
| `ListProjectsV4` | GET | `/v4/projects` | 获取用户有权限的DevStar存量DevCloud项目列表 |
| `ListTemplates` | POST | `/v1/templates/query` | 查询模板列表 |
| `ListTemplateViewHistories` | GET | `/v1/templates/view-histories` | 我浏览的模板记录 |
| `RunCodehubTemplateJob` | POST | `/v1/jobs/codehub` | CodeHub 模板生成代码 |
| `RunDevstarTemplateJob` | POST | `/v1/jobs/template` | Devstar 模板生成代码 |
| `ShowApplicationDependentResources` | GET | `/v3/applications/{application_id}/dependent-resources` | 获取应用依赖元数据资源 |
| `ShowApplicationReleaseRepositories` | GET | `/v1/applications/{application_id}/release-repositories` | 通过应用Id获取软件发布仓库列表  |
| `ShowApplicationResDeleteStatus` | GET | `/v1/application-resources/{application_id}/delete-status` | 查询应用关联资源删除状态 |
| `ShowApplicationV3` | GET | `/v3/applications/{application_id}` | 获取应用详情 |
| `ShowDeploymentJobs` | GET | `/v1/applications/{application_id}/environments/{environment_tag}/deployment-jobs/detail` | 查询应用环境部署任务详情 |
| `ShowJobDetail` | GET | `/v1/jobs/{job_id}` | 查询任务详情 |
| `ShowPipelineLastStatusV2` | GET | `/v2/pipelines/{pipeline_id}/status` | 查询流水线最近一次运行状态查询接口 |
| `ShowRepositoryByCloudIde` | GET | `/v1/repositories/{repository_id}/show/cloudide` | 使用 CloudIDE 实例打开应用代码 |
| `ShowRepositoryStatisticalDataV2` | GET | `/v2/repositories/{repository_id}/statistical-data` | 应用代码仓库统计信息 |
| `ShowTemplateFile` | GET | `/v1/templates/{template_id}/files` | 读取模板文件 |
| `ShowTemplateV3` | GET | `/v3/templates/{template_id}` | 查询模板详情(V3) |
| `StartPipeline` | POST | `/v2/pipelines/{pipeline_id}/start` | 根据流水线Id操作流水线启动 |
| `UpdateApplication` | PUT | `/v3/applications/{application_id}` | 更新应用信息 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
