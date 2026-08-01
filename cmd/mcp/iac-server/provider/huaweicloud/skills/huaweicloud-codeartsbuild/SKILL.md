---
name: huaweicloud-codeartsbuild
description: HuaweiCloud CodeArtsBuild API guide. 135 APIs covering GroupManager, LogManager, 文件管理, 构建任务管理, 构建报告.
---

# HuaweiCloud CodeArtsBuild API Guide

135 APIs. Tags: GroupManager, LogManager, 文件管理, 构建任务管理, 构建报告, 构建记录, 模板管理, 租户相关, 编译构建(待下线), 编译构建(旧), 镜像模板

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddFavouriteCustomTemplate` | POST | `/v1/template/custom/{uuid}/follow` | 收藏自定义模板 |
| `AddFavouriteOfficialTemplate` | POST | `/v1/template/official/{uuid}/follow` | 收藏官方模板 |
| `AddFavouriteTask` | POST | `/v1/job/{job_id}/follow` | 收藏任务 |
| `AddKeystorePermission` | POST | `/v2/keystore/permission/add` | 添加文件权限 |
| `ApplyProjectPermission` | POST | `/v1/job/project/permission` | 任务是否使用项目级权限 |
| `BatchDeleteBuildJobs` | DELETE | `/v1/job/batch-delete` | 批量删除构建任务 |
| `BatchUpdateJobRolePermission` | POST | `/v1/job/permissions/batch` | 批量修改任务权限 |
| `CheckJobCountIsTopLimit` | GET | `/v1/job/check/count` | 检查任务数量是否上限 |
| `CheckJobInternal` | GET | `/v1/job/permission/internal` | 是否已开启内网安全访问 |
| `CheckJobNameIsExists` | GET | `/v1/job/check/exist` | 查看项目下任务名是否存在 |
| `CheckWebhookUrl` | POST | `/v1/job/check/webhook-url` | 检查webhook地址参数 |
| `ClearRecyclingJobs` | DELETE | `/v1/job/recycling-empty` | 清空回收站中的任务 |
| `CopyJob` | POST | `/v1/job/copy` | 复制构建任务 |
| `CreateBuildJob` | POST | `/v3/jobs/create` | 创建构建任务 |
| `CreateJobGroup` | POST | `/v1/job/{project_id}/group/create` | 创建构建任务分组 |
| `CreateNewJob` | POST | `/v1/job/create` | 创建构建任务 |
| `CreateTemplate` | POST | `/v1/template/create` | 创建构建模板 |
| `CreateTemplates` | POST | `/v3/templates/create` | 创建构建模板 |
| `DeleteBuildJob` | POST | `/v3/jobs/{job_id}/delete` | 删除构建任务 |
| `DeleteGroup` | DELETE | `/v1/job/{project_id}/group/delete` | 删除分组 |
| `DeleteKeystore` | DELETE | `/v2/keystore/{keystore_id}/delete` | 删除文件管理文件 |
| `DeleteKeystorePermission` | DELETE | `/v2/keystore/permission/{permission_id}/delete` | 文件管理删除权限 |
| `DeleteRecyclingJobs` | DELETE | `/v1/job/recycling-deletion` | 删除回收站中的任务 |
| `DeleteTemplate` | DELETE | `/v1/template/{uuid}/delete` | 删除构建模板 |
| `DeleteTemplates` | DELETE | `/v3/templates/{uuid}/delete` | 删除构建模板 |
| `DeleteTheJob` | DELETE | `/v1/job/{job_id}/delete` | 删除任务 |
| `DisableBuildJob` | POST | `/v3/jobs/{job_id}/disable` | 禁用构建任务 |
| `DisableNotice` | POST | `/v3/jobs/notice/{job_id}/disable` | 取消通知 |
| `DisableTheJob` | POST | `/v1/job/{job_id}/disable` | 禁用任务 |
| `DownloadBuildFullLog` | GET | `/v1/log/{record_id}/download-log` | 下载全量构建日志 |

... and 105 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
