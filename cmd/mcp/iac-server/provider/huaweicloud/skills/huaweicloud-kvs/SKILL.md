---
name: huaweicloud-kvs
description: HuaweiCloud KVS API guide. 14 APIs covering KV接口, 仓接口, 表接口.
---

# HuaweiCloud KVS API Guide

14 APIs. Tags: KV接口, 仓接口, 表接口

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchGetKv` | POST | `/v1/batch-get-kv` | 批量读请求 |
| `BatchWriteKv` | POST | `/v1/batch-write-kv` | 批量写请求 |
| `CheckHealth` | POST | `/v1/check-health` | 网络信道健康检查 |
| `CreateTable` | POST | `/v1/create-table` | 创建表 |
| `DeleteKv` | POST | `/v1/delete-kv` | 删除单个kv |
| `DeleteTable` | POST | `/v1/delete-table` | 删除表 |
| `DescribeTable` | POST | `/v1/describe-table` | 查询表 |
| `GetKv` | POST | `/v1/get-kv` | 查询单个kv |
| `ListStore` | POST | `/v1/list-store` | 列举仓 |
| `ListTable` | POST | `/v1/list-table` | 列举表 |
| `PutKv` | POST | `/v1/put-kv` | 上传单个kv |
| `ScanKv` | POST | `/v1/scan-kv` | 扫描所有kv |
| `ScanSkeyKv` | POST | `/v1/scan-skey-kv` | 扫描分区键内kv |
| `UpdateKv` | POST | `/v1/update-kv` | 更新单个kv |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
