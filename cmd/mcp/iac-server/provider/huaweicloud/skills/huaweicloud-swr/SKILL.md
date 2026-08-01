---
name: huaweicloud-swr
description: HuaweiCloud SWR API guide. 170 APIs covering API版本信息, 临时登录指令, 任务管理(SWR企业版), 共享帐号管理, 其他.
---

# HuaweiCloud SWR API Guide

170 APIs. Tags: API版本信息, 临时登录指令, 任务管理(SWR企业版), 共享帐号管理, 其他, 制品仓库管理(SWR企业版), 制品扫描(SWR企业版), 制品清理(SWR企业版), 制品版本管理(SWR企业版), 命名空间管理(SWR企业版), 域名管理(SWR企业版), 实例管理(SWR企业版), 标签管理(SWR企业版), 特性开关(SWR企业版), 组织权限管理, 组织管理, 老化策略管理(SWR企业版), 触发器管理, 触发器管理(SWR企业版), 访问凭证管理(SWR企业版), 访问控制管理(SWR企业版), 配额管理, 镜像仓库管理, 镜像同步管理, 镜像同步管理(SWR企业版), 镜像权限管理, 镜像版本不可变(SWR企业版), 镜像版本管理, 镜像签名管理(SWR企业版), 镜像老化规则管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddDomainName` | POST | `/v2/{project_id}/instances/{instance_id}/domainname` | 增加域名 |
| `CheckAgency` | GET | `/v2/manage/agency` | 查询委托是否存在 |
| `CreateAgency` | POST | `/v2/manage/agency` | 创建委托 |
| `CreateAuthorizationToken` | POST | `/v2/manage/utils/authorizationtoken` | 生成增强型登录指令(新) |
| `CreateImageSyncRepo` | POST | `/v2/manage/namespaces/{namespace}/repos/{repository}/sync_repo` | 创建镜像自动同步任务 |
| `CreateImmutableRule` | POST | `/v2/{project_id}/instances/{instance_id}/namespaces/{namespace_name}/immutabletagrules` | 创建不可变Tag策略 |
| `CreateInstance` | POST | `/v2/{project_id}/instances` | 创建实例 |
| `CreateInstanceEndpointPolicy` | POST | `/v2/{project_id}/instances/{instance_id}/endpoint-policy` | 开启或关闭公网访问 |
| `CreateInstanceInternalEndpoint` | POST | `/v2/{project_id}/instances/{instance_id}/internal-endpoints` | 新增内网访问 |
| `CreateInstanceLtCredential` | POST | `/v2/{project_id}/instances/{instance_id}/long-term-credential` | 创建长期访问凭证 |
| `CreateInstanceNamespace` | POST | `/v2/{project_id}/instances/{instance_id}/namespaces` | 创建命名空间 |
| `CreateInstanceRegistry` | POST | `/v2/{project_id}/instances/{instance_id}/registries` | 创建镜像同步的目标仓库 |
| `CreateInstanceReplicationPolicy` | POST | `/v2/{project_id}/instances/{instance_id}/replication/policies` | 创建镜像同步策略 |
| `CreateInstanceResourceTags` | POST | `/v2/{project_id}/{resource_type}/{resource_id}/tags/create` | 批量添加资源标签 |
| `CreateInstanceRetentionPolicy` | POST | `/v2/{project_id}/instances/{instance_id}/namespaces/{namespace_name}/retention/policies` | 创建老化策略 |
| `CreateInstanceSignPolicy` | POST | `/v2/{project_id}/instances/{instance_id}/namespaces/{namespace_name}/signature/policies` | 创建签名策略 |
| `CreateInstanceTempCredential` | POST | `/v2/{project_id}/instances/{instance_id}/temp-credential` | 获取临时访问凭证 |
| `CreateInstanceWebhook` | POST | `/v2/{project_id}/instances/{instance_id}/namespaces/{namespace_name}/webhook/policies` | 创建触发器 |
| `CreateManualImageSyncRepo` | POST | `/v2/manage/namespaces/{namespace}/repos/{repository}/sync_images` | 手动同步镜像 |
| `CreateNamespace` | POST | `/v2/manage/namespaces` | 创建组织 |
| `CreateNamespaceAuth` | POST | `/v2/manage/namespaces/{namespace}/access` | 创建组织权限 |
| `CreateRepo` | POST | `/v2/manage/namespaces/{namespace}/repos` | 在组织下创建镜像仓库 |
| `CreateRepoDomains` | POST | `/v2/manage/namespaces/{namespace}/repositories/{repository}/access-domains` | 创建共享帐号 |
| `CreateRepoTag` | POST | `/v2/manage/namespaces/{namespace}/repos/{repository}/tags` | 创建镜像tag |
| `CreateRetention` | POST | `/v2/manage/namespaces/{namespace}/repos/{repository}/retentions` | 创建镜像老化规则 |
| `CreateSecret` | POST | `/v2/manage/utils/secret` | 生成临时登录指令 |
| `CreateSubResourceTags` | POST | `/v2/{project_id}/{resource_type}/{resource_id}/{sub_resource_type}/{sub_resource_id}/tags/create` | 批量添加子资源标签 |
| `CreateTrigger` | POST | `/v2/manage/namespaces/{namespace}/repos/{repository}/triggers` | 创建触发器 |
| `CreateUserRepositoryAuth` | POST | `/v2/manage/namespaces/{namespace}/repos/{repository}/access` | 创建镜像权限 |
| `DeleteDomainName` | DELETE | `/v2/{project_id}/instances/{instance_id}/domainname/{domainname_id}` | 删除域名 |

... and 140 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
