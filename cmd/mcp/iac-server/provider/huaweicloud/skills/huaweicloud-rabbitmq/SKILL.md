---
name: huaweicloud-rabbitmq
description: HuaweiCloud RabbitMQ API guide. 49 APIs covering Binding管理, Exchange管理, Queue管理, Vhost管理, 其他接口.
---

# HuaweiCloud RabbitMQ API Guide

49 APIs. Tags: Binding管理, Exchange管理, Queue管理, Vhost管理, 其他接口, 后台任务管理, 实例管理, 标签管理, 生命周期管理, 用户管理, 规格变更管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateOrDeleteRabbitMqTag` | POST | `/v2/{project_id}/rabbitmq/{instance_id}/tags/action` | 批量添加或删除实例标签 |
| `BatchDeleteExchanges` | POST | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts/{vhost}/exchanges` | 批量删除指定Exchange |
| `BatchDeleteQueues` | POST | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts/{vhost}/queues` | 批量删除指定Queue |
| `BatchDeleteVhosts` | POST | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts` | 批量删除指定Vhost |
| `BatchRestartOrDeleteInstances` | POST | `/v2/{project_id}/instances/action` | 批量删除实例 |
| `CreateBinding` | POST | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts/{vhost}/exchanges/{exchange}/binding` | 添加绑定 |
| `CreateExchange` | PUT | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts/{vhost}/exchanges` | 创建Exchange |
| `CreatePostPaidInstanceByEngine` | POST | `/v2/{engine}/{project_id}/instances` | 创建实例 |
| `CreateQueue` | PUT | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts/{vhost}/queues` | 创建Queue |
| `CreateUser` | POST | `/v2/{project_id}/instances/{instance_id}/users` | 创建用户 |
| `CreateVhost` | PUT | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts` | 创建Vhost |
| `DeleteBackgroundTask` | DELETE | `/v2/{project_id}/instances/{instance_id}/tasks/{task_id}` | 删除后台任务管理中的指定记录 |
| `DeleteBinding` | DELETE | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts/{vhost}/exchanges/{exchange}/destination-type/{destination_type}/destination/{destination}/properties-key/{properties_key}/unbinding` | 删除绑定 |
| `DeleteInstance` | DELETE | `/v2/{project_id}/instances/{instance_id}` | 删除指定实例 |
| `DeleteQueueInfo` | DELETE | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts/{vhost}/queues/{queue}/contents` | 清空Queue消息 |
| `DeleteScheduledTask` | DELETE | `/v2/{project_id}/instances/{instance_id}/scheduled-tasks/{task_id}` | 删除定时任务管理中的指定记录 |
| `DeleteUser` | DELETE | `/v2/{project_id}/instances/{instance_id}/users/{user_name}` | 删除用户 |
| `ListAvailableZones` | GET | `/v2/available-zones` | 查询可用区信息 |
| `ListBackgroundTasks` | GET | `/v2/{project_id}/instances/{instance_id}/tasks` | 查询实例的后台任务列表 |
| `ListBindings` | GET | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts/{vhost}/exchanges/{exchange}/binding` | 查询Exchange绑定信息列表 |
| `ListConfigFeatures` | GET | `/v2/config/features` | 查询特性开关列表 |
| `ListEngineProducts` | GET | `/v2/{engine}/products` | 查询产品规格列表 |
| `ListExchanges` | GET | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts/{vhost}/exchanges` | 查询Exchange列表 |
| `ListInstancesDetails` | GET | `/v2/{project_id}/instances` | 查询所有实例列表 |
| `ListPlugins` | GET | `/v2/{project_id}/instances/{instance_id}/rabbitmq/plugins` | 查询插件列表 |
| `ListQueues` | GET | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts/{vhost}/queues` | 查询所属Vhost下Queue的列表 |
| `ListScheduledTasks` | GET | `/v2/{project_id}/instances/{instance_id}/scheduled-tasks` | 查询实例的定时任务列表 |
| `ListUser` | GET | `/v2/{project_id}/instances/{instance_id}/users` | 查询用户列表 |
| `ListVhosts` | GET | `/v2/rabbitmq/{project_id}/instances/{instance_id}/vhosts` | 查询Vhost列表 |
| `ModifyRecyclePolicy` | PUT | `/v2/{project_id}/recycle` | 更新回收站策略 |

... and 19 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
