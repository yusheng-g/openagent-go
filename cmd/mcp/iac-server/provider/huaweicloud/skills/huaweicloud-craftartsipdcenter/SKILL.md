---
name: huaweicloud-craftartsipdcenter
description: HuaweiCloud CraftArtsIPDCenter API guide. 8 APIs covering 数字化制造云平台生产数据管理.
---

# HuaweiCloud CraftArtsIPDCenter API Guide

8 APIs. Tags: 数字化制造云平台生产数据管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCancelWos` | POST | `/wom/openapi/v1/wos/batch-cancel` | 批量取消工单 |
| `BatchCreateWoInstantiations` | POST | `/wom/openapi/v1/wos/batch-instantiate` | 批量实例化工单 |
| `BatchCreateWos` | POST | `/wom/openapi/v1/wos/batch-create` | 导入工单 |
| `BatchDeleteWos` | POST | `/wom/openapi/v1/wos/batch-delete` | 批量删除工单 |
| `BatchGenerateWoSchemes` | POST | `/wom/openapi/v1/wo-schemes/batch-generate` | 批量生成工单方案 |
| `SearchWoInfo` | POST | `/wom/openapi/v1/wos/wo-info` | 按工单获取工单相关信息 |
| `SearchWoPartInfo` | GET | `/wom/openapi/v1/wos/wo-part-info` | 获取工单产品信息 |
| `SearchWosForPage` | POST | `/wom/openapi/v1/wos/batch-detail` | 分页查询工单 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
