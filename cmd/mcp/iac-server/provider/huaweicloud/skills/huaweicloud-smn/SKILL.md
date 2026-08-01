---
name: huaweicloud-smn
description: HuaweiCloud SMN API guide. 71 APIs covering Application endpoint操作, Application操作, Application直发消息操作, 主题操作, 云日志操作.
---

# HuaweiCloud SMN API Guide

71 APIs. Tags: Application endpoint操作, Application操作, Application直发消息操作, 主题操作, 云日志操作, 使用标签管理服务, 协议操作, 发布消息操作, 密钥操作, 授权云服务操作, 查询版本操作, 模板操作, 订阅操作, 订阅过滤策略操作, 证书操作, 通知策略

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddSubscription` | POST | `/v2/{project_id}/notifications/topics/{topic_urn}/subscriptions` | 订阅 |
| `AddSubscriptionFromSubscriptionUser` | POST | `/v2/{project_id}/notifications/topics/{topic_urn}/subscriptions/from-subscription-users` | 导入订阅 |
| `BatchCreateOrDeleteResourceTags` | POST | `/v2/{project_id}/{resource_type}/{resource_id}/tags/action` | 批量添加删除资源标签 |
| `BatchCreateSubscriptionsFilterPolices` | POST | `/v2/{project_id}/notifications/subscriptions/filter_polices` | 批量创建订阅过滤策略 |
| `BatchDeleteSubscriptions` | DELETE | `/v2/{project_id}/notifications/subscriptions` | 批量删除订阅 |
| `BatchDeleteSubscriptionsByTopic` | DELETE | `/v2/{project_id}/notifications/topics/{topic_urn}/subscriptions` | 批量删除指定主题的订阅 |
| `BatchDeleteSubscriptionsFilterPolices` | DELETE | `/v2/{project_id}/notifications/subscriptions/filter_polices` | 批量删除订阅过滤策略 |
| `BatchUpdateSubscriptionsFilterPolices` | PUT | `/v2/{project_id}/notifications/subscriptions/filter_polices` | 批量更新订阅过滤策略 |
| `CancelSubscription` | DELETE | `/v2/{project_id}/notifications/subscriptions/{subscription_urn}` | 取消订阅 |
| `ConfirmSubscription` | GET | `/rest/v2/notifications/subscription/confirm` | 确认订阅 |
| `CreateApplication` | POST | `/v2/{project_id}/notifications/applications` | 创建Application |
| `CreateApplicationEndpoint` | POST | `/v2/{project_id}/notifications/applications/{application_urn}/endpoints` | 创建Application endpoint |
| `CreateKmsKey` | POST | `/v2/{project_id}/notifications/topics/{topic_urn}/kms` | 主题绑定KMS密钥 |
| `CreateLogtank` | POST | `/v2/{project_id}/notifications/topics/{topic_urn}/logtanks` | 绑定云日志 |
| `CreateMessageTemplate` | POST | `/v2/{project_id}/notifications/message_template` | 创建消息模板 |
| `CreateNotifyPolicy` | POST | `/v2/{project_id}/notifications/topics/{topic_urn}/notify-policy` | 创建通知策略 |
| `CreateResourceTag` | POST | `/v2/{project_id}/{resource_type}/{resource_id}/tags` | 添加资源标签 |
| `CreateTopic` | POST | `/v2/{project_id}/notifications/topics` | 创建主题 |
| `DeleteApplication` | DELETE | `/v2/{project_id}/notifications/applications/{application_urn}` | 删除Application |
| `DeleteApplicationEndpoint` | DELETE | `/v2/{project_id}/notifications/endpoints/{endpoint_urn}` | 删除Application endpoint |
| `DeleteKmsKey` | DELETE | `/v2/{project_id}/notifications/topics/{topic_urn}/kms/{id}` | 删除主题下KMS密钥 |
| `DeleteLogtank` | DELETE | `/v2/{project_id}/notifications/topics/{topic_urn}/logtanks/{logtank_id}` | 解绑云日志 |
| `DeleteMessageTemplate` | DELETE | `/v2/{project_id}/notifications/message_template/{message_template_id}` | 删除消息模板 |
| `DeleteNotifyPolicy` | DELETE | `/v2/{project_id}/notifications/topics/{topic_urn}/notify-policy/{notify_policy_id}` | 删除通知策略 |
| `DeleteResourceTag` | DELETE | `/v2/{project_id}/{resource_type}/{resource_id}/tags/{key}` | 删除资源标签 |
| `DeleteSubscriptionsByTopic` | DELETE | `/v2/{project_id}/notifications/topics/{topic_urn}/subscriptions/{subscription_urn}` | 删除指定主题的订阅 |
| `DeleteTopic` | DELETE | `/v2/{project_id}/notifications/topics/{topic_urn}` | 删除主题 |
| `DeleteTopicAttributeByName` | DELETE | `/v2/{project_id}/notifications/topics/{topic_urn}/attributes/{name}` | 删除指定名称的主题策略 |
| `DeleteTopicAttributes` | DELETE | `/v2/{project_id}/notifications/topics/{topic_urn}/attributes` | 删除所有主题策略 |
| `DownloadHttpCert` | GET | `/smn/{certificate_id}` | 下载证书 |

... and 41 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
