---
name: huaweicloud-mrs
description: HuaweiCloud MRS API guide. 58 APIs covering IAM同步管理接口, SQL接口, 作业管理接口, 可用区, 委托管理.
---

# HuaweiCloud MRS API Guide

58 APIs. Tags: IAM同步管理接口, SQL接口, 作业管理接口, 可用区, 委托管理, 弹性伸缩接口, 数据连接管理接口, 标签管理接口, 版本元数据查询, 集群HDFS文件接口, 集群管理接口

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddComponent` | POST | `/v2/{project_id}/clusters/{cluster_id}/components` | 集群添加组件 |
| `BatchCreateClusterTags` | POST | `/v1.1/{project_id}/clusters/{cluster_id}/tags/action` | 批量添加集群标签 |
| `BatchDeleteClusterTags` | POST | `/v1.1/{project_id}/clusters/{cluster_id}/tags/action` | 批量删除集群标签 |
| `BatchDeleteJobs` | POST | `/v2/{project_id}/clusters/{cluster_id}/job-executions/batch-delete` | 批量删除作业 |
| `CancelSql` | POST | `/v2/{project_id}/clusters/{cluster_id}/sql-execution/{sql_id}/cancel` | 取消SQL执行任务 |
| `CancelSyncIamUser` | DELETE | `/v2/{project_id}/clusters/{cluster_id}/iam-sync-user` | 指定用户、用户组取消同步 |
| `CreateAutoScalingPolicy` | POST | `/v2/{project_id}/autoscaling-policy/{cluster_id}` | 创建弹性伸缩策略 |
| `CreateCluster` | POST | `/v1.1/{project_id}/run-job-flow` | 创建集群并执行作业 |
| `CreateClusterTag` | POST | `/v1.1/{project_id}/clusters/{cluster_id}/tags` | 给指定集群添加标签 |
| `CreateDataConnector` | POST | `/v2/{project_id}/data-connectors` | 创建数据连接 |
| `CreateExecuteJob` | POST | `/v2/{project_id}/clusters/{cluster_id}/job-executions` | 新增并执行作业 |
| `CreateScalingPolicy` | POST | `/v1.1/{project_id}/autoscaling-policy/{cluster_id}` | 配置弹性伸缩规则 |
| `DeleteAutoScalingPolicy` | DELETE | `/v2/{project_id}/autoscaling-policy/{cluster_id}` | 删除弹性伸缩策略 |
| `DeleteCluster` | DELETE | `/v1.1/{project_id}/clusters/{cluster_id}` | 删除集群 |
| `DeleteClusterTag` | DELETE | `/v1.1/{project_id}/clusters/{cluster_id}/tags/{key}` | 删除指定集群的标签 |
| `DeleteDataConnector` | DELETE | `/v2/{project_id}/data-connectors/{connector_id}` | 删除数据连接 |
| `ExecuteSql` | POST | `/v2/{project_id}/clusters/{cluster_id}/sql-execution` | 提交SQL语句 |
| `ExpandCluster` | POST | `/v2/{project_id}/clusters/{cluster_id}/expand` | 扩容集群 |
| `ListAllTags` | GET | `/v1.1/{project_id}/clusters/tags` | 查询所有标签 |
| `ListAsyncTaskStatus` | GET | `/v1/{project_id}/clusters/{cluster_id}/async_task_status/update_ecs_agency` | 查询指定集群切换委托任务状态 |
| `ListAvailableZones` | GET | `/v1.1/{region_id}/available-zones` | 查询可用区信息 |
| `ListClusterManagerAuthState` | GET | `/v2/{project_id}/clusters/{cluster_id}/manager-auth` | 查询集群界面授权状态 |
| `ListClusters` | GET | `/v1.1/{project_id}/cluster_infos` | 查询集群列表 |
| `ListClustersByTags` | POST | `/v1.1/{project_id}/clusters/resource_instances/action` | 查询特定标签的集群列表 |
| `ListClusterSshState` | GET | `/v1/cluster/{cluster_id}/ssh` | 查询集群节点授权状态 |
| `ListClusterTags` | GET | `/v1.1/{project_id}/clusters/{cluster_id}/tags` | 查询指定集群的标签 |
| `ListDataConnector` | GET | `/v2/{project_id}/data-connectors` | 查询数据连接列表 |
| `ListHosts` | GET | `/v1.1/{project_id}/clusters/{cluster_id}/hosts` | 查询主机列表 |
| `ListNodes` | GET | `/v2/{project_id}/clusters/{cluster_id}/nodes` | 查询集群节点列表 |
| `ListSecurityRuleStatus` | GET | `/v2/{project_id}/clusters/{cluster_id}/security-rule/status` | 获取当前集群通信安全授权状态 |

... and 28 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
