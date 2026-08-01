---
name: huaweicloud-voicecall
description: HuaweiCloud VoiceCall API guide. 5 APIs covering 语音回呼, 语音通知, 语音验证码.
---

# HuaweiCloud VoiceCall API Guide

5 APIs. Tags: 语音回呼, 语音通知, 语音验证码

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateCallBack` | POST | `/rest/httpsessions/click2Call/v2.0` | 语音回呼 |
| `CreateCallNotify` | POST | `/rest/httpsessions/callnotify/v2.0` | 语音通知 |
| `CreateCallVerify` | POST | `/rest/httpsessions/callVerify/v1.0` | 语音验证码 |
| `ShowVoiceRecord` | GET | `/rest/provision/voice/record/v1.0` | 获取录音文件URL |
| `StopCallBack` | POST | `/rest/httpsessions/callStop/v2.0` | 终止呼叫 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
