---
name: huaweicloud-rocketmq
description: HuaweiCloud RocketMQ API guide. 70 APIs covering Topic管理, 元数据迁移, 其他接口, 参数管理, 后台任务管理.
---

# HuaweiCloud RocketMQ API Guide

70 APIs. Tags: Topic管理, 元数据迁移, 其他接口, 参数管理, 后台任务管理, 实例管理, 实例诊断, 标签管理, 消息管理, 消费组管理, 生命周期管理, 用户管理, 规格变更管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateOrDeleteRocketmqTag` | POST | `/v2/{project_id}/rocketmq/{instance_id}/tags/action` | 批量添加或删除实例标签 |
| `BatchDeleteDiagnosisRecords` | POST | `/v2/{project_id}/{engine}/instances/{instance_id}/diagnosis/batch-delete` | 批量删除实例诊断报告 |
| `BatchDeleteInstances` | POST | `/v2/{project_id}/instances/action` | 批量删除实例 |
| `BatchDeleteRocketMqMigrationTask` | POST | `/v2/{project_id}/instances/{instance_id}/metadata/batch-delete` | 批量删除元数据迁移任务 |
| `BatchUpdateConsumerGroup` | PUT | `/v2/{project_id}/instances/{instance_id}/groups` | 批量修改消费组 |
| `CreateConsumerGroupOrBatchDeleteConsumerGroup` | POST | `/v2/{project_id}/instances/{instance_id}/groups` | 创建消费组或批量删除消费组 |
| `CreateDiagnosisTask` | POST | `/v2/{engine}/{project_id}/instances/{instance_id}/diagnosis` | 创建实例诊断任务 |
| `CreateInstanceByEngine` | POST | `/v2/{engine}/{project_id}/instances` | 创建实例 |
| `CreateRocketMqMigrationTask` | POST | `/v2/{project_id}/instances/{instance_id}/metadata` | 新建元数据迁移任务 |
| `CreateTopicOrBatchDeleteTopic` | POST | `/v2/{project_id}/instances/{instance_id}/topics` | 创建主题或批量删除主题 |
| `CreateUser` | POST | `/v2/{project_id}/instances/{instance_id}/users` | 创建用户 |
| `DeleteBackgroundTask` | DELETE | `/v2/{project_id}/instances/{instance_id}/tasks/{task_id}` | 删除后台任务管理中的指定记录 |
| `DeleteConsumerGroup` | DELETE | `/v2/{project_id}/instances/{instance_id}/groups/{group}` | 删除指定消费组 |
| `DeleteInstance` | DELETE | `/v2/{project_id}/instances/{instance_id}` | 删除指定的实例 |
| `DeleteScheduledTask` | DELETE | `/v2/{project_id}/instances/{instance_id}/scheduled-tasks/{task_id}` | 删除定时任务管理中的指定记录 |
| `DeleteTopic` | DELETE | `/v2/{project_id}/instances/{instance_id}/topics/{topic}` | 删除指定主题 |
| `DeleteUser` | DELETE | `/v2/{project_id}/instances/{instance_id}/users/{user_name}` | 删除用户 |
| `ListAvailableZones` | GET | `/v2/available-zones` | 查询可用区信息 |
| `ListBackgroundTasks` | GET | `/v2/{project_id}/instances/{instance_id}/tasks` | 查询实例的后台任务列表 |
| `ListBrokers` | GET | `/v2/{project_id}/instances/{instance_id}/brokers` | 查询代理列表 |
| `ListConfigFeatures` | GET | `/v2/config/features` | 获取特性开关列表 |
| `ListConsumeGroupAccessPolicy` | GET | `/v2/{engine}/{project_id}/instances/{instance_id}/groups/{group}/accesspolicy` | 查询消费组的授权用户列表 |
| `ListConsumerGroupOfTopic` | GET | `/v2/{project_id}/instances/{instance_id}/topics/{topic}/groups` | 查询主题消费组列表 |
| `ListDiagnosisReports` | GET | `/v2/{engine}/{project_id}/instances/{instance_id}/diagnosis` | 查询实例诊断报告列表 |
| `ListEngineProducts` | GET | `/v2/{engine}/products` | 查询产品规格列表 |
| `ListInstanceConsumerGroups` | GET | `/v2/{project_id}/instances/{instance_id}/groups` | 查询消费组列表 |
| `ListInstances` | GET | `/v2/{project_id}/instances` | 查询所有实例列表 |
| `ListMessages` | GET | `/v2/{engine}/{project_id}/instances/{instance_id}/messages` | 查询消息 |
| `ListMessageTrace` | GET | `/v2/{engine}/{project_id}/instances/{instance_id}/trace` | 查询消息轨迹 |
| `ListRocketInstanceTopics` | GET | `/v2/{project_id}/instances/{instance_id}/topics` | 查询主题列表 |

... and 40 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
