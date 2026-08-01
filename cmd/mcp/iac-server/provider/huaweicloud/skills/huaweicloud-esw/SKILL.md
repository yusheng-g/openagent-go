---
name: huaweicloud-esw
description: HuaweiCloud ESW API guide. 17 APIs covering 二层连接, 企业交换机.
---

# HuaweiCloud ESW API Guide

17 APIs. Tags: 二层连接, 企业交换机

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BindVport` | POST | `/v3/{project_id}/l2cg/connections/{connection_id}/vports/bind` | 将一个虚拟IP绑定到二层连接上 |
| `CreateConnection` | POST | `/v3/{project_id}/l2cg/instances/{instance_id}/connections` | 创建二层连接 |
| `CreateInstance` | POST | `/v3/{project_id}/l2cg/instances` | 创建ESW实例 |
| `DeleteConnection` | DELETE | `/v3/{project_id}/l2cg/instances/{instance_id}/connections/{connection_id}` | 删除二层连接 |
| `DeleteInstance` | DELETE | `/v3/{project_id}/l2cg/instances/{instance_id}` | 删除ESW实例 |
| `ListAvailabilityZones` | GET | `/v3/{project_id}/l2cg/availability-zones` | 查询ESW实例可用区 |
| `ListConnections` | GET | `/v3/{project_id}/l2cg/instances/{instance_id}/connections` | 查询实例下的二层连接列表 |
| `ListConnectionsAllInstances` | GET | `/v3/{project_id}/l2cg/connections` | 查询二层连接列表 |
| `ListFlavors` | GET | `/v3/{project_id}/l2cg/flavors` | 查询ESW实例规格列表 |
| `ListInstances` | GET | `/v3/{project_id}/l2cg/instances` | 查询ESW实例列表 |
| `ListQuotas` | GET | `/v3/{project_id}/l2cg/quotas` | 查询ESW实例配额 |
| `ListResourceJobs` | GET | `/v3/{project_id}/l2cg/resources/{resource_id}/jobs` | 查询任务的执行状态 |
| `ShowConnection` | GET | `/v3/{project_id}/l2cg/instances/{instance_id}/connections/{connection_id}` | 查询二层连接详情 |
| `ShowInstance` | GET | `/v3/{project_id}/l2cg/instances/{instance_id}` | 查询ESW实例详情 |
| `UnbindVport` | POST | `/v3/{project_id}/l2cg/connections/{connection_id}/vports/unbind` | 将一个虚拟IP从二层连接解绑 |
| `UpdateConnection` | PUT | `/v3/{project_id}/l2cg/instances/{instance_id}/connections/{connection_id}` | 更新二层连接 |
| `UpdateInstance` | PUT | `/v3/{project_id}/l2cg/instances/{instance_id}` | 更新ESW实例 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
