---
name: huaweicloud-ivs
description: HuaweiCloud IVS API guide. 6 APIs covering 人证核身标准版(三要素), 人证核身证件版(二要素).
---

# HuaweiCloud IVS API Guide

6 APIs. Tags: 人证核身标准版(三要素), 人证核身证件版(二要素)

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `DetectExtentionByIdCardImage` | POST | `/v2.0/ivs-idcard-extention` | 人证核身证件版(二要素) |
| `DetectExtentionByNameAndId` | POST | `/v2.0/ivs-idcard-extention` | 人证核身证件版(二要素) |
| `DetectStandardByIdCardImage` | POST | `/v2.0/ivs-standard` | 人证核身标准版(三要素) |
| `DetectStandardByNameAndId` | POST | `/v2.0/ivs-standard` | 人证核身标准版(三要素) |
| `DetectStandardByVideoAndIdCardImage` | POST | `/v2.0/ivs-standard` | 人证核身标准版(三要素) |
| `DetectStandardByVideoAndNameAndId` | POST | `/v2.0/ivs-standard` | 人证核身标准版(三要素) |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
