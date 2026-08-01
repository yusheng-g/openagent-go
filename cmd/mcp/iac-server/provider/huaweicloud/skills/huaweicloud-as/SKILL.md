---
name: huaweicloud-as
description: HuaweiCloud AS API guide. 71 APIs covering 伸缩活动日志管理, 伸缩策略日志管理, 弹性伸缩API管理, 弹性伸缩实例管理, 弹性伸缩策略管理. Detailed swagger definitions in references/<APIName>.json.
---

# HuaweiCloud AS API Guide

71 APIs. Tags: 伸缩活动日志管理, 伸缩策略日志管理, 弹性伸缩API管理, 弹性伸缩实例管理, 弹性伸缩策略管理, 弹性伸缩策略管理V2, 弹性伸缩组管理, 弹性伸缩配置, 标签管理, 生命周期挂钩管理, 计划任务, 通知管理, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AttachCallbackInstanceLifeCycleHook` | PUT | `/autoscaling-api/v1/{project_id}/scaling_instance_hook/{scaling_group_id}/callback` | 伸缩实例生命周期回调 |
| `BatchAddScalingInstances` | POST | `/autoscaling-api/v1/{project_id}/scaling_group_instance/{scaling_group_id}/action` | 批量添加实例 |
| `BatchDeleteScalingConfigs` | POST | `/autoscaling-api/v1/{project_id}/scaling_configurations` | 批量删除弹性伸缩配置 |
| `BatchDeleteScalingPolicies` | POST | `/autoscaling-api/v1/{project_id}/scaling_policies/action` | 批量删除弹性伸缩策略。 |
| `BatchPauseScalingPolicies` | POST | `/autoscaling-api/v1/{project_id}/scaling_policies/action` | 批量停用弹性伸缩策略。 |
| `BatchProtectScalingInstances` | POST | `/autoscaling-api/v1/{project_id}/scaling_group_instance/{scaling_group_id}/action` | 批量设置实例保护 |
| `BatchRemoveScalingInstances` | POST | `/autoscaling-api/v1/{project_id}/scaling_group_instance/{scaling_group_id}/action` | 批量移除实例 |
| `BatchResumeScalingPolicies` | POST | `/autoscaling-api/v1/{project_id}/scaling_policies/action` | 批量启用弹性伸缩策略。 |
| `BatchSetScalingInstancesStandby` | POST | `/autoscaling-api/v1/{project_id}/scaling_group_instance/{scaling_group_id}/action` | 批量将实例转为备用状态 |
| `BatchUnprotectScalingInstances` | POST | `/autoscaling-api/v1/{project_id}/scaling_group_instance/{scaling_group_id}/action` | 批量取消实例保护 |
| `BatchUnsetScalingInstancesStantby` | POST | `/autoscaling-api/v1/{project_id}/scaling_group_instance/{scaling_group_id}/action` | 批量将实例移出备用状态 |
| `CloseWarmPool` | DELETE | `/autoscaling-api/{project_id}/scaling-groups/{scaling_group_id}/warm-pool` | 关闭暖池 |
| `CloseWarmPoolNew` | DELETE | `/v2/{project_id}/scaling-groups/{scaling_group_id}/warm-pool` | 关闭暖池(V2版本) |
| `CreateGroupScheduledTask` | POST | `/autoscaling-api/v1/{project_id}/scaling-groups/{scaling_group_id}/scheduled-tasks` | 创建计划任务 |
| `CreateLifyCycleHook` | POST | `/autoscaling-api/v1/{project_id}/scaling_lifecycle_hook/{scaling_group_id}` | 创建生命周期挂钩 |
| `CreateScalingConfig` | POST | `/autoscaling-api/v1/{project_id}/scaling_configuration` | 创建弹性伸缩配置 |
| `CreateScalingGroup` | POST | `/autoscaling-api/v1/{project_id}/scaling_group` | 创建弹性伸缩组 |
| `CreateScalingNotification` | PUT | `/autoscaling-api/v1/{project_id}/scaling_notification/{scaling_group_id}` | 配置伸缩组通知 |
| `CreateScalingPolicy` | POST | `/autoscaling-api/v1/{project_id}/scaling_policy` | 创建弹性伸缩策略 |
| `CreateScalingTagInfo` | POST | `/autoscaling-api/v1/{project_id}/{resource_type}/{resource_id}/tags/action` | 创建标签 |
| `CreateScalingV2Policy` | POST | `/autoscaling-api/v2/{project_id}/scaling_policy` | 创建弹性伸缩策略(V2版本) |
| `DeleteGroupScheduledTask` | DELETE | `/autoscaling-api/v1/{project_id}/scaling-groups/{scaling_group_id}/scheduled-tasks/{scheduled_task_id}` | 删除计划任务 |
| `DeleteLifecycleHook` | DELETE | `/autoscaling-api/v1/{project_id}/scaling_lifecycle_hook/{scaling_group_id}/{lifecycle_hook_name}` | 删除生命周期挂钩 |
| `DeleteScalingConfig` | DELETE | `/autoscaling-api/v1/{project_id}/scaling_configuration/{scaling_configuration_id}` | 删除弹性伸缩配置 |
| `DeleteScalingGroup` | DELETE | `/autoscaling-api/v1/{project_id}/scaling_group/{scaling_group_id}` | 删除弹性伸缩组 |
| `DeleteScalingInstance` | DELETE | `/autoscaling-api/v1/{project_id}/scaling_group_instance/{instance_id}` | 移出弹性伸缩组实例 |
| `DeleteScalingNotification` | DELETE | `/autoscaling-api/v1/{project_id}/scaling_notification/{scaling_group_id}/{topic_urn}` | 删除伸缩组通知 |
| `DeleteScalingPolicy` | DELETE | `/autoscaling-api/v1/{project_id}/scaling_policy/{scaling_policy_id}` | 删除弹性伸缩策略 |
| `DeleteScalingTagInfo` | POST | `/autoscaling-api/v1/{project_id}/{resource_type}/{resource_id}/tags/action` | 删除标签 |
| `ExecuteScalingPolicy` | POST | `/autoscaling-api/v1/{project_id}/scaling_policy/{scaling_policy_id}/action` | 执行弹性伸缩策略。 |

... and 41 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
