---
name: huaweicloud-lakeformation
description: HuaweiCloud LakeFormation API guide. 95 APIs covering Configuration, Credential, OBS管理, QM, User.
---

# HuaweiCloud LakeFormation API Guide

95 APIs. Tags: Configuration, Credential, OBS管理, QM, User, 元数据统计, 函数管理, 分区管理, 分区统计信息, 委托管理, 实例管理, 接入管理, 数据库管理, 数据表管理, 数据表统计, 服务授权管理, 标签管理服务, 用户组管理, 目录管理, 规格管理, 角色管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddPartitions` | POST | `/v1/{project_id}/instances/{instance_id}/catalogs/{catalog_name}/databases/{database_name}/tables/{table_name}/partitions/batch-create` | 批量添加分区信息 |
| `ApplyForAccess` | POST | `/v1/{project_id}/instances/{instance_id}/access` | 申请接入服务 |
| `AssociatePrincipals` | POST | `/v1/{project_id}/instances/{instance_id}/roles/{role_name}/grant-principals` | 将一个或者多个用户/用户组加入角色 |
| `AssociateRoles` | POST | `/v1/{project_id}/instances/{instance_id}/users/{user_name}/grant-roles` | 将多个角色授予User |
| `AuthorizeAccessService` | POST | `/v1/{project_id}/access-service` | 接入服务授权 |
| `BatchAuthorizeInterface` | POST | `/v1/{project_id}/instances/{instance_id}/policies/grant` | 批量授权 |
| `BatchCancelAuthorizationInterface` | POST | `/v1/{project_id}/instances/{instance_id}/policies/revoke` | 取消批量授权 |
| `BatchCheckPermission` | POST | `/v1/{project_id}/instances/{instance_id}/policies/check-permission` | 批量鉴权 |
| `BatchDeletePartition` | POST | `/v1/{project_id}/instances/{instance_id}/catalogs/{catalog_name}/databases/{database_name}/tables/{table_name}/partitions/batch-drop` | 批量删除分区信息 |
| `BatchListPartitionByValues` | POST | `/v1/{project_id}/instances/{instance_id}/catalogs/{catalog_name}/databases/{database_name}/tables/{table_name}/partitions/batch-get` | 批量获取分区信息 |
| `BatchShowPartitionColumnStatistics` | POST | `/v1/{project_id}/instances/{instance_id}/catalogs/{catalog_name}/databases/{database_name}/tables/{table_name}/partitions/column-statistics/batch-get` | 批量获取分区的列统计信息 |
| `BatchUpdateLakeFormationInstanceTags` | PUT | `/v1/{project_id}/instances/{instance_id}/tags` | 批量更新标签 |
| `BatchUpdatePartition` | POST | `/v1/{project_id}/instances/{instance_id}/catalogs/{catalog_name}/databases/{database_name}/tables/{table_name}/partitions/batch-alter` | 批量修改分区信息 |
| `CountMetaObj` | POST | `/v1/{project_id}/instances/{instance_id}/metaobj/count` | 元数据数量统计 |
| `CreateAccessClient` | POST | `/v1/{project_id}/instances/{instance_id}/access-clients` | 创建服务接入客户端 |
| `CreateAgency` | POST | `/v2/agency` | 创建委托 |
| `CreateAgreement` | POST | `/v2/agreement` | 注册租户协议 |
| `CreateCatalog` | POST | `/v1/{project_id}/instances/{instance_id}/catalogs` | 创建catalog |
| `CreateDatabase` | POST | `/v1/{project_id}/instances/{instance_id}/catalogs/{catalog_name}/databases` | 创建数据库 |
| `CreateFunction` | POST | `/v1/{project_id}/instances/{instance_id}/catalogs/{catalog_name}/databases/{database_name}/functions` | 创建函数 |
| `CreateLakeFormationInstance` | POST | `/v1/{project_id}/instances` | 创建实例 |
| `CreateRole` | POST | `/v1/{project_id}/instances/{instance_id}/roles` | 创建role |
| `CreateTable` | POST | `/v1/{project_id}/instances/{instance_id}/catalogs/{catalog_name}/databases/{database_name}/tables` | 创建表 |
| `DeleteAccessClient` | DELETE | `/v1/{project_id}/instances/{instance_id}/access-clients/{client_id}` | 删除服务接入客户端 |
| `DeleteAgency` | DELETE | `/v2/agency` | 删除委托 |
| `DeleteAgreement` | DELETE | `/v2/agreement` | 删除租户协议 |
| `DeleteCatalog` | DELETE | `/v1/{project_id}/instances/{instance_id}/catalogs/{catalog_name}` | 删除catalog对象 |
| `DeleteDatabase` | DELETE | `/v1/{project_id}/instances/{instance_id}/catalogs/{catalog_name}/databases/{database_name}` | 删除数据库 |
| `DeleteFunction` | DELETE | `/v1/{project_id}/instances/{instance_id}/catalogs/{catalog_name}/databases/{database_name}/functions/{function_name}` | 删除函数 |
| `DeleteLakeFormationInstance` | DELETE | `/v1/{project_id}/instances/{instance_id}` | 删除实例 |

... and 65 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
