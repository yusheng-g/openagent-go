---
name: huaweicloud-moderation
description: HuaweiCloud Moderation API guide. 9 APIs covering 图像内容审核, 图像审核批量同步接口, 文本内容审核, 视频内容审核, 音频内容审核.
---

# HuaweiCloud Moderation API Guide

9 APIs. Tags: 图像内容审核, 图像审核批量同步接口, 文本内容审核, 视频内容审核, 音频内容审核, 音频流内容审核

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCheckImageSync` | POST | `/v3/{project_id}/moderation/image/batch` | 图像审核批量同步接口 |
| `CheckImageModeration` | POST | `/v3/{project_id}/moderation/image` | 图像内容审核 |
| `RunCloseAudioStreamModerationJob` | None | `None` | 关闭音频流内容审核作业 |
| `RunCreateAudioModerationJob` | POST | `/v3/{project_id}/moderation/audio/jobs` | 创建音频内容审核作业 |
| `RunCreateAudioStreamModerationJob` | None | `None` | 创建音频流内容审核作业 |
| `RunCreateVideoModerationJob` | POST | `/v3/{project_id}/moderation/video/jobs` | 创建视频内容审核作业 |
| `RunQueryAudioModerationJob` | GET | `/v3/{project_id}/moderation/audio/jobs/{job_id}` | 查询音频内容审核作业 |
| `RunQueryVideoModerationJob` | GET | `/v3/{project_id}/moderation/video/jobs/{job_id}` | 查询视频内容审核作业 |
| `RunTextModeration` | POST | `/v3/{project_id}/moderation/text` | 文本内容审核 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
