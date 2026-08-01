---
name: huaweicloud-live
description: HuaweiCloud Live API guide. 122 APIs covering CES查询, HTTPS证书管理, OBS桶管理, OTT频道管理, 域名管理.
---

# HuaweiCloud Live API Guide

122 APIs. Tags: CES查询, HTTPS证书管理, OBS桶管理, OTT频道管理, 域名管理, 录制回调管理, 录制管理, 录制索引管理, 截图管理, 拉流管理, 数据统计分析, 日志管理, 流监控, 流管理, 流连接管理, 直播水印管理, 转码模板管理, 通知管理, 鉴权管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchShowIpBelongs` | GET | `/v1/{project_id}/cdn/ip-info` | 查询IP归属信息 |
| `CheckDomainVerification` | POST | `/v1/{project_id}/domain/verification` | 域名归属权认证 |
| `CreateDomain` | POST | `/v1/{project_id}/domain` | 创建直播域名 |
| `CreateDomainMapping` | PUT | `/v1/{project_id}/domains_mapping` | 域名映射 |
| `CreateFlowOutput` | POST | `/v1/{project_id}/flows/outputs` | 创建转推输出 |
| `CreateFlows` | POST | `/v1/{project_id}/flows` | 创建流 |
| `CreateOttChannelInfo` | POST | `/v1/{project_id}/ott/channels` | 新建OTT频道 |
| `CreatePullTask` | POST | `/v1/{project_id}/pull/stream/task` | 创建直播拉流转推任务 |
| `CreateRecordCallbackConfig` | POST | `/v1/{project_id}/record/callbacks` | 创建录制回调配置 |
| `CreateRecordIndex` | POST | `/v1/{project_id}/record/indexes` | 创建录制视频索引文件 |
| `CreateRecordRule` | POST | `/v1/{project_id}/record/rules` | 创建录制规则 |
| `CreateSnapshotConfig` | POST | `/v1/{project_id}/stream/snapshot` | 创建直播截图配置 |
| `CreateStreamForbidden` | POST | `/v1/{project_id}/stream/blocks` | 禁止直播推流 |
| `CreateStreamForbiddenOnce` | POST | `/v1/{project_id}/stream/block-once` | 禁推闪断 |
| `CreateTranscodingsTemplate` | POST | `/v1/{project_id}/template/transcodings` | 创建直播转码模板 |
| `CreateUrlAuthchain` | POST | `/v1/{project_id}/auth/chain` | 生成URL鉴权串 |
| `CreateWatermarkRule` | POST | `/v1/{project_id}/watermark/rules` | 创建水印规则 |
| `CreateWatermarkTemplate` | POST | `/v1/{project_id}/watermark/templates` | 创建水印模板 |
| `DeleteDomain` | DELETE | `/v1/{project_id}/domain` | 删除直播域名 |
| `DeleteDomainHttpsCert` | DELETE | `/v1/{project_id}/guard/https-cert` | 删除指定域名的https证书配置 |
| `DeleteDomainKeyChain` | DELETE | `/v1/{project_id}/guard/key-chain` | 删除指定域名的Key防盗链配置 |
| `DeleteDomainMapping` | DELETE | `/v1/{project_id}/domains_mapping` | 删除直播域名映射关系 |
| `DeleteFlow` | DELETE | `/v1/{project_id}/flows` | 删除流 |
| `DeleteFlowOutput` | DELETE | `/v1/{project_id}/flows/outputs` | 删除转推输出 |
| `DeleteOttChannelInfo` | DELETE | `/v1/{project_id}/ott/channels` | 删除频道信息 |
| `DeletePublishTemplate` | DELETE | `/v1/{project_id}/notifications/publish` | 删除直播推流通知配置 |
| `DeletePullTask` | DELETE | `/v1/{project_id}/pull/stream/task` | 删除直播拉流转推任务 |
| `DeleteRecordCallbackConfig` | DELETE | `/v1/{project_id}/record/callbacks/{id}` | 删除录制回调配置 |
| `DeleteRecordRule` | DELETE | `/v1/{project_id}/record/rules/{id}` | 删除录制规则 |
| `DeleteRefererChain` | DELETE | `/v1/{project_id}/guard/referer-chain` | 删除Referer防盗链黑白名单 |

... and 92 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
