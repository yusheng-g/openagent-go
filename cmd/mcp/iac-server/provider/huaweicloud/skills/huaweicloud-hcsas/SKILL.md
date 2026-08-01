---
name: huaweicloud-hcsas
description: HuaweiCloud HCSAS API guide. 37 APIs covering 伸缩活动日志管理, 弹性伸缩实例, 弹性伸缩策略, 弹性伸缩组, 弹性伸缩配置.
---

# HuaweiCloud HCSAS API Guide

37 APIs. Tags: 伸缩活动日志管理, 弹性伸缩实例, 弹性伸缩策略, 弹性伸缩组, 弹性伸缩配置, 标签管理, 生命周期挂钩管理, 通知管理, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchDeleteConfig` | POST | `/autoscaling-api/v1/{tenant_id}/scaling_configurations` | 批量删除弹性伸缩配置 |
| `BatchResumeInstances` | POST | `/autoscaling-api/v1/{tenant_id}/scaling_group_instance/{scaling_group_id}/action` | 批量操作实例 |
| `ChangeScalingGroup` | PUT | `/autoscaling-api/v1/{tenant_id}/scaling_group/{scaling_group_id}` | 修改弹性伸缩组 |
| `ChangeScalingTagInfo` | POST | `/autoscaling-api/v1/{tenant_id}/{resource_type}/{resource_id}/tags/action` | 创建、更新或删除标签 |
| `CreateLifecycleHook` | POST | `/autoscaling-api/v1/{tenant_id}/scaling_lifecycle_hook/{scaling_group_id}` | 创建生命周期挂钩 |
| `CreatePolicy` | POST | `/autoscaling-api/v1/{tenant_id}/scaling_policy` | 创建弹性伸缩策略 |
| `CreateScalingConfig` | POST | `/autoscaling-api/v1/{tenant_id}/scaling_configuration` | 创建弹性伸缩配置 |
| `CreateScalingGroup` | POST | `/autoscaling-api/v1/{tenant_id}/scaling_group` | 创建弹性伸缩组 |
| `DeleteConfig` | DELETE | `/autoscaling-api/v1/{tenant_id}/scaling_configuration/{scaling_configuration_id}` | 删除弹性伸缩配置 |
| `DeleteInstance` | DELETE | `/autoscaling-api/v1/{tenant_id}/scaling_group_instance/{instance_id}` | 移出弹性伸缩组实例 |
| `DeleteLifecycleHook` | DELETE | `/autoscaling-api/v1/{tenant_id}/scaling_lifecycle_hook/{scaling_group_id}/{lifecycle_hook_name}` | 删除生命周期挂钩 |
| `DeletePolicy` | DELETE | `/autoscaling-api/v1/{tenant_id}/scaling_policy/{scaling_policy_id}` | 删除弹性伸缩策略 |
| `DeleteScalingGroup` | DELETE | `/autoscaling-api/v1/{tenant_id}/scaling_group/{scaling_group_id}` | 删除弹性伸缩组 |
| `DeleteScalingNotification` | DELETE | `/autoscaling-api/v1/{tenant_id}/scaling_notification/{scaling_group_id}/{topic_urn}` | 删除伸缩组通知 |
| `InvokeInstanceLifecycleHook` | PUT | `/autoscaling-api/v1/{tenant_id}/scaling_instance_hook/{scaling_group_id}/callback` | 伸缩实例生命周期回调 |
| `ListLifecycleHook` | GET | `/autoscaling-api/v1/{tenant_id}/scaling_lifecycle_hook/{scaling_group_id}/list` | 查询生命周期挂钩列表 |
| `ListPolicy` | GET | `/autoscaling-api/v1/{tenant_id}/scaling_policy/{scaling_group_id}/list` | 查询弹性伸缩策略列表 |
| `ListPolicyInstance` | GET | `/autoscaling-api/v1/{tenant_id}/quotas/{scaling_group_id}` | 查询弹性伸缩策略和伸缩实例配额 |
| `ListQuotas` | GET | `/autoscaling-api/v1/{tenant_id}/quotas` | 查询配额 |
| `ListScalingActivityLog` | GET | `/autoscaling-api/v1/{tenant_id}/scaling_activity_log/{scaling_group_id}` | 查询伸缩活动日志 |
| `ListScalingConfig` | GET | `/autoscaling-api/v1/{tenant_id}/scaling_configuration` | 查询弹性伸缩配置列表 |
| `ListScalingGroup` | GET | `/autoscaling-api/v1/{tenant_id}/scaling_group` | 查询弹性伸缩组列表 |
| `ListScalingGroupInstance` | GET | `/autoscaling-api/v1/{tenant_id}/scaling_group_instance/{scaling_group_id}/list` | 查询弹性伸缩组中的实例列表 |
| `ListScalingNotification` | GET | `/autoscaling-api/v1/{tenant_id}/scaling_notification/{scaling_group_id}` | 查询伸缩组通知列表 |
| `ModifyLifecycleHook` | PUT | `/autoscaling-api/v1/{tenant_id}/scaling_lifecycle_hook/{scaling_group_id}/{lifecycle_hook_name}` | 修改生命周期挂钩 |
| `ModifyPolicy` | PUT | `/autoscaling-api/v1/{tenant_id}/scaling_policy/{scaling_policy_id}` | 修改弹性伸缩策略 |
| `ResumePolicy` | POST | `/autoscaling-api/v1/{tenant_id}/scaling_policy/{scaling_policy_id}/action` | 执行或启用或停止弹性伸缩策略。 |
| `ResumeScalingGroup` | POST | `/autoscaling-api/v1/{tenant_id}/scaling_group/{scaling_group_id}/action` | 启用或停止弹性伸缩组 |
| `SetScalingNotification` | PUT | `/autoscaling-api/v1/{tenant_id}/scaling_notification/{scaling_group_id}` | 配置伸缩组通知 |
| `ShowInstance` | POST | `/autoscaling-api/v1/{tenant_id}/{resource_type}/resource_instances/action` | 查询资源实例 |

... and 7 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
