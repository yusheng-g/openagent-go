---
name: huaweicloud-cbs
description: HuaweiCloud CBS API guide. 29 APIs covering 其他问答, 其他问答API, 形象管理, 素材管理, 视频管理.
---

# HuaweiCloud CBS API Guide

29 APIs. Tags: 其他问答, 其他问答API, 形象管理, 素材管理, 视频管理, 问答会话, 问答机器人, 问答统计

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CollectHotQuestions` | GET | `/v1/{project_id}/qabots/{qabot_id}/qa-pairs/hots` | 热点问题统计 |
| `CollectKeyWords` | GET | `/v1/{project_id}/qabots/{qabot_id}/requests/keywords` | 关键词统计 |
| `CollectReplyRates` | GET | `/v1/{project_id}/qabots/{qabot_id}/requests/reply-rates` | 问答统计 |
| `CollectSessionStats` | GET | `/v1/{project_id}/qabots/{qabot_id}/requests/session-stats` | 访问统计 |
| `CreateSession` | POST | `/v1/{project_id}/qabots/{qabot_id}/sessions` | 开启会话 |
| `DeleteSession` | DELETE | `/v1/{project_id}/qabots/{qabot_id}/sessions/{session_id}` | 关闭会话 |
| `ExecuteComposeVideo` | POST | `/v1/{project_id}/digital-human/video/{video_id}/compose` | 合成视频(按包周期收费) |
| `ExecuteComposeVideoOndemand` | POST | `/v1/{project_id}/digital-human/video/{video_id}/compose/on-demand` | 合成视频(按需收费) |
| `ExecuteCreateVideo` | POST | `/v1/{project_id}/digital-human/video` | 创建视频 |
| `ExecuteDeleteimageById` | DELETE | `/v1/{project_id}/digital-human/images/{image_id}` | 删除图片 |
| `ExecuteDeleteVideoById` | DELETE | `/v1/{project_id}/digital-human/video/{video_id}` | 删除视频 |
| `ExecuteGetCharacterInfoById` | GET | `/v1/{project_id}/digital-human/characters/{character_id}` | 获取形象详情 |
| `ExecuteGetCharacters` | GET | `/v1/{project_id}/digital-human/characters` | 获取形象列表 |
| `ExecuteGetFramsListByImagesId` | GET | `/v1/{project_id}/digital-human/image-frames` | 获取播报框 |
| `ExecuteGetImagesList` | GET | `/v1/{project_id}/digital-human/images` | 获取图片列表 |
| `ExecuteGetVideoInfoById` | GET | `/v1/{project_id}/digital-human/video/{video_id}/info` | 获取视频详情 |
| `ExecuteGetVideosList` | GET | `/v1/{project_id}/digital-human/video` | 获取视频列表 |
| `ExecutePostCreateImages` | POST | `/v1/{project_id}/digital-human/images` | 创建图片 |
| `ExecuteQaChat` | POST | `/v1/{project_id}/qabots/{qabot_id}/chat` | 问答机器人会话 |
| `ExecuteSession` | POST | `/v1/{project_id}/qabots/{qabot_id}/sessions/{session_id}` | 处理会话 |
| `ExecuteUpdateImageName` | PUT | `/v1/{project_id}/digital-human/images/{image_id}` | 修改图片名 |
| `ExecuteUpdateVideoById` | PUT | `/v1/{project_id}/digital-human/video/{video_id}` | 更新视频名 |
| `ExecuteUpdateVideoInfoById` | PUT | `/v1/{project_id}/digital-human/video/{video_id}/info` | 配置视频 |
| `ExecuteUploadImage` | POST | `/v1/{project_id}/digital-human/video/{video_id}/upload/image` | 上传播报插图 |
| `ExecuteUploadPpt` | POST | `/v1/{project_id}/digital-human/video/{video_id}/upload/ppt` | 通过pdf上传多张插图 |
| `ListSuggestions` | POST | `/v1/{project_id}/qabots/{qabot_id}/suggestions` | 获取问题提示 |
| `PostRequests` | POST | `/v1/{project_id}/qabots/{qabot_id}/requests` | PostRequests |
| `TagLabor` | POST | `/v1/{project_id}/qabots/{qabot_id}/requests/{request_id}/labor` | 标记为转人工 |
| `TagSatisfaction` | POST | `/v1/{project_id}/qabots/{qabot_id}/requests/{request_id}/satisfaction` | 问答满意评价 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
