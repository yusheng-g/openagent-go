---
name: huaweicloud-sis
description: HuaweiCloud SIS API guide. 13 APIs covering 声音复刻接口, 热词管理接口, 语音合成接口, 语音识别接口.
---

# HuaweiCloud SIS API Guide

13 APIs. Tags: 声音复刻接口, 热词管理接口, 语音合成接口, 语音识别接口

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CollectTranscriberJob` | GET | `/v1/{project_id}/asr/transcriber/jobs/{job_id}` | 获取录音文件识别结果 |
| `CreateVocabulary` | POST | `/v1/{project_id}/asr/vocabularies` | 创建热词表 |
| `CreateVoice` | POST | `/v1/{project_id}/vcs/voices` | 注册接口 |
| `DeleteVocabulary` | DELETE | `/v1/{project_id}/asr/vocabularies/{vocabulary_id}` | 删除热词表 |
| `GenerateSpeech` | POST | `/v1/{project_id}/vcs/voices/clone` | 合成接口 |
| `ListVoices` | GET | `/v1/{project_id}/vcs/voices` | 查询接口 |
| `PushTranscriberJobs` | POST | `/v1/{project_id}/asr/transcriber/jobs` | 提交录音文件识别任务 |
| `RecognizeFlashAsr` | POST | `/v1/{project_id}/asr/flash` | 录音文件识别极速版 |
| `RecognizeShortAudio` | POST | `/v1/{project_id}/asr/short-audio` | 一句话识别 |
| `RunTts` | POST | `/v1/{project_id}/tts` | 语音合成 |
| `ShowVocabularies` | GET | `/v1/{project_id}/asr/vocabularies` | 查询热词表列表 |
| `ShowVocabulary` | GET | `/v1/{project_id}/asr/vocabularies/{vocabulary_id}` | 查询热词表信息 |
| `UpdateVocabulary` | PUT | `/v1/{project_id}/asr/vocabularies/{vocabulary_id}` | 更新热词表 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
