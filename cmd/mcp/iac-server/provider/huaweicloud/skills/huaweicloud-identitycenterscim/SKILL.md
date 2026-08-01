---
name: huaweicloud-identitycenterscim
description: HuaweiCloud IdentityCenterSCIM API guide. 12 APIs covering 服务提供商管理, 用户管理, 用户组管理.
---

# HuaweiCloud IdentityCenterSCIM API Guide

12 APIs. Tags: 服务提供商管理, 用户管理, 用户组管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateGroup` | POST | `/{tenant_id}/scim/v2/Groups` | 创建用户组 |
| `CreateUser` | POST | `/{tenant_id}/scim/v2/Users` | 创建用户 |
| `DeleteGroup` | DELETE | `/{tenant_id}/scim/v2/Groups/{group_id}` | 删除用户组 |
| `DeleteUser` | DELETE | `/{tenant_id}/scim/v2/Users/{user_id}` | 删除用户 |
| `GetGroup` | GET | `/{tenant_id}/scim/v2/Groups/{group_id}` | 查询用户组详情 |
| `GetUser` | GET | `/{tenant_id}/scim/v2/Users/{user_id}` | 查询用户详情 |
| `ListGroups` | GET | `/{tenant_id}/scim/v2/Groups` | 列出用户组 |
| `ListUsers` | GET | `/{tenant_id}/scim/v2/Users` | 列出用户 |
| `PatchGroup` | PATCH | `/{tenant_id}/scim/v2/Groups/{group_id}` | 部分更新用户组 |
| `PatchUser` | PATCH | `/{tenant_id}/scim/v2/Users/{user_id}` | 部分更新用户 |
| `PutUser` | PUT | `/{tenant_id}/scim/v2/Users/{user_id}` | 更新用户 |
| `ServiceProviderConfig` | GET | `/{tenant_id}/scim/v2/ServiceProviderConfig` | 查询服务提供商配置 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
