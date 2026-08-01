---
name: huaweicloud-iam
description: HuaweiCloud IAM API guide. 235 APIs covering Credential管理, IAM用户管理, MFA设备管理, OIDC身份提供商管理, SAML身份提供商管理. Detailed swagger definitions in references/<APIName>.json.
---

# HuaweiCloud IAM API Guide

235 APIs. Tags: Credential管理, IAM用户管理, MFA设备管理, OIDC身份提供商管理, SAML身份提供商管理, Token管理, 企业项目管理, 凭据管理, 区域管理, 委托及信任委托管理, 委托管理, 安全设置, 授权概要查询, 服务和终端节点, 权限管理, 版本信息管理, 用户组管理, 联邦身份认证管理, 自定义代理, 自定义策略管理, 账号功能管理, 账号管理, 资源标签管理, 身份策略管理, 项目管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddClientIDToOIDCProviderV5` | POST | `/v5/oidc-providers/{provider_id}/add-client-id` | 向指定OIDC提供商添加客户端 ID |
| `AddUserToGroupV5` | POST | `/v5/groups/{group_id}/add-user` | 添加IAM用户到用户组 |
| `AssociateAgencyWithAllProjectsPermission` | PUT | `/v3.0/OS-INHERIT/domains/{domain_id}/agencies/{agency_id}/roles/{role_id}/inherited_to_projects` | 为委托授予所有项目服务权限 |
| `AssociateAgencyWithDomainPermission` | PUT | `/v3.0/OS-AGENCY/domains/{domain_id}/agencies/{agency_id}/roles/{role_id}` | 为委托授予全局服务权限 |
| `AssociateAgencyWithProjectPermission` | PUT | `/v3.0/OS-AGENCY/projects/{project_id}/agencies/{agency_id}/roles/{role_id}` | 为委托授予项目服务权限 |
| `AssociateRoleToAgencyOnEnterpriseProject` | PUT | `/v3.0/OS-PERMISSION/subjects/agency/scopes/enterprise-project/role-assignments` | 基于委托为企业项目授权 |
| `AssociateRoleToGroupOnEnterpriseProject` | PUT | `/v3.0/OS-PERMISSION/enterprise-projects/{enterprise_project_id}/groups/{group_id}/roles/{role_id}` | 基于用户组为企业项目授权 |
| `AssociateRoleToUserOnEnterpriseProject` | PUT | `/v3.0/OS-PERMISSION/enterprise-projects/{enterprise_project_id}/users/{user_id}/roles/{role_id}` | 基于用户为企业项目授权 |
| `AttachAgencyPolicyV5` | POST | `/v5/policies/{policy_id}/attach-agency` | 为委托或信任委托附加身份策略 |
| `AttachGroupPolicyV5` | POST | `/v5/policies/{policy_id}/attach-group` | 为用户组附加身份策略 |
| `AttachUserPolicyV5` | POST | `/v5/policies/{policy_id}/attach-user` | 为IAM用户附加身份策略 |
| `ChangePasswordV5` | POST | `/v5/caller-password` | 修改IAM用户密码 |
| `CheckAllProjectsPermissionForAgency` | HEAD | `/v3.0/OS-INHERIT/domains/{domain_id}/agencies/{agency_id}/roles/{role_id}/inherited_to_projects` | 检查委托下是否具有所有项目服务权限 |
| `CheckDomainPermissionForAgency` | HEAD | `/v3.0/OS-AGENCY/domains/{domain_id}/agencies/{agency_id}/roles/{role_id}` | 查询委托是否拥有全局服务权限 |
| `CheckProjectPermissionForAgency` | HEAD | `/v3.0/OS-AGENCY/projects/{project_id}/agencies/{agency_id}/roles/{role_id}` | 查询委托是否拥有项目服务权限 |
| `CreateAccessKeyV5` | POST | `/v5/users/{user_id}/access-keys` | 创建永久访问密钥 |
| `CreateAgency` | POST | `/v3.0/OS-AGENCY/agencies` | 创建委托 |
| `CreateAgencyCustomPolicy` | POST | `/v3.0/OS-ROLE/roles` | 创建委托自定义策略 |
| `CreateAgencyV5` | POST | `/v5/agencies` | 创建信任委托 |
| `CreateBindingDevice` | PUT | `/v3.0/OS-MFA/mfa-devices/bind` | 绑定MFA设备 |
| `CreateCloudServiceCustomPolicy` | POST | `/v3.0/OS-ROLE/roles` | 创建云服务自定义策略 |
| `CreateGroupV5` | POST | `/v5/groups` | 创建用户组 |
| `CreateLoginProfileV5` | POST | `/v5/users/{user_id}/login-profile` | 创建IAM用户登录信息 |
| `CreateLoginToken` | POST | `/v3.0/OS-AUTH/securitytoken/logintokens` | 获取自定义代理登录票据 |
| `CreateMetadata` | POST | `/v3-ext/OS-FEDERATION/identity_providers/{idp_id}/protocols/{protocol_id}/metadata` | 导入Metadata文件 |
| `CreateMfaDevice` | POST | `/v3.0/OS-MFA/virtual-mfa-devices` | 创建MFA设备 |
| `CreateOIDCProviderV5` | POST | `/v5/oidc-providers` | 创建OIDC提供商 |
| `CreateOpenIdConnectConfig` | POST | `/v3.0/OS-FEDERATION/identity-providers/{idp_id}/openid-connect-config` | 创建OpenId Connect身份提供商配置 |
| `CreatePermanentAccessKey` | POST | `/v3.0/OS-CREDENTIAL/credentials` | 创建永久访问密钥 |
| `CreatePolicyV5` | POST | `/v5/policies` | 创建自定义身份策略 |

... and 205 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
