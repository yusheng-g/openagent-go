---
name: huaweicloud-imagesearch
description: HuaweiCloud ImageSearch API guide. 5 APIs covering 删除数据, 搜索, 更新数据, 检查数据, 添加数据.
---

# HuaweiCloud ImageSearch API Guide

5 APIs. Tags: 删除数据, 搜索, 更新数据, 检查数据, 添加数据

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `RunAddData` | POST | `/v2/{project_id}/mms/{service_name}/data/add` | 添加数据 |
| `RunCheckData` | POST | `/v2/{project_id}/mms/{service_name}/data/check` | 检查数据 |
| `RunDeleteData` | POST | `/v2/{project_id}/mms/{service_name}/data/delete` | 删除数据 |
| `RunSearch` | POST | `/v2/{project_id}/mms/{service_name}/search` | 搜索 |
| `RunUpdateData` | POST | `/v2/{project_id}/mms/{service_name}/data/update` | 更新数据 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
