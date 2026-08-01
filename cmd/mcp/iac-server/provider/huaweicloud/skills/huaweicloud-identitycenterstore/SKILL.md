---
name: huaweicloud-identitycenterstore
description: HuaweiCloud IdentityCenterStore API guide. 54 APIs covering 服务提供商管理, 用户用户组绑定关系管理, 用户管理, 用户组管理, 自动预置管理.
---

# HuaweiCloud IdentityCenterStore API Guide

54 APIs. Tags: 服务提供商管理, 用户用户组绑定关系管理, 用户管理, 用户组管理, 自动预置管理, 自定义密码策略管理, 身份提供商管理, 身份源配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchDeleteSessions` | POST | `/v1/identity-stores/{identity_store_id}/users/{user_id}/sessions/batch-delete` | 批量删除用户登录会话 |
| `BatchListMfaDevicesForUser` | POST | `/v1/identity-stores/{identity_store_id}/users/retrieve-mfa-devices` | 列出用户MFA设备 |
| `CreateBearerToken` | POST | `/v1/identity-stores/{identity_store_id}/tenant/{tenant_id}/bearer-token` | 创建访问令牌 |
| `CreateExternalIdPConfigurationForDirectory` | POST | `/v1/identity-stores/{identity_store_id}/external-idp` | 创建外部身份提供商配置 |
| `CreateGroup` | POST | `/v1/identity-stores/{identity_store_id}/groups` | 创建用户组 |
| `CreateGroupMembership` | POST | `/v1/identity-stores/{identity_store_id}/group-memberships` | 绑定用户和组 |
| `CreateProvisioningTenant` | POST | `/v1/identity-stores/{identity_store_id}/provision-tenant` | 启用自动预置 |
| `CreateSpCertificate` | POST | `/v1/identity-stores/{identity_store_id}/saml-certificates` | 创建服务提供商证书 |
| `CreateUser` | POST | `/v1/identity-stores/{identity_store_id}/users` | 创建用户 |
| `DeleteBearerToken` | DELETE | `/v1/identity-stores/{identity_store_id}/tenant/{tenant_id}/bearer-token/{token_id}` | 删除访问令牌 |
| `DeleteExternalIdPCertificate` | DELETE | `/v1/identity-stores/{identity_store_id}/external-idp/{idp_id}/certificate/{certificate_id}` | 删除外部身份提供商证书 |
| `DeleteExternalIdPConfigurationForDirectory` | DELETE | `/v1/identity-stores/{identity_store_id}/external-idp/{idp_id}` | 删除外部身份提供商配置 |
| `DeleteGroup` | DELETE | `/v1/identity-stores/{identity_store_id}/groups/{group_id}` | 删除用户组 |
| `DeleteGroupMembership` | DELETE | `/v1/identity-stores/{identity_store_id}/group-memberships/{membership_id}` | 解绑用户和组 |
| `DeleteMfaDeviceForUser` | DELETE | `/v1/identity-stores/{identity_store_id}/users/{user_id}/mfa-devices/{device_id}` | 删除用户MFA设备 |
| `DeleteProvisioningTenant` | DELETE | `/v1/identity-stores/{identity_store_id}/tenant/{tenant_id}` | 删除自动预置 |
| `DeleteSpCertificate` | DELETE | `/v1/identity-stores/{identity_store_id}/saml-certificates/{certificate_id}` | 删除服务提供商证书 |
| `DeleteUser` | DELETE | `/v1/identity-stores/{identity_store_id}/users/{user_id}` | 删除用户 |
| `DescribeGroup` | GET | `/v1/identity-stores/{identity_store_id}/groups/{group_id}` | 查询用户组详情 |
| `DescribeGroupMembership` | GET | `/v1/identity-stores/{identity_store_id}/group-memberships/{membership_id}` | 查询绑定关系详情 |
| `DescribeGroups` | POST | `/v1/identity-stores/{identity_store_id}/groups/batch-query` | 批量查询指定用户组详情 |
| `DescribePasswordPolicy` | GET | `/v1/identity-stores/{identity_store_id}/password-policy` | 查询自定义密码策略 |
| `DescribeUser` | GET | `/v1/identity-stores/{identity_store_id}/users/{user_id}` | 查询用户详情 |
| `DescribeUsers` | POST | `/v1/identity-stores/{identity_store_id}/users/batch-query` | 批量查询指定用户详情 |
| `DisableExternalIdPConfigurationForDirectory` | POST | `/v1/identity-stores/{identity_store_id}/external-idp/{idp_id}/disable` | 停用外部身份提供商 |
| `DisableUser` | POST | `/v1/identity-stores/{identity_store_id}/users/{user_id}/disable` | 禁用用户 |
| `EnableExternalIdPConfigurationForDirectory` | POST | `/v1/identity-stores/{identity_store_id}/external-idp/{idp_id}/enable` | 启用外部身份提供商 |
| `EnableUser` | POST | `/v1/identity-stores/{identity_store_id}/users/{user_id}/enable` | 启用用户 |
| `GetGroupId` | POST | `/v1/identity-stores/{identity_store_id}/groups/retrieve-group-id` | 查询用户组ID |
| `GetGroupMembershipId` | POST | `/v1/identity-stores/{identity_store_id}/group-memberships/retrieve-group-membership-id` | 查询绑定关系ID |

... and 24 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
