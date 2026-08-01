---
name: huaweicloud-image
description: HuaweiCloud Image API guide. 6 APIs covering 主体识别, 名人识别, 图像标签, 媒资图像标签(分类), 媒资图像标签(检测).
---

# HuaweiCloud Image API Guide

6 APIs. Tags: 主体识别, 名人识别, 图像标签, 媒资图像标签(分类), 媒资图像标签(检测), 翻拍识别

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `RunCelebrityRecognition` | POST | `/v2/{project_id}/image/celebrity-recognition` | 名人识别 |
| `RunImageMainObjectDetection` | POST | `/v3/{project_id}/image/main-object-detection` | 主体识别 |
| `RunImageMediaTagging` | POST | `/v2/{project_id}/image/media-tagging` | 标签识别 |
| `RunImageMediaTaggingDet` | POST | `/v2/{project_id}/image/media-tagging-det` | 媒资图像标签检测 |
| `RunImageTagging` | POST | `/v2/{project_id}/image/tagging` | 图像标签 |
| `RunRecaptureDetect` | POST | `/v2/{project_id}/image/recapture-detect` | 翻拍识别 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
