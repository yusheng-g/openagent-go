---
name: huaweicloud-dss
description: HuaweiCloud DSS API guide. 4 APIs covering DSS管理.
---

# HuaweiCloud DSS API Guide

4 APIs. Tags: DSS管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ListPools` | GET | `/v1/{project_id}/pools/detail` | 获取专属存储详情列表 |
| `ListVersion` | GET | `/` | 查询版本号列表 |
| `ShowPool` | GET | `/v1/{project_id}/pools/{dss_id}` | 获取单个专属存储池详情 |
| `ShowVersions` | GET | `/{api_version}` | 查询指定版本号详情 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
