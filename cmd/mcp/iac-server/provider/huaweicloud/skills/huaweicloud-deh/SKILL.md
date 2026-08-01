---
name: huaweicloud-deh
description: HuaweiCloud DeH API guide. 16 APIs covering 查询API版本信息, 标签管理, 生命周期管理, 配额设置.
---

# HuaweiCloud DeH API Guide

16 APIs. Tags: 查询API版本信息, 标签管理, 生命周期管理, 配额设置

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateDedicatedHostTags` | POST | `/v1.0/{project_id}/dedicated-host-tags/{dedicated_host_id}/tags/action` | 批量添加专属主机标签 |
| `BatchDeleteDedicatedHostTags` | POST | `/v1.0/{project_id}/dedicated-host-tags/{dedicated_host_id}/tags/action` | 批量删除专属主机标签 |
| `CreateDedicatedHost` | POST | `/v1.0/{project_id}/dedicated-hosts` | 分配专属主机 |
| `DeleteDedicatedHost` | DELETE | `/v1.0/{project_id}/dedicated-hosts/{dedicated_host_id}` | 释放专属主机 |
| `ListDedicatedHostAllTypes` | GET | `/v1.0/{project_id}/dedicated-host-types` | 查询专属主机类型列表 |
| `ListDedicatedHosts` | GET | `/v1.0/{project_id}/dedicated-hosts` | 查询专属主机列表 |
| `ListDedicatedHostsByTags` | POST | `/v1.0/{project_id}/dedicated-host-tags/resource_instances/action` | 按标签查询专属主机列表 |
| `ListDedicatedHostTags` | GET | `/v1.0/{project_id}/dedicated-host-tags/tags` | 查询所有专属主机标签 |
| `ListDedicatedHostTypes` | GET | `/v1.0/{project_id}/availability-zone/{availability_zone}/dedicated-host-types` | 查询可用的专属主机类型 |
| `ListDehVersions` | GET | `/` | 查询API版本信息列表 |
| `ListServersDedicatedHost` | GET | `/v1.0/{project_id}/dedicated-hosts/{dedicated_host_id}/servers` | 查询专属主机上的云服务器 |
| `ShowDedicatedHost` | GET | `/v1.0/{project_id}/dedicated-hosts/{dedicated_host_id}` | 查询专属主机详情 |
| `ShowDedicatedHostTags` | GET | `/v1.0/{project_id}/dedicated-host-tags/{dedicated_host_id}/tags` | 查询指定专属主机标签 |
| `ShowDehVersion` | GET | `/{api_version}` | 查询指定API版本信息 |
| `ShowQuotaSets` | GET | `/v1.0/{project_id}/quota-sets/{tenant_id}` | 查询租户的专属主机配额 |
| `UpdateDedicatedHost` | PUT | `/v1.0/{project_id}/dedicated-hosts/{dedicated_host_id}` | 更新专属主机属性 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
