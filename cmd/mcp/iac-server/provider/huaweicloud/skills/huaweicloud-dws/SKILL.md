---
name: huaweicloud-dws
description: HuaweiCloud DWS API guide. 198 APIs covering 主机监控, 事件管理, 任务管理, 加密集群, 升级管理.
---

# HuaweiCloud DWS API Guide

198 APIs. Tags: 主机监控, 事件管理, 任务管理, 加密集群, 升级管理, 参数配置, 可用区, 告警管理, 审计日志, 容灾管理, 快照管理, 数据库权限管理, 数据源, 日志管理, 标签管理, 节点变更, 资源管理, 连接管理, 逻辑集群管理, 配额管理, 集群管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddQueueUserList` | POST | `/v2/{project_id}/clusters/{cluster_id}/workload/queues/{queue_name}/users/batch-create` | 添加资源池的绑定用户 |
| `AddWorkloadPlanStage` | POST | `/v2/{project_id}/clusters/{cluster_id}/workload/plans/{plan_id}/stages` | 添加资源管理计划阶段 |
| `AddWorkloadQueue` | PUT | `/v2/{project_id}/clusters/{cluster_id}/workload/queues` | 添加资源池 |
| `AddWorkloadRule` | POST | `/v1/{project_id}/clusters/{cluster_id}/workload/rules` | 添加异常规则 |
| `AssociateEip` | POST | `/v2/{project_id}/clusters/{cluster_id}/eips/{eip_id}` | 集群绑定EIP |
| `AssociateElb` | POST | `/v2/{project_id}/clusters/{cluster_id}/elbs/{elb_id}` | 集群绑定ELB |
| `BatchCreateClusterCn` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/cns/batch-create` | 批量增加CN节点 |
| `BatchCreateResourceTag` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/tags/batch-create` | 批量添加标签 |
| `BatchDeleteClusterCn` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/cns/batch-delete` | 批量删除CN节点 |
| `BatchDeleteResourceTag` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/tags/batch-delete` | 批量删除标签 |
| `CancelReadonlyCluster` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/cancel-readonly` | 解除只读 |
| `ChangeSecurityGroup` | PUT | `/v1/{project_id}/clusters/{cluster_id}/security-group` | 修改集群安全组 |
| `CheckCluster` | POST | `/v2/{project_id}/cluster-precheck` | 创建集群前检查 |
| `CheckClusterShrink` | GET | `/v1/{project_id}/clusters/{cluster_id}/shrink-check` | 集群缩容前检查 |
| `CheckDisasterName` | GET | `/v2/{project_id}/disaster-recovery/check-name` | 检查容灾名称 |
| `CheckGrowCluster` | POST | `/v2/{project_id}/clusters/{cluster_id}/grow-check` | 集群扩容前检查 |
| `CheckTableRestore` | POST | `/v1/{project_id}/snapshots/{snapshot_id}/table-restore-check` | 用户恢复表名检测 |
| `ConvertToLogicalCluster` | POST | `/v2/{project_id}/clusters/{cluster_id}/convert-to-logical-cluster/{name}` | 物理集群转换到逻辑集群 |
| `CopySnapshot` | POST | `/v1.0/{project_id}/snapshots/{snapshot_id}/linked-copy` | 复制快照 |
| `CreateAlarmSub` | POST | `/v2/{project_id}/alarm-subs` | 创建告警订阅 |
| `CreateCluster` | POST | `/v1.0/{project_id}/clusters` | 创建集群 |
| `CreateClusterDns` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/dns` | 申请域名 |
| `CreateClusterV2` | POST | `/v2/{project_id}/clusters` | 创建集群V2 |
| `CreateClusterWorkload` | POST | `/v2/{project_id}/clusters/{cluster_id}/workload` | 打开或关闭资源管理功能 |
| `CreateDatabaseUser` | POST | `/v1/{project_id}/clusters/{cluster_id}/db-manager/users` | 创建数据库用户/角色 |
| `CreateDataSource` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/ext-data-sources` | 创建数据源 |
| `CreateDisasterRecovery` | POST | `/v2/{project_id}/disaster-recoveries` | 创建容灾 |
| `CreateEventSub` | POST | `/v2/{project_id}/event-subs` | 创建订阅事件 |
| `CreateLogicalCluster` | POST | `/v2/{project_id}/clusters/{cluster_id}/logical-clusters` | 创建逻辑集群 |
| `CreateLogicalClusterPlan` | POST | `/v1/{project_id}/clusters/{cluster_id}/logical-cluster-plans` | 添加逻辑集群定时增删计划 |

... and 168 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
