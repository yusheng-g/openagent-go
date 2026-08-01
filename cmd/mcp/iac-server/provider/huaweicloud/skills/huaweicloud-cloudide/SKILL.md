---
name: huaweicloud-cloudide
description: HuaweiCloud CloudIDE API guide. 44 APIs covering IDE实例管理, codebreeze, codebreezetsbot, 帐号权限管理, 技术栈管理.
---

# HuaweiCloud CloudIDE API Guide

44 APIs. Tags: IDE实例管理, codebreeze, codebreezetsbot, 帐号权限管理, 技术栈管理, 插件市场, 插件管理, 模板管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddExtensionEvaluation` | POST | `/v1/marketplace/extension/evaluation` | 添加插件评论 |
| `AddExtensionEvaluationReply` | POST | `/v1/marketplace/extension/evaluation/reply` | 添加评论回复、回复评论回复 |
| `AddExtensionStar` | POST | `/v1/marketplace/star` | 添加新评星 |
| `CheckInstanceAccess` | GET | `/v2/instances/{instance_id}/access` | 查询用户是否有权限访问某个IDE实例 |
| `CheckMaliciousExtensionEvaluation` | POST | `/v1/marketplace/extension/evaluation/accusation` | 举报评论,举报回复 |
| `CheckName` | GET | `/v2/instances/duplicate` | 查询IDE实例名是否重复 |
| `CreateAcceptance` | POST | `/v2/aims/codemodelserver/code-generation/acceptance` | CreateAcceptance接口 |
| `CreateApply` | POST | `/v2/aims/codemodelserver/join-request` | CreateJoinRequest接口 |
| `CreateEvent` | POST | `/v2/aims/codemodelserver/management/event` | CreateEvent接口 |
| `CreateExtensionAuthorization` | POST | `/v2/extension/authorization/{instance_id}` | 设置ide实例对插件的授权 |
| `CreateInstance` | POST | `/v2/{org_id}/instances` | 创建IDE实例 |
| `CreateInstanceBy3rd` | POST | `/v2/instances` | 外部第三方集成商创建IDE实例 |
| `CreateLogin` | POST | `/v2/aims/codemodelserver/code-generation/login` | CreateLogin接口 |
| `CreateRequest` | POST | `/v2/aims/codemodelserver/code-generation/request` | Create Request接口 |
| `DeleteEvaluation` | DELETE | `/v1/marketplace/evaluation/{evaluation_id}` | 删除评论 |
| `DeleteEvaluationReply` | DELETE | `/v1/marketplace/evaluation/reply/{reply_id}` | 删除回复 |
| `DeleteInstance` | DELETE | `/v2/instances/{instance_id}` | 删除IDE实例 |
| `ListExtensions` | POST | `/v1/marketplace/extension/extensionquery` | 查询插件列表 |
| `ListInstances` | GET | `/v2/instances` | 查询IDE实例列表 |
| `ListOrgInstances` | GET | `/v2/{org_id}/instances` | 查询某个租户下的IDE实例列表 |
| `ListProjectTemplates` | GET | `/v2/templates` | 查询技术栈模板工程 |
| `ListPublisher` | GET | `/v1/marketplace/publishers/mine` | 获取当前用户下的发布商列表 |
| `ListStacks` | GET | `/v2/stacks/tag` | 按region获取标签所有技术栈 |
| `PublishExtension` | POST | `/v1/marketplace/extension/{task_id}/archiving` | 插件发布 |
| `ShowAccountStatus` | GET | `/v2/permission/account/status` | 查询当前帐号访问权限 |
| `ShowCategoryList` | GET | `/v1/marketplace/extension/category` | 查询插件分类 |
| `ShowExtensionAuthorization` | GET | `/v2/extension/authorization` | 查询ide实例对插件的授权情况 |
| `ShowExtensionDetail` | POST | `/v1/marketplace/extension/public/detail` | 查询插件详细信息 |
| `ShowExtensionEvaluation` | GET | `/v1/marketplace/feedback/evaluation` | 查询插件评价 |
| `ShowExtensionEvaluationStar` | GET | `/v1/marketplace/feedback/star` | 查询插件评星 |

... and 14 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
