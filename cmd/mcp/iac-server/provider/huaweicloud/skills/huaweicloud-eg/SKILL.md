---
name: huaweicloud-eg
description: HuaweiCloud EG API guide. 65 APIs covering API版本管理, obs桶管理, 事件模型管理, 事件流管理, 事件源管理.
---

# HuaweiCloud EG API Guide

65 APIs. Tags: API版本管理, obs桶管理, 事件模型管理, 事件流管理, 事件源管理, 事件目标分类管理, 事件示例管理, 事件管理, 事件订阅管理, 事件通道管理, 服务委托管理, 特性管理, 监控指标管理, 目标连接管理, 触发器管理, 访问端点管理, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CheckPutEvents` | POST | `/v1/{project_id}/events/check` | 预校验指定事件源发布事件成功 |
| `CreateAgencies` | POST | `/v1/{project_id}/service-agencies` | 创建服务委托 |
| `CreateChannel` | POST | `/v1/{project_id}/channels` | 创建自定义事件通道 |
| `CreateConnection` | POST | `/v1/{project_id}/connections` | 创建目标连接 |
| `CreateEndpoint` | POST | `/v1/{project_id}/endpoints` | 创建访问端点 |
| `CreateEventSchema` | POST | `/v1/{project_id}/schemas` | 创建自定义事件模型 |
| `CreateEventSchemaVersion` | POST | `/v1/{project_id}/schemas/{schema_id}/versions` | 创建自定义事件模型版本 |
| `CreateEventSource` | POST | `/v1/{project_id}/sources` | 创建自定义事件源 |
| `CreateEventStreaming` | POST | `/v1/{project_id}/eventstreamings` | 创建事件流 |
| `CreateSubscription` | POST | `/v1/{project_id}/subscriptions` | 创建事件订阅 |
| `CreateSubscriptionTarget` | POST | `/v1/{project_id}/subscriptions/{subscription_id}/targets` | 创建事件订阅目标 |
| `DeleteChannel` | DELETE | `/v1/{project_id}/channels/{channel_id}` | 删除自定义事件通道 |
| `DeleteConnection` | DELETE | `/v1/{project_id}/connections/{connection_id}` | 删除目标连接 |
| `DeleteEndpoint` | DELETE | `/v1/{project_id}/endpoints/{endpoint_id}` | 删除访问端点 |
| `DeleteEventSchema` | DELETE | `/v1/{project_id}/schemas/{schema_id}` | 删除事件模型 |
| `DeleteEventSchemaVersion` | DELETE | `/v1/{project_id}/schemas/{schema_id}/versions/{version}` | 删除事件模型版本 |
| `DeleteEventSource` | DELETE | `/v1/{project_id}/sources/{source_id}` | 删除自定义事件源 |
| `DeleteEventStreaming` | DELETE | `/v1/{project_id}/eventstreamings/{eventstreaming_id}` | 删除事件流 |
| `DeleteSubscription` | DELETE | `/v1/{project_id}/subscriptions/{subscription_id}` | 删除事件订阅 |
| `DeleteSubscriptionTarget` | DELETE | `/v1/{project_id}/subscriptions/{subscription_id}/targets/{target_id}` | 删除事件订阅目标 |
| `DiscoverEventSchemaFromData` | POST | `/v1/{project_id}/schema-discover` | 事件模型自动发现 |
| `ExecuteSubscriptionOperation` | POST | `/v1/{project_id}/subscriptions/operation` | 操作事件订阅 |
| `ListAgencies` | GET | `/v1/{project_id}/service-agencies` | 查询服务委托 |
| `ListApiVersions` | GET | `/` | 获取API版本列表 |
| `ListChannels` | GET | `/v1/{project_id}/channels` | 查询事件通道列表 |
| `ListConnections` | GET | `/v1/{project_id}/connections` | 查询目标连接列表 |
| `ListEndpoints` | GET | `/v1/{project_id}/endpoints` | 查询访问端点 |
| `ListEventSchema` | GET | `/v1/{project_id}/schemas` | 查询事件模型列表 |
| `ListEventSchemaVersions` | GET | `/v1/{project_id}/schemas/{schema_id}/versions` | 查询事件模型版本列表 |
| `ListEventSources` | GET | `/v1/{project_id}/sources` | 查询事件源列表 |

... and 35 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
