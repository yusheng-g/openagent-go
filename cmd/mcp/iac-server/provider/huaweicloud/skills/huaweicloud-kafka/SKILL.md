---
name: huaweicloud-kafka
description: HuaweiCloud Kafka API guide. 113 APIs covering Smart Connect, 主题管理, 其他接口, 后台任务管理, 实例管理.
---

# HuaweiCloud Kafka API Guide

113 APIs. Tags: Smart Connect, 主题管理, 其他接口, 后台任务管理, 实例管理, 日志管理, 标签管理, 消息管理, 消费组管理, 生命周期管理, 用户管理, 规格变更管理, 诊断管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateOrDeleteKafkaTag` | POST | `/v2/{project_id}/kafka/{instance_id}/tags/action` | 批量添加或删除实例标签 |
| `BatchDeleteGroup` | POST | `/v2/{project_id}/instances/{instance_id}/groups/batch-delete` | Kafka实例批量删除消费组 |
| `BatchDeleteInstanceTopic` | POST | `/v2/{project_id}/instances/{instance_id}/topics/delete` | Kafka实例批量删除Topic |
| `BatchDeleteInstanceUsers` | PUT | `/v2/{project_id}/instances/{instance_id}/users` | 批量删除用户 |
| `BatchDeleteMessageDiagnosisReports` | DELETE | `/v2/{project_id}/kafka/instances/{instance_id}/message-diagnosis-tasks` | 批量删除消息积压诊断报告 |
| `BatchRestartOrDeleteInstances` | POST | `/v2/{project_id}/instances/action` | 批量重启或删除实例 |
| `CloseKafkaManager` | DELETE | `/v2/{project_id}/kafka/instances/{instance_id}/management` | 关闭Kafka Manager |
| `CreateConnector` | POST | `/v2/{project_id}/instances/{instance_id}/connector` | 开启Smart Connect(按需实例) |
| `CreateConnectorTask` | POST | `/v2/{project_id}/instances/{instance_id}/connector/tasks` | 创建Smart Connect任务 |
| `CreateInstanceTopic` | POST | `/v2/{project_id}/instances/{instance_id}/topics` | Kafka实例创建Topic |
| `CreateInstanceUser` | POST | `/v2/{project_id}/instances/{instance_id}/users` | 创建用户 |
| `CreateKafkaConsumerGroup` | POST | `/v2/{project_id}/kafka/instances/{instance_id}/group` | 创建消费组 |
| `CreateKafkaReassignmentTask` | POST | `/v2/{project_id}/kafka/instances/{instance_id}/reassign` | Kafka实例开始分区平衡任务 |
| `CreateKafkaRebalanceLogTask` | POST | `/v2/kafka/{project_id}/instances/{instance_id}/log/rebalance-log` | 开启Kafka实例重平衡日志功能 |
| `CreateKafkaTopicQuota` | POST | `/v2/kafka/{project_id}/instances/{instance_id}/kafka-topic-quota` | 创建Topic流控配置 |
| `CreateKafkaUserClientQuotaTask` | POST | `/v2/kafka/{project_id}/instances/{instance_id}/kafka-user-client-quota` | 创建用户/客户端流控配置 |
| `CreateMessageDiagnosisTask` | POST | `/v2/{project_id}/kafka/instances/{instance_id}/message-diagnosis-tasks` | 创建消息积压诊断任务 |
| `CreatePostPaidKafkaInstance` | POST | `/v2/{project_id}/kafka/instances` | 创建Kafka实例 |
| `DeleteBackgroundTask` | DELETE | `/v2/{project_id}/instances/{instance_id}/tasks/{task_id}` | 删除后台任务管理中的指定记录 |
| `DeleteConnector` | POST | `/v2/{project_id}/kafka/instances/{instance_id}/delete-connector` | 关闭Smart Connect(按需实例) |
| `DeleteConnectorTask` | DELETE | `/v2/{project_id}/instances/{instance_id}/connector/tasks/{task_id}` | 删除Smart Connect任务 |
| `DeleteConsumerGroupOffsets` | POST | `/v2/kafka/{project_id}/instances/{instance_id}/groups/{group}/delete-offset` | 删除消费组在指定Topic的消费位点 |
| `DeleteInstance` | DELETE | `/v2/{project_id}/instances/{instance_id}` | 删除指定的实例 |
| `DeleteInstanceConsumerGroup` | DELETE | `/v2/{engine}/{project_id}/instances/{instance_id}/groups/{group}` | 删除指定消费组 |
| `DeleteKafkaTopicMessages` | POST | `/v2/{project_id}/kafka/instances/{instance_id}/topics/{topic}/messages/delete` | 删除Kafka消息 |
| `DeleteKafkaTopicQuota` | DELETE | `/v2/kafka/{project_id}/instances/{instance_id}/kafka-topic-quota` | 删除Topic流控配置 |
| `DeleteKafkaUserClientQuotaTask` | DELETE | `/v2/kafka/{project_id}/instances/{instance_id}/kafka-user-client-quota` | 删除用户/客户端流控配置 |
| `DeleteScheduledTask` | DELETE | `/v2/{project_id}/instances/{instance_id}/scheduled-tasks/{task_id}` | 删除指定的定时任务 |
| `ListAvailableZones` | GET | `/v2/available-zones` | 查询可用区信息 |
| `ListBackgroundTasks` | GET | `/v2/{project_id}/instances/{instance_id}/tasks` | 查询实例的后台任务列表 |

... and 83 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
