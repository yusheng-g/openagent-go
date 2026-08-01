---
name: huaweicloud-projectman
description: HuaweiCloud ProjectMan API guide. 94 APIs covering IPD工作流, IPD工作项, IPD工作项状态, IPD项目接口, OpenAPI管理.
---

# HuaweiCloud ProjectMan API Guide

94 APIs. Tags: IPD工作流, IPD工作项, IPD工作项状态, IPD项目接口, OpenAPI管理, Plan, Scrum项目的工作项, Scrum项目的模块, Scrum项目的状态, Scrum项目的迭代, Scrum项目的领域, Severity, 用户信息, 看板项目的工作项, 项目信息, 项目成员, 项目指标, 项目统计

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddApplyJoinProjectForAgc` | GET | `/v4/projects/{project_id}/members/agc-join` | AGC调用 当前用户申请加入项目 |
| `AddIssueWorkHours` | POST | `/v4/projects/{project_id}/issues/{issue_id}/work-hours` | 添加指定工作项工时 |
| `AddMemberV4` | POST | `/v4/projects/{project_id}/member` | 添加项目成员 |
| `BatchAddMembersV4` | POST | `/v4/projects/{project_id}/members` | 批量添加项目成员 |
| `BatchDeleteIssuesV4` | DELETE | `/v4/projects/{project_id}/issues` | 批量删除工作项 |
| `BatchDeleteIterationsV4` | DELETE | `/v4/projects/{project_id}/iterations` | 批量删除项目的迭代 |
| `BatchDeleteMembersV4` | DELETE | `/v4/projects/{project_id}/members` | 批量删除项目成员 |
| `BatchListAssociatedIssues` | GET | `/v4/projects/{project_id}/issues/batch-associated-issues` | 查询当前项目下已经关联的工作项 |
| `BatchUpdateChildNickNames` | PUT | `/v4/domain/child-users` | 更新子用户昵称 |
| `CancelProjectDomain` | DELETE | `/v4/projects/{project_id}/domains/{domain_id}` | 取消领域与项目的关联关系 |
| `CheckProjectNameV4` | POST | `/v4/projects/check-name` | 检查项目名称是否存在 |
| `CreateCustomfields` | POST | `/v3/{project_id}/custom-fields` | 创建工作项类型自定义字段 |
| `CreateIpdProjectIssue` | POST | `/v1/ipdprojectservice/projects/{project_id}/issues` | 创建工作项 |
| `CreateIpdProjectIssueAttachment` | POST | `/v1/ipdprojectservice/projects/{project_id}/issues/{issue_id}/attachments/upload` | 上传issue附件 |
| `CreateIssueV4` | POST | `/v4/projects/{project_id}/issue` | 创建工作项 |
| `CreateIterationV4` | POST | `/v4/projects/{project_id}/iteration` | 创建Scrum项目迭代 |
| `CreateProjectDomain` | POST | `/v4/projects/{project_id}/domain` | 创建项目的领域 |
| `CreateProjectModule` | POST | `/v4/projects/{project_id}/module` | 创建项目的模块 |
| `CreateProjectV4` | POST | `/v4/project` | 创建项目 |
| `CreateScrumPlanToProject` | POST | `/v3/plan/{project_id}/management` | 新增需求规划 |
| `CreateSystemIssueV4` | POST | `/v4/projects/{project_id}/system/issue` | 细粒度权限用户创建工作项 |
| `DeleteAttachment` | DELETE | `/v4/projects/{project_id}/issues/{issue_id}/attachments/{attachment_id}` | 删除附件 |
| `DeleteIssueV4` | DELETE | `/v4/projects/{project_id}/issues/{issue_id}` | 删除工作项 |
| `DeleteIterationV4` | DELETE | `/v4/projects/{project_id}/iterations/{iteration_id}` | 删除项目迭代 |
| `DeleteProjectModule` | DELETE | `/v4/projects/{project_id}/modules/{module_id}` | 删除项目的模块 |
| `DeleteProjectV4` | DELETE | `/v4/projects/{project_id}` | 删除项目 |
| `DeleteScrumPlanInProject` | DELETE | `/v3/plan/{project_id}/management` | 删除规划(支持批量) |
| `DownloadAttachment` | GET | `/v4/projects/{project_id}/issues/{issue_id}/attachments/{attachment_id}` | 下载工作项附件 |
| `DownloadImageFile` | GET | `/v4/projects/{project_id}/image-file` | 下载图片 |
| `DownloadIpdIssueAttachment` | GET | `/v1/ipdprojectservice/projects/{project_id}/attachments/download/{id}` | 根据ID下载工作项附件 |

... and 64 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
