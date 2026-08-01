---
name: huaweicloud-cpcs
description: HuaweiCloud CPCS API guide. 43 APIs covering 应用管理, 应用集群关联, 访问密钥管理, 资源监控, 鉴权相关.
---

# HuaweiCloud CPCS API Guide

43 APIs. Tags: 应用管理, 应用集群关联, 访问密钥管理, 资源监控, 鉴权相关, 镜像管理, 集群实例管理, 集群端口管理, 集群管理, 集群认证, 集群跳转

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddClusterPort` | POST | `/v1/{project_id}/dew/cpcs/cluster/{cluster_id}/port` | 创建集群模式端口 |
| `AssociateApps` | POST | `/v1/{project_id}/dew/cpcs/associate-apps` | 创建密码服务集群与应用绑定关系 |
| `AuthorizeAccessKeys` | POST | `/v1/{project_id}/dew/cpcs/cluster/{cluster_id}/authorize-access-keys` | 密码服务集群授予应用访问密钥的访问权限 |
| `BatchDisableAccessKeys` | POST | `/v1/{project_id}/dew/cpcs/apps/{app_id}/access-keys/disable` | 停用应用的访问密钥 |
| `BatchEnableAccessKeys` | POST | `/v1/{project_id}/dew/cpcs/apps/{app_id}/access-keys/enable` | 启用应用的访问密钥 |
| `CancelAuthorizeAccessKeys` | POST | `/v1/{project_id}/dew/cpcs/cluster/{cluster_id}/de-authorize-access-keys` | 密码服务集群解除对访问密钥的授权 |
| `CheckClusterPort` | PUT | `/v1/{project_id}/dew/cpcs/cluster/{cluster_id}/port/{id}` | 检测集群模式端口是否正常 |
| `CreateApp` | POST | `/v1/{project_id}/dew/cpcs/apps` | 创建应用 |
| `CreateAppAccessKey` | POST | `/v1/{project_id}/dew/cpcs/apps/{app_id}/access-keys` | 创建访问密钥 |
| `CreateCluster` | POST | `/v1/{project_id}/dew/cpcs/cluster` | 创建密码服务集群 |
| `DeleteAccessKey` | DELETE | `/v1/{project_id}/dew/cpcs/apps/{app_id}/access-keys/{access_key_id}` | 删除应用的访问密钥 |
| `DeleteApp` | DELETE | `/v1/{project_id}/dew/cpcs/apps/{app_id}` | 删除应用 |
| `DeleteCcspCluster` | DELETE | `/v1/{project_id}/dew/cpcs/cluster/{cluster_id}` | 删除密码服务集群 |
| `DeleteClusterPort` | DELETE | `/v1/{project_id}/dew/cpcs/cluster/{cluster_id}/port/{id}` | 删除集群模式端口 |
| `DisableCcspInstance` | POST | `/v1/{project_id}/dew/cpcs/instances/{instance_id}/disable` | 停用密码服务实例的业务功能 |
| `DisassociateApps` | POST | `/v1/{project_id}/dew/cpcs/disassociate-apps` | 解除密码服务集群与应用绑定关系 |
| `EnableCcspInstance` | POST | `/v1/{project_id}/dew/cpcs/instances/{instance_id}/enable` | 启用密码服务实例的业务功能 |
| `ListCcspTenantImages` | GET | `/v1/{project_id}/dew/cpcs/images` | 查询密码服务的镜像 |
| `ListClusterPort` | GET | `/v1/{project_id}/dew/cpcs/cluster/{cluster_id}/port` | 查询集群模式端口列表 |
| `ShowAccessKey` | GET | `/v1/{project_id}/dew/cpcs/apps/{app_id}/access-keys/{access_key_id}` | 下载访问密钥 |
| `ShowAppAccessKeyList` | GET | `/v1/{project_id}/dew/cpcs/apps/{app_id}/access-keys` | 查询应用的访问密钥列表 |
| `ShowAppList` | GET | `/v1/{project_id}/dew/cpcs/apps` | 查询应用列表 |
| `ShowAssociationList` | GET | `/v1/{project_id}/dew/cpcs/associations` | 查询密码服务集群与应用的绑定关系列表 |
| `ShowAuditLog` | GET | `/v1/{project_id}/dew/cpcs/platform/audit-log` | 查询平台审计日志 |
| `ShowAvailableAz` | GET | `/v1/{project_id}/dew/cpcs/az` | 查询可创建密码服务集群的可用区列表 |
| `ShowCcspCluster` | GET | `/v1/{project_id}/dew/cpcs/cluster/{cluster_id}` | 查询密码服务集群详情 |
| `ShowCcspClusterList` | GET | `/v1/{project_id}/dew/cpcs/cluster` | 查询密码服务集群列表 |
| `ShowCcspInstanceInfo` | GET | `/v1/{project_id}/dew/cpcs/instances` | 查询密码服务实例列表 |
| `ShowClusterAccessKeyList` | GET | `/v1/{project_id}/dew/cpcs/cluster/{cluster_id}/access-keys` | 查询密码服务集群已授权的访问密钥列表 |
| `ShowClusterUri` | GET | `/v1/{project_id}/dew/cpcs/cluster/{cluster_id}/uri` | 获取密码服务管理界面URL |

... and 13 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
