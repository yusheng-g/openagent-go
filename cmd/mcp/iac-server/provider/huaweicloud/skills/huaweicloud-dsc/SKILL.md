---
name: huaweicloud-dsc
description: HuaweiCloud DSC API guide. 35 APIs covering API调用记录, 告警通知, 图片水印, 敏感数据发现, 数据动态脱敏.
---

# HuaweiCloud DSC API Guide

35 APIs. Tags: API调用记录, 告警通知, 图片水印, 敏感数据发现, 数据动态脱敏, 数据水印, 数据静态脱敏, 文档水印, 资产管理, 资源管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddBuckets` | POST | `/v1/{project_id}/sdg/asset/obs/buckets` | 添加资产授权 |
| `AddRule` | POST | `/v1/{project_id}/sdg/server/scan/rules` | 创建扫描规则 |
| `AddRuleGroup` | POST | `/v1/{project_id}/sdg/server/scan/groups` | 创建扫描规则组 |
| `AddScanJob` | POST | `/v1/{project_id}/sdg/scan/job` | 创建扫描任务 |
| `BatchAddDataMask` | POST | `/v1/{project_id}/data/mask` | 对数据进行脱敏 |
| `ChangeDbTemplateOperation` | POST | `/v1/{project_id}/sdg/server/mask/dbs/templates/{template_id}/operation` | 开启/停止脱敏任务 |
| `ChangeRule` | PUT | `/v1/{project_id}/sdg/server/scan/rules` | 修改扫描规则 |
| `CreateDatabaseWaterMark` | POST | `/v1/{project_id}/sdg/database/watermark/embed` | 嵌入数据水印 |
| `CreateDocWatermark` | POST | `/v1/{project_id}/sdg/doc/watermark/embed` | 文档嵌入水印 |
| `CreateDocWatermarkByAddress` | POST | `/v1/{project_id}/doc-address/watermark/embed` | 文档嵌入水印(文件地址版本) |
| `CreateImageWatermark` | POST | `/v1/{project_id}/image/watermark/embed` | 图片嵌入暗水印 |
| `CreateImageWatermarkByAddress` | POST | `/v1/{project_id}/image-address/watermark/embed` | 图片嵌入暗水印(文件地址版本) |
| `CreateProductOrder` | POST | `/v1/{project_id}/period/order` | 实例下单 |
| `DeleteBucket` | DELETE | `/v1/{project_id}/sdg/asset/obs/bucket/{bucket_id}` | 删除资产授权 |
| `DeleteRule` | DELETE | `/v1/{project_id}/sdg/server/scan/rules/{rule_id}` | 删除扫描规则 |
| `DeleteRuleGroup` | DELETE | `/v1/{project_id}/sdg/server/scan/groups/{group_id}` | 删除扫描规则组 |
| `DeleteScanJob` | DELETE | `/v1/{project_id}/sdg/scan/job/{job_id}` | 删除扫描任务 |
| `ListBuckets` | GET | `/v1/{project_id}/sdg/asset/obs/buckets` | 查看资产列表 |
| `ListDbMaskTask` | GET | `/v1/{project_id}/sdg/server/mask/dbs/templates/{template_id}/tasks` | 查询脱敏任务执行列表 |
| `ListRuleGroups` | GET | `/v1/{project_id}/sdg/server/scan/groups` | 查询扫描规则组列表 |
| `ShowDatabaseWaterMark` | POST | `/v1/{project_id}/sdg/database/watermark/extract` | 提取数据水印 |
| `ShowDocWatermark` | POST | `/v1/{project_id}/sdg/doc/watermark/extract` | 文档提取暗水印 |
| `ShowDocWatermarkByAddress` | POST | `/v1/{project_id}/doc-address/watermark/extract` | 文档提取暗水印(文档地址版本) |
| `ShowImageWatermark` | POST | `/v1/{project_id}/image/watermark/extract` | 提取图片中的文字暗水印 |
| `ShowImageWatermarkByAddress` | POST | `/v1/{project_id}/image-address/watermark/extract` | 提取图片中的文字暗水印(文件地址版本) |
| `ShowImageWatermarkWithImage` | POST | `/v1/{project_id}/image/watermark/extract-image` | 提取图片中的图片暗水印 |
| `ShowImageWatermarkWithImageByAddress` | POST | `/v1/{project_id}/image-address/watermark/extract-image` | 提取图片中的图片暗水印(文件地址版本) |
| `ShowOpenApiCalledRecords` | GET | `/v1/{project_id}/openapi/called-records` | 查询OpenApi调用记录 |
| `ShowRules` | GET | `/v1/{project_id}/sdg/server/scan/rules` | 查看规则列表 |
| `ShowScanJobResults` | GET | `/v1/{project_id}/sdg/scan/job/{job_id}/results` | 查询指定任务扫描结果 |

... and 5 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
