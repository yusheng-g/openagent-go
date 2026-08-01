---
name: huaweicloud-vcm
description: HuaweiCloud VCM API guide. 8 APIs covering 视频内容审核, 长语音内容审核.
---

# HuaweiCloud VCM API Guide

8 APIs. Tags: 视频内容审核, 长语音内容审核

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CheckAudioJob` | GET | `/v2/{project_id}/services/audio-moderation/tasks/{task_id}` | 查询单个作业 |
| `CheckVideoJob` | GET | `/v2/{project_id}/services/video-moderation/tasks/{task_id}` | 查询单个作业 |
| `CreateAudioJob` | POST | `/v2/{project_id}/services/audio-moderation/tasks` | 创建作业 |
| `CreateVideoJob` | POST | `/v2/{project_id}/services/video-moderation/tasks` | 创建作业 |
| `DeleteDemoInfo` | DELETE | `/v2/{project_id}/services/audio-moderation/tasks/{task_id}` | 删除语音作业 |
| `DeleteVideoJob` | DELETE | `/v2/{project_id}/services/video-moderation/tasks/{task_id}` | 删除作业 |
| `ListAudioJobs` | GET | `/v2/{project_id}/services/audio-moderation/tasks` | 查询作业列表 |
| `ListVideoJobs` | GET | `/v2/{project_id}/services/video-moderation/tasks` | 查询作业列表 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
