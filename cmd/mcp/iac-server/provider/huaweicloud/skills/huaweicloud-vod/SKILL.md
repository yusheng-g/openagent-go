---
name: huaweicloud-vod
description: HuaweiCloud VOD API guide. 73 APIs covering Https配置, OBS托管管理, 媒资上传, 媒资分类, 媒资刷新.
---

# HuaweiCloud VOD API Guide

73 APIs. Tags: Https配置, OBS托管管理, 媒资上传, 媒资分类, 媒资刷新, 媒资处理, 媒资存储模式管理, 媒资管理, 媒资编辑管理, 媒资预热, 字幕管理, 密钥查询, 截图管理, 水印模板管理, 统计分析, 转码产物管理, 转码模板管理, 转码模板组管理, 转码模板集合管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CancelAssetTranscodeTask` | DELETE | `/v1.0/{project_id}/asset/process` | 取消媒资转码任务 |
| `CancelExtractAudioTask` | DELETE | `/v1.0/{project_id}/asset/extract_audio` | 取消提取音频任务 |
| `CheckMd5Duplication` | GET | `/v1.0/{project_id}/asset/duplication` | 上传检验 |
| `ConfirmAssetUpload` | POST | `/v1.0/{project_id}/asset/status/uploaded` | 确认媒资上传 |
| `ConfirmImageUpload` | POST | `/v1.0/{project_id}/watermark/status/uploaded` | 确认水印图片上传 |
| `CreateAssetByFileUpload` | POST | `/v1.0/{project_id}/asset` | 创建媒资:上传方式 |
| `CreateAssetCategory` | POST | `/v1.0/{project_id}/asset/category` | 创建媒资分类 |
| `CreateAssetEditTask` | POST | `/v1/{project_id}/asset/editing/tasks` | 创建编辑任务 |
| `CreateAssetProcessTask` | POST | `/v1.0/{project_id}/asset/process` | 媒资处理 |
| `CreateAssetReviewTask` | POST | `/v1.0/{project_id}/asset/review` | 创建审核媒资任务 |
| `CreateExtractAudioTask` | POST | `/v1.0/{project_id}/asset/extract_audio` | 音频提取 |
| `CreatePreheatingAsset` | POST | `/v1.0/{project_id}/asset/preheating` | CDN预热 |
| `CreateTakeOverTask` | POST | `/v1.0/{project_id}/asset/obs/host/stock/task` | 创建媒资:OBS托管方式 |
| `CreateTemplateGroup` | POST | `/v1.0/{project_id}/asset/template_group/transcodings` | 创建自定义转码模板组 |
| `CreateTemplateGroupCollection` | POST | `/v1.0/{project_id}/asset/template-collection/transcodings` | 创建转码模板组集合 |
| `CreateTranscodeTemplate` | POST | `/v2/{project_id}/asset/template/transcodings` | 创建自定义转码模板 |
| `CreateWatermarkTemplate` | POST | `/v1.0/{project_id}/template/watermark` | 创建水印模板 |
| `DeleteAssetCategory` | DELETE | `/v1.0/{project_id}/asset/category` | 删除媒资分类 |
| `DeleteAssetEditTask` | DELETE | `/v1/{project_id}/asset/editing/tasks` | 取消编辑任务 |
| `DeleteAssets` | DELETE | `/v1.0/{project_id}/asset` | 删除媒资 |
| `DeleteTemplateGroup` | DELETE | `/v1.0/{project_id}/asset/template_group/transcodings` | 删除自定义转码模板组 |
| `DeleteTemplateGroupCollection` | DELETE | `/v1.0/{project_id}/asset/template-collection/transcodings` | 删除转码模板组集合 |
| `DeleteThumbnails` | DELETE | `/v1/{project_id}/asset/thumbnails` | 删除媒资下的多个截图 |
| `DeleteTranscodeProduct` | DELETE | `/v1/{project_id}/asset/transcode-product` | 删除转码产物 |
| `DeleteTranscodeTemplate` | DELETE | `/v2/{project_id}/asset/template/transcodings` | 删除自定义模板 |
| `DeleteWatermarkTemplate` | DELETE | `/v1.0/{project_id}/template/watermark` | 删除水印模板 |
| `ListAssetCategory` | GET | `/v1.0/{project_id}/asset/category` | 查询指定分类信息 |
| `ListAssetDailySummaryLog` | GET | `/v1/{project_id}/asset/daily-summary` | 查询媒资日播放统计数据 |
| `ListAssetEditTask` | GET | `/v1/{project_id}/asset/editing/tasks` | 查询编辑任务 |
| `ListAssetList` | GET | `/v1.0/{project_id}/asset/list` | 查询媒资列表 |

... and 43 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
