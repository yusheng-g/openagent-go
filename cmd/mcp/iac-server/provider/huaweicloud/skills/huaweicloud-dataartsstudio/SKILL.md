---
name: huaweicloud-dataartsstudio
description: HuaweiCloud DataArtsStudio API guide. 386 APIs covering API管理接口, 业务指标接口, 业务资产接口, 主题层级接口, 主题管理接口.
---

# HuaweiCloud DataArtsStudio API Guide

386 APIs. Tags: API管理接口, 业务指标接口, 业务资产接口, 主题层级接口, 主题管理接口, 事实表接口, 企业模式发布包管理, 作业开发接口, 信息架构接口, 元数据实时同步, 元数据采集任务接口, 关系建模接口, 动态数据脱敏接口, 原子指标接口, 发布包管理, 基线运维, 复合指标接口, 安全管理员接口, 实例管理接口, 实例规格变更接口, 审批管理接口, 密级管理接口, 对账作业接口, 导入导出接口, 工作空间用户管理接口, 工作空间管理接口, 应用管理接口, 总览接口, 成本管理计算维度接口, 指标资产接口, 授权管理接口, 敏感数据结果管理接口, 数仓分层接口, 数据分类接口, 数据地图接口, 数据安全诊断接口, 数据开发-审批管理接口, 数据权限查询接口, 数据标准接口, 数据标准模板接口, 数据源元数据获取接口, 数据源接口, 数据连接管理, 数据连接管理接口, 服务目录管理接口, 权限审批接口, 权限应用接口, 权限申请接口, 权限管理接口, 标签接口, 概览, 汇总表接口, 流程架构接口, 消息管理接口, 版本信息接口, 用户同步任务接口, 用户同步接口, 申请管理接口, 监控运维, 目录接口, 目录管理, 码表管理接口, 空间资源权限策略管理接口, 统计资产接口, 维度接口, 维度表接口, 网关管理接口, 脚本开发接口, 自定义项接口, 血缘信息, 衍生指标接口, 规则分组接口, 规则模板接口, 识别规则接口, 质量作业接口, 质量规则接口, 购买实例接口, 资产信息, 资产分类接口, 资产分级接口, 资产管理接口, 运维管理接口, 通知管理, 队列权限接口, 限定接口, 集群管理接口

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptSecurityApplication` | POST | `/v1/{project_id}/security/permission-approve/approve/{id}` | 审批通过工单 |
| `AddDesignEntityTags` | PUT | `/v2/{project_id}/design/{entity_id}/tags` | 添加标签 |
| `AddTagToAsset` | POST | `/v3/{project_id}/tags/{term_guid}/assignedentities` | 标签关联到资产 |
| `AddWorkSpaceUsers` | POST | `/v2/{project_id}/{workspace_id}/users` | 添加工作空间用户 |
| `ApplySecurityTableAuthority` | POST | `/v1/{project_id}/security/permission-application/table` | 提交表权限申请 |
| `AssociateClassificationToEntity` | POST | `/v3/{project_id}/asset/entities/guid/{guid}/classification` | 资产关联分类 |
| `AssociateSecurityLevelToEntitie` | PUT | `/v3/{project_id}/asset/entities/guid/{guid}/security-level` | 资产关联密级 |
| `AuthorizeActionApiToInstance` | POST | `/v1/{project_id}/service/apis/authorize/action` | API授权操作(授权/取消授权/申请/续约) |
| `AuthorizeApiToInstance` | POST | `/v1/{project_id}/service/apis/{api_id}/instances/{instance_id}/authorize` | 批量授权API(专享版) |
| `AuthorizeDataConnection` | POST | `/v1/{project_id}/datasources/authorize_datasource` | 数据连接跨空间授权 |
| `BatchApproveApply` | POST | `/v1/{project_id}/service/applys` | 审核申请 |
| `BatchApproveSecurityApplications` | POST | `/v1/{project_id}/security/openapi/permission-approve/batch-approve` | 批量审批通过工单 |
| `BatchAssociateClassificationToEntities` | POST | `/v3/{project_id}/asset/entities/classification` | 批量资产关联分类 |
| `BatchAssociateSecurityLevelToEntities` | PUT | `/v3/{project_id}/asset/entities/security-level` | 批量资产关联密级 |
| `BatchCreateDesignTableModelsFromLogic` | PUT | `/v1/{project_id}/design/workspaces/{model_id}/transform` | 转换逻辑模型为物理模型 |
| `BatchCreateSecurityPermissionSetMembers` | POST | `/v1/{project_id}/security/permission-sets/{permission_set_id}/members/batch-create` | 批量添加权限集成员 |
| `BatchCreateSecurityPermissionSetPermissions` | POST | `/v1/{project_id}/security/permission-sets/{permission_set_id}/permissions/batch-append` | 批量添加权限集的权限 |
| `BatchDeleteSecurityDataCategories` | POST | `/v1/{project_id}/security/data-category/batch-delete` | 删除数据分类 |
| `BatchDeleteSecurityDataClassificationRule` | POST | `/v1/{project_id}/security/data-classification/rule/batch-delete` | 批量删除识别规则接口 |
| `BatchDeleteSecurityDynamicMaskingPolicies` | POST | `/v1/{project_id}/security/masking/dynamic/policies/batch-delete` | 批量删除动态脱敏策略 |
| `BatchDeleteSecurityPermissionSetMembers` | POST | `/v1/{project_id}/security/permission-sets/{permission_set_id}/members/batch-delete` | 批量删除权限集成员 |
| `BatchDeleteSecurityPermissionSetPermissions` | POST | `/v1/{project_id}/security/permission-sets/{permission_set_id}/permissions/batch-delete` | 删除权限集的权限 |
| `BatchDeleteSecurityResourcePermissionPolicies` | POST | `/v1/{project_id}/security/permission-resource/batch-delete` | 批量删除资源权限策略 |
| `BatchDeleteSecuritySecrecyLevels` | POST | `/v1/{project_id}/security/data-classification/secrecy-level/batch-delete` | 批量删除密级 |
| `BatchDeleteTemplates` | POST | `/v2/{project_id}/quality/rule-templates/batch-delete` | 批量删除规则模板 |
| `BatchOffline` | POST | `/v2/{project_id}/design/approvals/batch-offline` | 批量下线 |
| `BatchPublish` | POST | `/v2/{project_id}/design/approvals/batch-publish` | 批量发布 |
| `BatchRejectSecurityApplications` | POST | `/v1/{project_id}/security/openapi/permission-approve/batch-reject` | 批量驳回工单 |
| `BatchSyncMetadata` | POST | `/v1/{project_id}/metadata/async-bulk` | 元数据实时同步接口(邀测) |
| `BatchTag` | POST | `/v1/{project_id}/datamap/entities/guids/tags` | 批量打标签(邀测) |

... and 356 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
