---
name: huaweicloud-dns
description: HuaweiCloud DNS API guide. 117 APIs covering DNSSEC, 公网域名检测, 公网域名管理, 内网域名管理, 反向解析管理(v2). Detailed swagger definitions in references/<APIName>.json.
---

# HuaweiCloud DNS API Guide

117 APIs. Tags: DNSSEC, 公网域名检测, 公网域名管理, 内网域名管理, 反向解析管理(v2), 反向解析管理(v2.1), 名称服务器管理, 批量操作, 标签管理, 版本管理, 自定义线路管理, 解析器管理, 记录集管理(v2), 记录集管理(v2.1), 访问日志管理, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AssociateEndpointIpaddress` | POST | `/v2.1/endpoints/{endpoint_id}/ipaddresses` | 终端节点绑定IP地址 |
| `AssociateResolverQueryLogConfig` | POST | `/v2/resolver/queryloggingconfig/{id}/associatevpc` | 解析器访问日志关联VPC |
| `AssociateResolverRuleRouter` | POST | `/v2.1/resolverrules/{resolverrule_id}/associaterouter` | 解析器转发规则关联VPC |
| `AssociateRouter` | POST | `/v2/zones/{zone_id}/associaterouter` | 在内网域名上关联VPC |
| `BatchCreateCombinedPublicRecordsetsTask` | POST | `/v2.1/operation-task/batch-create-combined-recordset` | 批量创建公网记录集 |
| `BatchCreatePublicRecordsetsTask` | POST | `/v2.1/operation-task/batch-create-recordset` | 批量创建公网记录集 |
| `BatchCreatePublicZonesTask` | POST | `/v2.1/operation-task/batch-create-zone` | 批量创建公网域名 |
| `BatchCreateRecordSetsTask` | POST | `/v2.1/zones/{zone_id}/recordsets/batch-create-task` | 批量创建记录集 |
| `BatchCreateTag` | POST | `/v2/{project_id}/{resource_type}/{resource_id}/tags/action` | 为指定实例批量添加或删除标签 |
| `BatchDeletePtrRecords` | DELETE | `/v2.1/reverse/floatingips` | 批量删除反向解析 |
| `BatchDeletePublicRecordsetsTask` | POST | `/v2.1/operation-task/batch-delete-recordset` | 批量删除公网记录集 |
| `BatchDeleteRecordSets` | DELETE | `/v2.1/recordsets` | 批量删除记录集 |
| `BatchDeleteRecordSetWithLine` | DELETE | `/v2.1/zones/{zone_id}/recordsets` | 批量删除域名下的记录集 |
| `BatchDeleteZones` | DELETE | `/v2.1/zones` | 批量删除域名 |
| `BatchSetRecordSetsStatus` | PUT | `/v2.1/recordsets/statuses` | 批量设置记录集状态 |
| `BatchSetZonesStatus` | PUT | `/v2.1/zones/statuses` | 批量设置域名状态 |
| `BatchTransferPublicZonesTask` | POST | `/v2.1/operation-task/batch-transfer` | 批量转移公网域名 |
| `BatchUpdatePublicRecordsetsTask` | POST | `/v2.1/operation-task/batch-update-recordset` | 批量修改公网记录集 |
| `BatchUpdateRecordSetWithLine` | PUT | `/v2.1/zones/{zone_id}/recordsets` | 批量修改记录集 |
| `CreateAuthorizeTxtRecord` | POST | `/v2/authorize-txtrecord` | 创建公网子域名授权 |
| `CreateAuthorizeTxtRecordVerification` | POST | `/v2/authorize-txtrecord/{id}/verify` | 验证公网子域名授权 |
| `CreateCustomLine` | POST | `/v2.1/customlines` | 创建自定义线路 |
| `CreateEipRecordSet` | PATCH | `/v2/reverse/floatingips/{region}:{floatingip_id}` | 设置弹性公网IP的反向解析记录 |
| `CreateEndpoint` | POST | `/v2.1/endpoints` | 创建终端节点 |
| `CreateLineGroup` | POST | `/v2.1/linegroups` | 创建线路分组 |
| `CreatePrivateZone` | POST | `/v2/zones` | 创建内网域名 |
| `CreatePtr` | POST | `/v2.1/ptrs` | 创建弹性公网IP的反向解析记录 |
| `CreatePublicZone` | POST | `/v2/zones` | 创建公网域名 |
| `CreateRecordSet` | POST | `/v2/zones/{zone_id}/recordsets` | 创建记录集 |
| `CreateRecordSetWithBatchLines` | POST | `/v2.1/zones/{zone_id}/recordsets/batch/lines` | 批量线路创建记录集 |

... and 87 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
