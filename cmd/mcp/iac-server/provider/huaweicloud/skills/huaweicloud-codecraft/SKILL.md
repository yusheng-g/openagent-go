---
name: huaweicloud-codecraft
description: HuaweiCloud CodeCraft API guide. 4 APIs covering 作品管理.
---

# HuaweiCloud CodeCraft API Guide

4 APIs. Tags: 作品管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateCompetitionScore` | POST | `/v5/competitions/score-infos` | 登记第三方提交的作品信息(得分回调) |
| `ListCompetitionWorks` | GET | `/v5/competitions/works` | 获取指定时间内选手提交的作品 |
| `RegisterCompetitionInfo` | POST | `/v5/competitions/registrations` | 验证用户报名信息和团队信息 |
| `UpdateCompetitionScore` | PUT | `/v5/competitions/scores` | 修改平台提交的作品分数(得分回调) |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
