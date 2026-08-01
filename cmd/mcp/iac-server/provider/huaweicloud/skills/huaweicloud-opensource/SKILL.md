---
name: huaweicloud-opensource
description: HuaweiCloud OpenSource API guide. 9 APIs covering Hook管理, 仓库管理, 项目管理.
---

# HuaweiCloud OpenSource API Guide

9 APIs. Tags: Hook管理, 仓库管理, 项目管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateWebHook` | POST | `/v1/projects/{namespace}/{project}/hooks` | 创建webhook |
| `DeleteWebHook` | DELETE | `/v1/projects/{namespace}/{project}/hooks/{hook_id}` | 删除webhook |
| `ListProjectDetail` | GET | `/v1/projects/{namespace}/{project}` | 查询项目详情 |
| `ListProjects` | GET | `/v1/projects` | 获取项目列表 |
| `ListRepositoryBranches` | GET | `/v1/projects/{namespace}/{project}/repository/branches` | 查询仓库分支列表 |
| `ListRepositoryCommits` | GET | `/v1/projects/{namespace}/{project}/repository/commits` | 查询仓库提交历史列表 |
| `ListRepositoryTags` | GET | `/v1/projects/{namespace}/{project}/repository/tags` | 查询仓库标签列表 |
| `ListWebHookDetail` | GET | `/v1/projects/{namespace}/{project}/hooks/{hook_id}` | 获得webhook |
| `UpdateWebHook` | PUT | `/v1/projects/{namespace}/{project}/hooks/{hook_id}` | 修改webhook |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
