---
name: huaweicloud-manageoneschcs
description: HuaweiCloud ManageOneSCHCS API guide. 447 APIs covering Console Home, IAM1.0, VDC, VRM云服务, 产品管理.
---

# HuaweiCloud ManageOneSCHCS API Guide

447 APIs. Tags: Console Home, IAM1.0, VDC, VRM云服务, 产品管理, 服务构建器, 标签管理, 流程审批, 流程编排引擎, 计量计价, 订单管理, 资源生命周期管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ActionDeployment` | POST | `/rest/application/v3.0/applications/{application_id}/deployments-action` | 操作应用部署(已废弃) |
| `ActionExternalNetwork` | PUT | `/rest/orchestration/v3.0/fcvpc/available-external-networks/action` | 关联外部网络(FusionCompute) |
| `AddClusterToAvailableZone` | POST | `/rest/orchestration/v3.0/fcserviceaccess/clusters` | 添加集群(FusionCompute) |
| `AddResource` | PUT | `/rest/application/v3.0/applications/{application_id}/add-resources` | 应用添加资源(已废弃) |
| `AddResourceConfig` | POST | `/rest/application/v3.0/applications/resource-config` | 新增应用资源配置信息(已废弃) |
| `AddRolesInheritToProjects` | PUT | `/rest/vdc/v3.0/vdc-agencies/{agency_id}/domains/{domain_id}/roles/{role_id}/inherited_to_projects` | 设置租户支持的云服务委托 |
| `AddRolesToDomain` | PUT | `/rest/vdc/v3.0/vdc-agencies/{agency_id}/domains/{domain_id}/roles/{role_id}` | 委托基于租户授权 |
| `AddRolesToProjects` | PUT | `/rest/vdc/v3.0/vdc-agencies/{agency_id}/projects/{project_id}/roles/{role_id}` | 委托基于资源空间授权 |
| `AddUserToGroup` | PUT | `/rest/vdc/v3.2/groups/{group_id}/users/{user_id}` | 用户组添加用户 |
| `ApplyOpenapiStack` | POST | `/rest/vapp/v3.0/instances` | 北向调用场景申请实例 |
| `BatchCreateTags` | POST | `/rest/tag/v3.0/tags` | 创建/删除标签 |
| `BatchDeleteApplicationDeployments` | POST | `/rest/application/v3.0/applications/{application_id}/deployments/batch-delete` | 批量删除应用部署(已废弃) |
| `BatchDeleteApplicationModules` | POST | `/rest/application/v3.0/applications/{application_id}/modules/batch-delete` | 批量删除应用模块 |
| `BatchDeleteSubscriptions` | POST | `/rest/shoppingcart/v3.0/subscriptions/delete` | 删除购物车中的产品 |
| `BatchResourceEnterpriseActions` | POST | `/rest/v1.0/batch/enterprise/action` | 批量变更企业项目 |
| `BindUserRole` | PUT | `/rest/vdc/v3.1/users/{user_id}/projects/{project_id}/roles/{role_id}` | 根据用户标识,资源空间标识关联角色 |
| `ChangeApplicationStatus` | PUT | `/rest/application/v5.0/applications/{id}/change-status` | 变更应用状态 |
| `CheckDomainGroupRole` | HEAD | `/v3/domains/{domain_id}/groups/{group_id}/roles/{role_id}` | 查询租户中用户组的权限 |
| `CheckProjectGroupRole` | HEAD | `/v3/projects/{project_id}/groups/{group_id}/roles/{role_id}` | 查询项目对应的用户组是否包含权限 |
| `CheckUserInGroup` | HEAD | `/v3/groups/{group_id}/users/{user_id}` | 查询用户是否在用户组中 |
| `CommitSubscriptions` | POST | `/rest/shoppingcart/v3.0/subscriptions` | 购物车提交订购 |
| `CopyApplicationDeployment` | POST | `/rest/application/v3.0/applications/{application_id}/deployments/{id}/duplicate` | 复制应用部署(已废弃) |
| `CountTodoTasks` | GET | `/rest/todo/v1.0/count` | 查询待办列表总数 |
| `CountVDCResources` | GET | `/rest/vdc/v3.0/capacity/{vdc_id}/statics` | 查询VDC资源概览 |
| `CreateAKSK` | POST | `/rest/v3.0/aks/{user_id}` | 创建AK/SK |
| `CreateApplication` | POST | `/rest/application/v3.0/applications` | 创建应用(已废弃) |
| `CreateApplicationDeployment` | POST | `/rest/application/v3.0/applications/{application_id}/deployments` | 创建应用部署(已废弃) |
| `CreateApplicationModule` | POST | `/rest/application/v3.0/applications/{application_id}/modules` | 创建应用模块 |
| `CreateAvailableZones` | POST | `/rest/orchestration/v3.0/fcserviceaccess/available-zones` | 创建可用分区(FusionCompute) |
| `CreateCatalogs` | POST | `/silvan/rest/v1.0/catalogs` | 新增服务分类信息 |

... and 417 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
