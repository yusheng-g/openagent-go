---
name: huaweicloud-identitycenter
description: HuaweiCloud IdentityCenter API guide. 81 APIs covering MFA配置管理, 实例管理, 实例访问控制属性配置管理, 实例配置管理, 应用程序分配管理.
---

# HuaweiCloud IdentityCenter API Guide

81 APIs. Tags: MFA配置管理, 实例管理, 实例访问控制属性配置管理, 实例配置管理, 应用程序分配管理, 应用程序管理, 应用程序证书管理, 权限集管理, 标签管理, 账号分配管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AttachManagedPolicyToPermissionSet` | POST | `/v1/instances/{instance_id}/permission-sets/{permission_set_id}/attach-managed-policy` | 添加系统身份策略 |
| `AttachManagedRoleToPermissionSet` | POST | `/v1/instances/{instance_id}/permission-sets/{permission_set_id}/attach-managed-role` | 添加系统策略 |
| `CreateAccountAssignment` | POST | `/v1/instances/{instance_id}/account-assignments/create` | 创建账户分配 |
| `CreateAlias` | POST | `/v1/instances/{instance_id}/alias` | 自定义访问门户URL |
| `CreateApplicationAssignment` | POST | `/v1/instances/{instance_id}/applications/{application_instance_id}/assignments/create` | 应用程序分配用户或用户组 |
| `CreateApplicationInstance` | POST | `/v1/instances/{instance_id}/application-instances` | 创建应用程序实例 |
| `CreateApplicationInstanceCertificate` | POST | `/v1/instances/{instance_id}/application-instances/{application_instance_id}/certificates` | 创建应用程序实例证书 |
| `CreateInstanceAccessControlAttributeConfiguration` | POST | `/v1/instances/{instance_id}/access-control-attribute-configuration` | 启用指定实例的访问控制功能 |
| `CreatePermissionSet` | POST | `/v1/instances/{instance_id}/permission-sets` | 创建权限集 |
| `CreateTagResource` | POST | `/v1/instances/{resource_type}/{resource_id}/tags/create` | 为指定资源添加标签 |
| `DeleteAccountAssignment` | POST | `/v1/instances/{instance_id}/account-assignments/delete` | 删除账号分配 |
| `DeleteApplicationAssignment` | POST | `/v1/instances/{instance_id}/applications/{application_instance_id}/assignments/delete` | 删除应用程序已分配用户或用户组 |
| `DeleteApplicationInstance` | DELETE | `/v1/instances/{instance_id}/application-instances/{application_instance_id}` | 删除应用程序实例 |
| `DeleteApplicationInstanceCertificate` | DELETE | `/v1/instances/{instance_id}/application-instances/{application_instance_id}/certificates/{certificate_id}` | 删除应用程序实例证书 |
| `DeleteCustomPolicyFromPermissionSet` | DELETE | `/v1/instances/{instance_id}/permission-sets/{permission_set_id}/custom-policy` | 删除自定义身份策略 |
| `DeleteCustomRoleFromPermissionSet` | DELETE | `/v1/instances/{instance_id}/permission-sets/{permission_set_id}/custom-role` | 删除自定义策略 |
| `DeleteIdentityCenter` | POST | `/v1/service/delete` | 删除服务实例 |
| `DeleteInstanceAccessControlAttributeConfiguration` | DELETE | `/v1/instances/{instance_id}/access-control-attribute-configuration` | 解除指定实例的访问控制属性配置 |
| `DeletePermissionSet` | DELETE | `/v1/instances/{instance_id}/permission-sets/{permission_set_id}` | 删除权限集 |
| `DeleteProfile` | DELETE | `/v1/instances/{instance_id}/application-instances/{application_instance_id}/profiles/{profile_id}` | 删除应用程序实例与用户或用户组关联关系 |
| `DeleteTagResource` | POST | `/v1/instances/{resource_type}/{resource_id}/tags/delete` | 从指定资源中删除指定主键标签 |
| `DescribeAccountAssignmentCreationStatus` | GET | `/v1/instances/{instance_id}/account-assignments/creation-status/{request_id}` | 查询账户分配创建状态详情 |
| `DescribeAccountAssignmentDeletionStatus` | GET | `/v1/instances/{instance_id}/account-assignments/deletion-status/{request_id}` | 查询账户分配删除状态详情 |
| `DescribeApplication` | GET | `/v1/instances/{instance_id}/applications/{application_instance_id}` | 查询应用程序详情 |
| `DescribeApplicationProvider` | GET | `/v1/application-providers/{application_provider_id}` | 查询应用程序提供者详情 |
| `DescribeInstanceAccessControlAttributeConfiguration` | GET | `/v1/instances/{instance_id}/access-control-attribute-configuration` | 获取指定实例的访问控制属性配置 |
| `DescribePermissionSet` | GET | `/v1/instances/{instance_id}/permission-sets/{permission_set_id}` | 查询权限集详情 |
| `DescribePermissionSetProvisioningStatus` | GET | `/v1/instances/{instance_id}/permission-sets/provisioning-status/{request_id}` | 查询权限集预分配状态详情 |
| `DescribeRegisteredRegions` | GET | `/v1/registered-regions` | 查询服务实例开通所在区域 |
| `DetachManagedPolicyFromPermissionSet` | POST | `/v1/instances/{instance_id}/permission-sets/{permission_set_id}/detach-managed-policy` | 删除系统身份策略 |

... and 51 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
