---
name: huaweicloud-mpc
description: HuaweiCloud MPC API guide. 41 APIs covering 媒资转码接口, 抽帧截图接口, 授权与配置接口, 水印模板接口, 租户开通.
---

# HuaweiCloud MPC API Guide

41 APIs. Tags: 媒资转码接口, 抽帧截图接口, 授权与配置接口, 水印模板接口, 租户开通, 自定义转码模板接口, 自定义转码模板组接口, 视频解析接口, 转动图接口, 转封装接口

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CancelRemuxTask` | DELETE | `/v1/{project_id}/remux` | 取消转封装任务 |
| `CreateAgenciesTask` | POST | `/v1/{project_id}/agencies` | 请求委托任务 |
| `CreateAnimatedGraphicsTask` | POST | `/v1/{project_id}/animated-graphics` | 新建转动图任务 |
| `CreateExtractTask` | POST | `/v1/{project_id}/extract-metadata` | 新建视频解析任务 |
| `CreateRemuxTask` | POST | `/v1/{project_id}/remux` | 新建转封装任务 |
| `CreateRetryRemuxTask` | PUT | `/v1/{project_id}/remux` | 重试转封装任务 |
| `CreateTemplateGroup` | POST | `/v1/{project_id}/template_group/transcodings` | 新建转码模板组 |
| `CreateThumbnailsTask` | POST | `/v1/{project_id}/thumbnails` | 新建截图任务 |
| `CreateTranscodingTask` | POST | `/v1/{project_id}/transcodings` | 新建转码任务 |
| `CreateTransTemplate` | POST | `/v1/{project_id}/template/transcodings` | 新建转码模板 |
| `CreateWatermarkTemplate` | POST | `/v1/{project_id}/template/watermark` | 新建水印模板 |
| `DeleteAnimatedGraphicsTask` | DELETE | `/v1/{project_id}/animated-graphics` | 取消转动图任务 |
| `DeleteExtractTask` | DELETE | `/v1/{project_id}/extract-metadata` | 取消视频解析任务 |
| `DeleteRemuxTask` | DELETE | `/v1/{project_id}/remux/task` | 删除转封装任务记录 |
| `DeleteTemplate` | DELETE | `/v1/{project_id}/template/transcodings` | 删除转码模板 |
| `DeleteTemplateGroup` | DELETE | `/v1/{project_id}/template_group/transcodings` | 删除转码模板组 |
| `DeleteThumbnailsTask` | DELETE | `/v1/{project_id}/thumbnails` | 取消截图任务 |
| `DeleteTranscodingTask` | DELETE | `/v1/{project_id}/transcodings` | 取消转码任务 |
| `DeleteTranscodingTaskByConsole` | DELETE | `/v1/{project_id}/transcodings/task` | 删除转码任务记录 |
| `DeleteWatermarkTemplate` | DELETE | `/v1/{project_id}/template/watermark` | 删除水印模板 |
| `ListAllBuckets` | GET | `/v1/{project_id}/buckets` | 查询桶列表 |
| `ListAllObsObjList` | GET | `/v1.0-ext/{project_id}/objects` | 查询桶里的object |
| `ListAnimatedGraphicsTask` | GET | `/v1/{project_id}/animated-graphics` | 查询转动图任务 |
| `ListExtractTask` | GET | `/v1/{project_id}/extract-metadata` | 查询视频解析任务 |
| `ListNotifyEvent` | GET | `/v1/{project_id}/notification/event` | 查询转码服务端所有事件 |
| `ListNotifySmnTopicConfig` | GET | `/v1/{project_id}/notification` | 查询转码服务端事件通知 |
| `ListRemuxTask` | GET | `/v1/{project_id}/remux` | 查询转封装任务 |
| `ListStatSummary` | GET | `/v1/{project_id}/transcodings/summaries` | 查询点播概览信息 |
| `ListTemplate` | GET | `/v1/{project_id}/template/transcodings` | 查询转码模板 |
| `ListTemplateGroup` | GET | `/v1/{project_id}/template_group/transcodings` | 查询转码模板组 |

... and 11 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
