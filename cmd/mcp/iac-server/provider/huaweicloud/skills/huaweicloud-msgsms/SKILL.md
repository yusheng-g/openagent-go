---
name: huaweicloud-msgsms
description: HuaweiCloud MSGSMS API guide. 20 APIs covering 短信应用API, 短信签名API, 短信签名模板API.
---

# HuaweiCloud MSGSMS API Guide

20 APIs. Tags: 短信应用API, 短信签名API, 短信签名模板API

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateApp` | POST | `/v2/{project_id}/msgsms/apps` | 创建短信应用 |
| `CreateSignature` | POST | `/v2/{project_id}/msgsms/signatures` | 创建短信签名 |
| `CreateTemplate` | POST | `/v2/{project_id}/msgsms/templates` | 创建短信模板 |
| `DeleteSignature` | DELETE | `/v2/{project_id}/msgsms/signatures/{id}` | 删除短信签名 |
| `DeleteTemplate` | DELETE | `/v2/{project_id}/msgsms/templates/{id}` | 删除短信模板 |
| `EnableSignature` | PUT | `/v2/{project_id}/msgsms/signatures/{id}/active` | 申请激活签名 |
| `ListAppDetails` | GET | `/v2/{project_id}/msgsms/apps` | 查询短信应用 |
| `ListSendCountryDetails` | GET | `/v2/{project_id}/msgsms/country` | 查询发送国家 |
| `ListSignatureDetails` | GET | `/v2/{project_id}/msgsms/signatures` | 查询签名信息 |
| `ListTemplateDetails` | GET | `/v2/{project_id}/msgsms/templates` | 查询短信模板 |
| `ListTemplateVarilableDetails` | GET | `/v2/{project_id}/msgsms/templates/{id}/varilable` | 查询模板变量 |
| `ShowApp` | GET | `/v2/{project_id}/msgsms/apps/{id}` | 获取应用详情 |
| `ShowAppCount` | GET | `/v2/{project_id}/msgsms/apps-count` | 查询应用数量 |
| `ShowSignature` | GET | `/v2/{project_id}/msgsms/signatures/{id}` | 获取签名详情 |
| `ShowSignatureFile` | GET | `/v2/{project_id}/msgsms/upload-files` | 查询申请文件 |
| `ShowTemplate` | GET | `/v2/{project_id}/msgsms/templates/{id}` | 获取模板详情 |
| `UpdateApp` | PUT | `/v2/{project_id}/msgsms/apps/{id}` | 修改短信应用 |
| `UpdateSignature` | PUT | `/v2/{project_id}/msgsms/signatures/{id}` | 修改短信签名 |
| `UpdateTemplate` | PUT | `/v2/{project_id}/msgsms/templates/{id}` | 修改短信模板 |
| `UploadSignatureFile` | POST | `/v2/{project_id}/msgsms/upload-files` | 上传申请文件 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
