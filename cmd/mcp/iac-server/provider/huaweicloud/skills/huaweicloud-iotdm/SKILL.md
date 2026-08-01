---
name: huaweicloud-iotdm
description: HuaweiCloud IoTDM API guide. 13 APIs covering 实例任务管理, 实例管理, 实例规格管理.
---

# HuaweiCloud IoTDM API Guide

13 APIs. Tags: 实例任务管理, 实例管理, 实例规格管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BindInstanceTags` | POST | `/v5/iot/{project_id}/iotda-instances/{instance_id}/bind-tags` | 添加实例标签 |
| `ChangeInstanceChargeMode` | POST | `/v5/iot/{project_id}/iotda-instances/{instance_id}/change-charge-mode` | 修改实例计费模式 |
| `CreateInstance` | POST | `/v5/iot/{project_id}/iotda-instances` | 创建设备接入实例 |
| `DeleteInstance` | DELETE | `/v5/iot/{project_id}/iotda-instances/{instance_id}` | 删除实例 |
| `ListInstanceFlavors` | GET | `/v5/iot/{project_id}/iotda-instances/flavors` | 查询实例规格列表 |
| `ListInstances` | GET | `/v5/iot/{project_id}/iotda-instances` | 查询实例列表 |
| `ListInstanceTasks` | GET | `/v5/iot/{project_id}/iotda-instances/{instance_id}/tasks` | 查询实例任务列表 |
| `ResizeInstance` | POST | `/v5/iot/{project_id}/iotda-instances/{instance_id}/resize` | 修改实例规格信息 |
| `RetryInstanceTask` | POST | `/v5/iot/{project_id}/iotda-instances/{instance_id}/tasks/{task_id}/retry` | 重试实例任务 |
| `ShowInstance` | GET | `/v5/iot/{project_id}/iotda-instances/{instance_id}` | 查询实例详情 |
| `ShowInstanceTask` | GET | `/v5/iot/{project_id}/iotda-instances/{instance_id}/tasks/{task_id}` | 查询实例任务详情 |
| `UnbindInstanceTags` | POST | `/v5/iot/{project_id}/iotda-instances/{instance_id}/unbind-tags` | 删除实例标签 |
| `UpdateInstance` | PUT | `/v5/iot/{project_id}/iotda-instances/{instance_id}` | 修改实例信息 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
