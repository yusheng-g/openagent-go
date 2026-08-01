---
name: huaweicloud-config
description: HuaweiCloud Config API guide. 106 APIs covering 区域管理, 合规性, 合规规则包, 聚合器, 资源关系.
---

# HuaweiCloud Config API Guide

106 APIs. Tags: 区域管理, 合规性, 合规规则包, 聚合器, 资源关系, 资源历史, 资源标签, 资源清单, 资源记录器, 高级查询

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateRemediationExceptions` | POST | `/v1/resource-manager/domains/{domain_id}/policy-assignments/{policy_assignment_id}/remediation-exception/create` | 批量创建修正例外 |
| `BatchDeleteRemediationExceptions` | POST | `/v1/resource-manager/domains/{domain_id}/policy-assignments/{policy_assignment_id}/remediation-exception/delete` | 批量删除修正例外 |
| `CollectAllResourcesSummary` | GET | `/v1/resource-manager/domains/{domain_id}/all-resources/summary` | 列举资源概要 |
| `CollectConformancePackComplianceSummary` | GET | `/v1/resource-manager/domains/{domain_id}/conformance-packs/compliance/summary` | 列举合规规则包的结果概览 |
| `CollectPolicyAssignmentsStatesSummary` | GET | `/v1/resource-manager/domains/{domain_id}/policy-assignments/{policy_assignment_id}/policy-states/summary` | 查询规则的合规总结 |
| `CollectPolicyStatesSummary` | GET | `/v1/resource-manager/domains/{domain_id}/policy-states/summary` | 查询用户的合规总结 |
| `CollectRemediationExecutionStatusesSummary` | POST | `/v1/resource-manager/domains/{domain_id}/policy-assignments/{policy_assignment_id}/remediation-execution-statuses/summary` | 列举修正最新记录 |
| `CollectResourcesPolicyStatesSummary` | GET | `/v1/resource-manager/domains/{domain_id}/resources/policy-states/summary` | 查询用户资源的合规总结 |
| `CollectTrackedResourcesSummary` | GET | `/v1/resource-manager/domains/{domain_id}/tracked-resources/summary` | 列举资源记录器收集的资源概要 |
| `CountAllResources` | GET | `/v1/resource-manager/domains/{domain_id}/all-resources/count` | 查询资源数量 |
| `CountResourcesByTag` | POST | `/v1/resource-manager/{resource_type}/resource-instances/count` | 查询资源实例数量 |
| `CountTrackedResources` | GET | `/v1/resource-manager/domains/{domain_id}/tracked-resources/count` | 查询资源记录器收集的资源数量 |
| `CreateAggregationAuthorization` | PUT | `/v1/resource-manager/domains/{domain_id}/aggregators/aggregation-authorization` | 创建资源聚合器授权 |
| `CreateConfigurationAggregator` | PUT | `/v1/resource-manager/domains/{domain_id}/aggregators` | 创建资源聚合器 |
| `CreateConformancePack` | POST | `/v1/resource-manager/domains/{domain_id}/conformance-packs` | 创建合规规则包 |
| `CreateOrganizationConformancePack` | POST | `/v1/resource-manager/organizations/{organization_id}/conformance-packs` | 创建组织合规规则包 |
| `CreateOrganizationPolicyAssignment` | PUT | `/v1/resource-manager/organizations/{organization_id}/policy-assignments` | 创建组织合规规则 |
| `CreateOrUpdateRemediationConfiguration` | PUT | `/v1/resource-manager/domains/{domain_id}/policy-assignments/{policy_assignment_id}/remediation-configuration` | 创建或更新修正配置 |
| `CreatePolicyAssignments` | PUT | `/v1/resource-manager/domains/{domain_id}/policy-assignments` | 创建合规规则 |
| `CreateStoredQuery` | POST | `/v1/resource-manager/domains/{domain_id}/stored-queries` | 创建高级查询 |
| `CreateTrackerConfig` | PUT | `/v1/resource-manager/domains/{domain_id}/tracker-config` | 创建或更新记录器 |
| `DeleteAggregationAuthorization` | DELETE | `/v1/resource-manager/domains/{domain_id}/aggregators/aggregation-authorization/{authorized_account_id}` | 删除资源聚合器授权 |
| `DeleteConfigurationAggregator` | DELETE | `/v1/resource-manager/domains/{domain_id}/aggregators/{aggregator_id}` | 删除资源聚合器 |
| `DeleteConformancePack` | DELETE | `/v1/resource-manager/domains/{domain_id}/conformance-packs/{conformance_pack_id}` | 删除合规规则包 |
| `DeleteOrganizationConformancePack` | DELETE | `/v1/resource-manager/organizations/{organization_id}/conformance-packs/{conformance_pack_id}` | 删除组织合规规则包 |
| `DeleteOrganizationPolicyAssignment` | DELETE | `/v1/resource-manager/organizations/{organization_id}/policy-assignments/{organization_policy_assignment_id}` | 删除组织合规规则 |
| `DeletePendingAggregationRequest` | DELETE | `/v1/resource-manager/domains/{domain_id}/aggregators/pending-aggregation-request/{requester_account_id}` | 删除聚合器帐号中挂起的授权请求 |
| `DeletePolicyAssignment` | DELETE | `/v1/resource-manager/domains/{domain_id}/policy-assignments/{policy_assignment_id}` | 删除合规规则 |
| `DeleteRemediationConfiguration` | DELETE | `/v1/resource-manager/domains/{domain_id}/policy-assignments/{policy_assignment_id}/remediation-configuration` | 删除修正配置 |
| `DeleteStoredQuery` | DELETE | `/v1/resource-manager/domains/{domain_id}/stored-queries/{query_id}` | 删除高级查询 |

... and 76 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
