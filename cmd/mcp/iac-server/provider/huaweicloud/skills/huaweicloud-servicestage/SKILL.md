---
name: huaweicloud-servicestage
description: HuaweiCloud ServiceStage API guide. 71 APIs covering Application, Component, Environment, GIT仓库, GIT仓库文件.
---

# HuaweiCloud ServiceStage API Guide

71 APIs. Tags: Application, Component, Environment, GIT仓库, GIT仓库文件, GIT授权, Job, Meta, RuntimeStack, 实例, 应用, 环境, 组件, 部署任务

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ChangeApplication` | PUT | `/v2/{project_id}/cas/applications/{application_id}` | 修改应用信息 |
| `ChangeApplicationConfiguration` | PUT | `/v2/{project_id}/cas/applications/{application_id}/configuration` | 修改应用配置信息 |
| `ChangeComponent` | PUT | `/v2/{project_id}/cas/applications/{application_id}/components/{component_id}` | 根据组件ID修改组件信息 |
| `ChangeEnvironment` | PUT | `/v2/{project_id}/cas/environments/{environment_id}` | 修改环境信息 |
| `ChangeInstance` | PUT | `/v2/{project_id}/cas/applications/{application_id}/components/{component_id}/instances/{instance_id}` | 修改应用组件实例 |
| `ChangeResourceInEnvironment` | PATCH | `/v2/{project_id}/cas/environments/{environment_id}/resources` | 修改环境资源 |
| `CreateApplication` | POST | `/v3/{project_id}/cas/applications` | 创建应用 |
| `CreateComponent` | POST | `/v3/{project_id}/cas/applications/{application_id}/components` | 应用中创建组件 |
| `CreateEnvironment` | POST | `/v2/{project_id}/cas/environments` | 创建环境 |
| `CreateFile` | POST | `/v1/{project_id}/git/files/{namespace}/{project}/{path}` | 创建仓库文件 |
| `CreateHook` | POST | `/v1/{project_id}/git/repos/{namespace}/{project}/hooks` | 创建项目hook |
| `CreateInstance` | POST | `/v2/{project_id}/cas/applications/{application_id}/components/{component_id}/instances` | 创建组件实例 |
| `CreateOAuth` | POST | `/v1/{project_id}/git/auths/{repo_type}/oauth` | 创建OAuth授权 |
| `CreatePasswordAuth` | POST | `/v1/{project_id}/git/auths/{repo_type}/password` | 创建口令授权 |
| `CreatePersonalAuth` | POST | `/v1/{project_id}/git/auths/{repo_type}/personal` | 创建私人令牌授权 |
| `CreateProject` | POST | `/v1/{project_id}/git/repos/{namespace}/projects` | 创建软件仓库项目 |
| `CreateTag` | POST | `/v1/{project_id}/git/repos/{namespace}/{project}/tags` | 创建项目tag标签 |
| `DeleteApplication` | DELETE | `/v2/{project_id}/cas/applications/{application_id}` | 根据应用ID删除应用 |
| `DeleteApplicationConfiguration` | DELETE | `/v3/{project_id}/cas/applications/{application_id}/configuration` | 根据应用ID删除应用配置 |
| `DeleteAuthorize` | DELETE | `/v1/{project_id}/git/auths/{name}` | 删除仓库授权 |
| `DeleteComponent` | DELETE | `/v2/{project_id}/cas/applications/{application_id}/components/{component_id}` | 根据应用组件ID删除应用组件 |
| `DeleteEnvironment` | DELETE | `/v3/{project_id}/cas/environments/{environment_id}` | 根据环境ID删除环境 |
| `DeleteFile` | DELETE | `/v1/{project_id}/git/files/{namespace}/{project}/{path}` | 删除仓库文件 |
| `DeleteHook` | DELETE | `/v1/{project_id}/git/repos/{namespace}/{project}/hooks/{hook_id}` | 删除项目hook |
| `DeleteInstance` | DELETE | `/v2/{project_id}/cas/applications/{application_id}/components/{component_id}/instances/{instance_id}` | 删除应用组件实例 |
| `DeleteTag` | DELETE | `/v1/{project_id}/git/repos/{namespace}/{project}/tags/{tag_name}` | 删除项目tag标签 |
| `ListApplications` | GET | `/v2/{project_id}/cas/applications` | 获取所有应用 |
| `ListAuthorizations` | GET | `/v1/{project_id}/git/auths` | 获取仓库授权列表 |
| `ListBranches` | GET | `/v1/{project_id}/git/repos/{namespace}/{project}/branches` | 获取项目分支 |
| `ListCommits` | GET | `/v1/{project_id}/git/repos/{namespace}/{project}/commits` | 获取项目commit提交记录 |

... and 41 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
