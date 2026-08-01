---
name: huaweicloud-vpcep
description: HuaweiCloud VPCEP API guide. 33 APIs covering Endpoint, TAG功能, 版本管理, 终端节点功能, 终端节点服务功能.
---

# HuaweiCloud VPCEP API Guide

33 APIs. Tags: Endpoint, TAG功能, 版本管理, 终端节点功能, 终端节点服务功能, 资源配额功能

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptOrRejectEndpoint` | POST | `/v1/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}/connections/action` | 接受或拒绝终端节点的连接 |
| `AddEndpointServiceServerResource` | POST | `/v2/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}/add-server-resources` | 添加终端节点服务后端服务资源 |
| `AddOrRemoveServicePermissions` | POST | `/v1/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}/permissions/action` | 批量添加或移除终端节点服务的白名单 |
| `BatchAddEndpointServicePermissions` | POST | `/v1/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}/permissions/batch-create` | 批量添加终端节点服务的白名单 |
| `BatchAddOrRemoveResourceInstance` | POST | `/v1/{project_id}/{resource_type}/{resource_id}/tags/action` | 批量添加或删除资源标签接口 |
| `BatchRemoveEndpointServicePermissions` | POST | `/v1/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}/permissions/batch-delete` | 批量删除终端节点服务的白名单 |
| `CreateEndpoint` | POST | `/v1/{project_id}/vpc-endpoints` | 创建终端节点 |
| `CreateEndpointService` | POST | `/v1/{project_id}/vpc-endpoint-services` | 创建终端节点服务 |
| `DeleteEndpoint` | DELETE | `/v1/{project_id}/vpc-endpoints/{vpc_endpoint_id}` | 删除终端节点 |
| `DeleteEndpointPolicy` | DELETE | `/v1/{project_id}/vpc-endpoints/{vpc_endpoint_id}/policy` | 删除网关型终端节点策略(待下线) |
| `DeleteEndpointService` | DELETE | `/v1/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}` | 删除终端节点服务 |
| `ListEndpointInfoDetails` | GET | `/v1/{project_id}/vpc-endpoints/{vpc_endpoint_id}` | 查询终端节点详情 |
| `ListEndpoints` | GET | `/v1/{project_id}/vpc-endpoints` | 查询终端节点列表 |
| `ListEndpointService` | GET | `/v1/{project_id}/vpc-endpoint-services` | 查询终端节点服务列表 |
| `ListQueryProjectResourceTags` | GET | `/v1/{project_id}/{resource_type}/tags` | 查询租户资源标签接口 |
| `ListQuotaDetails` | GET | `/v1/{project_id}/quotas` | 查询配额 |
| `ListResourceInstances` | POST | `/v1/{project_id}/{resource_type}/resource_instances/action` | 查询资源实例接口 |
| `ListServiceConnections` | GET | `/v1/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}/connections` | 查询连接终端节点服务的连接列表 |
| `ListServiceDescribeDetails` | GET | `/v1/{project_id}/vpc-endpoint-services/describe` | 查询终端节点服务概要 |
| `ListServiceDetails` | GET | `/v1/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}` | 查询终端节点服务详情 |
| `ListServicePermissionsDetails` | GET | `/v1/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}/permissions` | 查询终端节点服务的白名单列表 |
| `ListServicePublicDetails` | GET | `/v1/{project_id}/vpc-endpoint-services/public` | 查询公共终端节点服务列表 |
| `ListSpecifiedVersionDetails` | GET | `/{version}` | 查询指定VPC终端节点接口版本信息 |
| `ListVersionDetails` | GET | `/` | 查询VPC终端节点接口版本列表 |
| `UpdateEndpointConnectionsDesc` | PUT | `/v1/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}/connections/description` | 更新终端节点连接描述 |
| `UpdateEndpointPolicy` | PUT | `/v1/{project_id}/vpc-endpoints/{vpc_endpoint_id}/policy` | 修改终端节点策略 |
| `UpdateEndpointRoutetable` | PUT | `/v1/{project_id}/vpc-endpoints/{vpc_endpoint_id}/routetables` | 修改终端节点的路由表 |
| `UpdateEndpointService` | PUT | `/v1/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}` | 修改终端节点服务 |
| `UpdateEndpointServiceName` | PUT | `/v1/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}/name` | 修改终端节点服务名称 |
| `UpdateEndpointServicePermissionDesc` | PUT | `/v1/{project_id}/vpc-endpoint-services/{vpc_endpoint_service_id}/permissions/{permission_id}` | 更新终端节点服务白名单描述 |

... and 3 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
