---
name: huaweicloud-tms
description: HuaweiCloud TMS API guide. 15 APIs covering 查询标签管理支持的服务, 查询版本操作, 资源标签, 配额, 预定义标签操作.
---

# HuaweiCloud TMS API Guide

15 APIs. Tags: 查询标签管理支持的服务, 查询版本操作, 资源标签, 配额, 预定义标签操作

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreatePredefineTags` | POST | `/v1.0/predefine_tags/action` | 创建预定义标签 |
| `CreateResourceTag` | POST | `/v1.0/resource-tags/batch-create` | 批量添加标签 |
| `DeletePredefineTags` | POST | `/v1.0/predefine_tags/action` | 删除预定义标签 |
| `DeleteResourceTag` | POST | `/v1.0/resource-tags/batch-delete` | 批量移除标签 |
| `ListApiVersions` | GET | `/` | 查询API版本列表 |
| `ListPredefineTags` | GET | `/v1.0/predefine_tags` | 查询预定义标签列表 |
| `ListProviders` | GET | `/v1.0/tms/providers` | 查询标签管理支持的服务 |
| `ListResource` | POST | `/v1.0/resource-instances/filter` | 根据标签过滤资源 |
| `ListTagKeys` | GET | `/v1.0/tag-keys` | 查询标签键列表 |
| `ListTags` | GET | `/v1.0/tags` | 查询标签列表 |
| `ListTagValues` | GET | `/v1.0/tag-values` | 查询标签值列表 |
| `ShowApiVersion` | GET | `/{api_version}` | 查询API版本号详情 |
| `ShowResourceTag` | GET | `/v2.0/resources/{resource_id}/tags` | 查询资源标签 |
| `ShowTagQuota` | GET | `/v1.0/tms/quotas` | 查询标签配额 |
| `UpdatePredefineTags` | PUT | `/v1.0/predefine_tags` | 修改预定义标签 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
