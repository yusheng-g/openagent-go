---
name: huaweicloud-dcos
description: HuaweiCloud DCOS API guide. 8 APIs covering 文件管理, 服务单管理, 资产管理.
---

# HuaweiCloud DCOS API Guide

8 APIs. Tags: 文件管理, 服务单管理, 资产管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ListOrder` | GET | `/v1/orders/list` | 客户查询服务单列表 |
| `SaveOrder` | POST | `/v1/orders/save` | 客户创建服务单 |
| `ShowOrder` | GET | `/v1/orders/{number}` | 客户查询服务单详情 |
| `ShowOrderCatalogue` | GET | `/v1/orders/catalogue` | 获取服务单目录列表 |
| `ShowOrderInformation` | GET | `/v1/orders/information/{model_code}` | 获取服务服务单项目信息 |
| `ShowPageAssetListResult` | POST | `/v1/assets` | 资产列表 |
| `UploadFile` | POST | `/v1/files/upload-file` | 上传附件 |
| `VerifyOrder` | POST | `/v1/orders/verify/{number}` | 验收服务单 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
