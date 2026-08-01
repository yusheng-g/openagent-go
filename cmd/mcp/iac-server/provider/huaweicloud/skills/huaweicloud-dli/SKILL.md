---
name: huaweicloud-dli
description: HuaweiCloud DLI API guide. 160 APIs covering Flink作业-作业相关API, Flink作业-模板相关API, Flink作业-管理相关API, SQL作业-作业相关API, SQL作业-拦截规则相关API.
---

# HuaweiCloud DLI API Guide

160 APIs. Tags: Flink作业-作业相关API, Flink作业-模板相关API, Flink作业-管理相关API, SQL作业-作业相关API, SQL作业-拦截规则相关API, SQL作业-模板相关API, Spark作业-作业相关API, Spark作业-模板相关API, 全局变量相关API, 动态脱敏策略相关API, 增强型跨源连接相关API, 已弃用-Flink作业-作业相关API, 已弃用-SQL作业相关API, 已弃用-Spark作业-作业相关API, 已弃用-分组资源相关API, 已弃用-委托与权限相关API, 已弃用-跨源连接相关API, 已弃用-队列相关API, 弹性资源池相关API, 数据目录相关API, 权限相关API, 资源标签相关API, 配额相关API, 队列相关API

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AssociateQueueToElasticResourcePool` | POST | `/v3/{project_id}/elastic-resource-pools/{elastic_resource_pool_name}/queues` | 关联队列到弹性资源池 |
| `AssociateQueueToEnhancedConnection` | POST | `/v2.0/{project_id}/datasource/enhanced-connections/{connection_id}/associate-queue` | 绑定队列 |
| `BatchCreateResourceTags` | POST | `/v3/{project_id}/{resource_type}/{resource_id}/tags/create` | 批量添加资源标签 |
| `BatchDeleteFlinkJobs` | POST | `/v1.0/{project_id}/streaming/jobs/delete` | 批量删除Flink作业 |
| `BatchDeleteQueuePlans` | POST | `/v1/{project_id}/queues/{queue_name}/plans/batch-delete` | 批量删除队列定时扩缩容计划 |
| `BatchDeleteResourceTags` | POST | `/v3/{project_id}/{resource_type}/{resource_id}/tags/delete` | 批量删除资源标签 |
| `BatchDeleteSqlJobTemplates` | POST | `/v1.0/{project_id}/sqls-deletion` | 批量删除SQL模板 |
| `BatchRunFlinkJobs` | POST | `/v1.0/{project_id}/streaming/jobs/run` | 批量运行Flink作业 |
| `BatchStopFlinkJobs` | POST | `/v1.0/{project_id}/streaming/jobs/stop` | 批量停止Flink作业 |
| `CancelSparkJob` | DELETE | `/v2.0/{project_id}/batches/{batch_id}` | 取消批处理作业 |
| `CancelSqlJob` | DELETE | `/v1.0/{project_id}/jobs/{job_id}` | 取消作业 |
| `CheckSql` | POST | `/v1.0/{project_id}/jobs/check-sql` | 检查SQL语法 |
| `CountResourcesByTags` | POST | `/v3/{project_id}/{resource_type}/resource-instances/count` | 查询资源实例数量 |
| `CreateAuthInfo` | POST | `/v2.0/{project_id}/datasource/auth-infos` | 创建跨源认证 |
| `CreateConnectivityTask` | POST | `/v1.0/{project_id}/queues/{queue_name}/connection-test` | 创建地址连通性请求 |
| `CreateDatabase` | POST | `/v1.0/{project_id}/databases` | 创建数据库 |
| `CreateDataMaskStrategy` | POST | `/v1/{project_id}/data-mask-strategy` | 创建动态脱敏策略 |
| `CreateDataMaskStrategyUserAuth` | PUT | `/v1/{project_id}/data-mask-strategy/user-authorization` | 动态脱敏策略授权 |
| `CreateDatasourceConnection` | POST | `/v2.0/{project_id}/datasource-connection` | 创建经典型跨源连接 |
| `CreateDliAgency` | POST | `/v2/{project_id}/agency` | 创建DLI委托 |
| `CreateElasticResourcePool` | POST | `/v3/{project_id}/elastic-resource-pools` | 创建弹性资源池 |
| `CreateEnhancedConnection` | POST | `/v2.0/{project_id}/datasource/enhanced-connections` | 创建增强型跨源连接 |
| `CreateEnhancedConnectionRoutes` | POST | `/v2.0/{project_id}/datasource/enhanced-connections/{connection_id}/routes` | 创建路由 |
| `CreateFlinkJarJob` | POST | `/v1.0/{project_id}/streaming/flink-jobs` | 新建Flink Jar作业 |
| `CreateFlinkSqlJob` | POST | `/v1.0/{project_id}/streaming/sql-jobs` | 新建Flink SQL作业 |
| `CreateFlinkSqlJobGraph` | POST | `/v3/{project_id}/streaming/jobs/{job_id}/gen-graph` | 生成flink SQL作业的静态流图 |
| `CreateFlinkSqlJobTemplate` | POST | `/v1.0/{project_id}/streaming/job-templates` | 新建Flink作业模板 |
| `CreateGlobalVariable` | POST | `/v1.0/{project_id}/variables` | 创建DLI全局变量 |
| `CreateIefMessageChannel` | POST | `/v1/{project_id}/edgesrv/message-channel` | 创建IEF消息通道 |
| `CreateIefSystemEvents` | POST | `/v1/{project_id}/edgesrv/system-events` | IEF系统事件上报 |

... and 130 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
