---
name: huaweicloud-obs
description: HuaweiCloud OBS API guide. 94 APIs covering 多段操作, 对象操作, 桶的基础操作, 桶的高级配置, 静态网站托管. Detailed swagger definitions in references/<APIName>.json.
---

# HuaweiCloud OBS API Guide

94 APIs. Tags: 多段操作, 对象操作, 桶的基础操作, 桶的高级配置, 静态网站托管

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AbortMultipartUpload` | DELETE | `/{object_key}` | 取消多段上传任务 |
| `AppendObject` | POST | `/{object_key}` | 追加写对象 |
| `CheckBucketOptions` | OPTIONS | `/` | OPTIONS桶 |
| `CheckObjectOptions` | OPTIONS | `/{object_key}` | OPTIONS对象 |
| `CompleteMultipartUpload` | POST | `/{object_key}` | 合并段 |
| `CopyObject` | PUT | `/{object_key}` | 复制对象 |
| `CopyPart` | PUT | `/{object_key}` | 拷贝段 |
| `CreateBucket` | PUT | `/` | 创建桶 |
| `DeleteBucket` | DELETE | `/` | 删除桶 |
| `DeleteBucketCors` | DELETE | `/` | 删除桶的CORS配置 |
| `DeleteBucketCustomdomain` | DELETE | `/` | 删除桶的自定义域名 |
| `DeleteBucketEncryption` | DELETE | `/` | 删除桶的加密配置 |
| `DeleteBucketInventory` | DELETE | `/` | 删除桶清单 |
| `DeleteBucketLifecycle` | DELETE | `/` | 删除桶的生命周期配置 |
| `DeleteBucketMirrorBackToSource` | DELETE | `/` | 删除桶的镜像回源规则 |
| `DeleteBucketObsCompressPolicy` | DELETE | `/` | 删除桶在线解压策略 |
| `DeleteBucketPolicy` | DELETE | `/` | 删除桶策略 |
| `DeleteBucketPublicAccessBlock` | DELETE | `/` | 删除桶级阻止公共访问配置 |
| `DeleteBucketReplication` | DELETE | `/` | 删除桶的跨区域复制配置 |
| `DeleteBucketTagging` | DELETE | `/` | 删除桶标签 |
| `DeleteBucketWebsite` | DELETE | `/` | 删除桶的网站配置 |
| `DeleteDirectcoldaccess` | DELETE | `/` | 删除桶归档对象直读策略 |
| `DeleteDisPolicy` | DELETE | `/` | 删除DIS通知策略 |
| `DeleteObject` | DELETE | `/{object_key}` | 删除对象 |
| `DeleteObjects` | POST | `/` | 批量删除对象 |
| `DeleteObjectTagging` | DELETE | `/{object_key}` | 删除对象标签 |
| `GetBucketAcl` | GET | `/` | 获取桶ACL |
| `GetBucketCors` | GET | `/` | 获取桶的CORS配置 |
| `GetBucketCustomdomain` | GET | `/` | 获取桶的自定义域名 |
| `GetBucketEncryption` | GET | `/` | 获取桶的加密配置 |

... and 64 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
