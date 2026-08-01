---
name: huaweicloud-sdrs
description: HuaweiCloud SDRS API guide. 49 APIs covering API版本信息, Job管理, 任务中心, 保护实例, 保护组.
---

# HuaweiCloud SDRS API Guide

49 APIs. Tags: API版本信息, Job管理, 任务中心, 保护实例, 保护组, 复制对, 大屏管理, 容灾演练, 查询双活域, 标签管理, 租户配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddProtectedInstanceNic` | POST | `/v1/{project_id}/protected-instances/{protected_instance_id}/nic` | 保护实例添加网卡 |
| `AddProtectedInstanceTags` | POST | `/v1/{project_id}/protected-instances/{protected_instance_id}/tags` | 添加保护实例标签 |
| `AttachProtectedInstanceReplication` | POST | `/v1/{project_id}/protected-instances/{protected_instance_id}/attachreplication` | 保护实例挂载复制对 |
| `BatchAddTags` | POST | `/v1/{project_id}/protected-instances/{protected_instance_id}/tags/action` | 批量添加保护实例标签 |
| `BatchCreateProtectedInstances` | POST | `/v1/{project_id}/protected-instances/batch` | 批量创建保护实例 |
| `BatchDeleteProtectedInstances` | POST | `/v1/{project_id}/protected-instances/delete` | 批量删除保护实例 |
| `BatchDeleteTags` | POST | `/v1/{project_id}/protected-instances/{protected_instance_id}/tags/action` | 批量删除保护实例标签 |
| `CreateDisasterRecoveryDrill` | POST | `/v1/{project_id}/disaster-recovery-drills` | 创建容灾演练 |
| `CreateProtectedInstance` | POST | `/v1/{project_id}/protected-instances` | 创建保护实例 |
| `CreateProtectionGroup` | POST | `/v1/{project_id}/server-groups` | 创建保护组 |
| `CreateReplication` | POST | `/v1/{project_id}/replications` | 创建复制对 |
| `DeleteAllServerGroupFailureJobs` | DELETE | `/v1/{project_id}/task-center/failure-jobs/batch` | 删除所有保护组失败任务 |
| `DeleteDisasterRecoveryDrill` | DELETE | `/v1/{project_id}/disaster-recovery-drills/{disaster_recovery_drill_id}` | 删除容灾演练 |
| `DeleteFailureJob` | DELETE | `/v1/{project_id}/task-center/failure-jobs/{failure_job_id}` | 删除单个失败任务 |
| `DeleteProtectedInstance` | DELETE | `/v1/{project_id}/protected-instances/{protected_instance_id}` | 删除保护实例 |
| `DeleteProtectedInstanceNic` | POST | `/v1/{project_id}/protected-instances/{protected_instance_id}/nic/delete` | 保护实例删除网卡 |
| `DeleteProtectedInstanceTag` | DELETE | `/v1/{project_id}/protected-instances/{protected_instance_id}/tags/{key}` | 删除保护实例标签 |
| `DeleteProtectionGroup` | DELETE | `/v1/{project_id}/server-groups/{server_group_id}` | 删除保护组 |
| `DeleteReplication` | DELETE | `/v1/{project_id}/replications/{replication_id}` | 删除复制对 |
| `DeleteServerGroupFailureJobs` | DELETE | `/v1/{project_id}/task-center/{server_group_id}/failure-jobs/batch` | 删除指定保护组内的所有失败任务 |
| `DetachProtectedInstanceReplication` | DELETE | `/v1/{project_id}/protected-instances/{protected_instance_id}/detachreplication/{replication_id}` | 保护实例卸载复制对 |
| `ExpandReplication` | POST | `/v1/{project_id}/replications/{replication_id}/action` | 复制对扩容 |
| `ListActiveActiveDomains` | GET | `/v1/{project_id}/active-domains` | 查询双活域 |
| `ListApiVersions` | GET | `/` | 查询API版本信息 |
| `ListDisasterRecoveryDrills` | GET | `/v1/{project_id}/disaster-recovery-drills` | 查询容灾演练列表 |
| `ListFailureJobs` | GET | `/v1/{project_id}/task-center/failure-jobs` | 查询失败任务列表 |
| `ListProtectedInstances` | GET | `/v1/{project_id}/protected-instances` | 查询保护实例列表 |
| `ListProtectedInstancesByTags` | POST | `/v1/{project_id}/protected-instances/resource_instances/action` | 通过标签查询保护实例 |
| `ListProtectedInstancesProjectTags` | GET | `/v1/{project_id}/protected-instances/tags` | 查询保护实例项目标签 |
| `ListProtectedInstanceTags` | GET | `/v1/{project_id}/protected-instances/{protected_instance_id}/tags` | 查询保护实例标签 |

... and 19 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
