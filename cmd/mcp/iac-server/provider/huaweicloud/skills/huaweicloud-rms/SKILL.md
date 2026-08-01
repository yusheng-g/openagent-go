---
name: huaweicloud-rms
description: HuaweiCloud RMS API guide. 62 APIs covering 区域管理, 合规性, 聚合器, 资源关系, 资源历史.
---

# HuaweiCloud RMS API Guide

62 APIs. Tags: 区域管理, 合规性, 聚合器, 资源关系, 资源历史, 资源清单, 资源记录器, 高级查询

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CollectAllResourcesSummary` | GET | `/v1/resource-manager/domains/{domain_id}/all-resources/summary` | 列举资源概要 |
| `CountAllResources` | GET | `/v1/resource-manager/domains/{domain_id}/all-resources/count` | 查询资源数量 |
| `CreateAggregationAuthorization` | PUT | `/v1/resource-manager/domains/{domain_id}/aggregators/aggregation-authorization` | 创建资源聚合器授权 |
| `CreateConfigurationAggregator` | PUT | `/v1/resource-manager/domains/{domain_id}/aggregators` | 创建资源聚合器 |
| `CreateOrganizationPolicyAssignment` | PUT | `/v1/resource-manager/organizations/{organization_id}/policy-assignments` | 创建或更新组织合规规则 |
| `CreatePolicyAssignments` | PUT | `/v1/resource-manager/domains/{domain_id}/policy-assignments` | 创建合规规则 |
| `CreateStoredQuery` | POST | `/v1/resource-manager/domains/{domain_id}/stored-queries` | 创建高级查询 |
| `CreateTrackerConfig` | PUT | `/v1/resource-manager/domains/{domain_id}/tracker-config` | 创建或更新记录器 |
| `DeleteAggregationAuthorization` | DELETE | `/v1/resource-manager/domains/{domain_id}/aggregators/aggregation-authorization/{authorized_account_id}` | 删除资源聚合器授权 |
| `DeleteConfigurationAggregator` | DELETE | `/v1/resource-manager/domains/{domain_id}/aggregators/{aggregator_id}` | 删除资源聚合器 |
| `DeleteOrganizationPolicyAssignment` | DELETE | `/v1/resource-manager/organizations/{organization_id}/policy-assignments/{organization_policy_assignment_id}` | 删除组织合规规则 |
| `DeletePendingAggregationRequest` | DELETE | `/v1/resource-manager/domains/{domain_id}/aggregators/pending-aggregation-request/{requester_account_id}` | 删除聚合器帐号中挂起的授权请求 |
| `DeletePolicyAssignment` | DELETE | `/v1/resource-manager/domains/{domain_id}/policy-assignments/{policy_assignment_id}` | 删除合规规则 |
| `DeleteStoredQuery` | DELETE | `/v1/resource-manager/domains/{domain_id}/stored-queries/{query_id}` | 删除高级查询 |
| `DeleteTrackerConfig` | DELETE | `/v1/resource-manager/domains/{domain_id}/tracker-config` | 删除记录器 |
| `DisablePolicyAssignment` | POST | `/v1/resource-manager/domains/{domain_id}/policy-assignments/{policy_assignment_id}/disable` | 停用合规规则 |
| `EnablePolicyAssignment` | POST | `/v1/resource-manager/domains/{domain_id}/policy-assignments/{policy_assignment_id}/enable` | 启用合规规则 |
| `ListAggregateComplianceByPolicyAssignment` | POST | `/v1/resource-manager/domains/{domain_id}/aggregators/aggregate-data/policy-assignments/compliance` | 查询聚合合规规则列表 |
| `ListAggregateDiscoveredResources` | POST | `/v1/resource-manager/domains/{domain_id}/aggregators/aggregate-data/aggregate-discovered-resources` | 查询聚合器中资源的列表 |
| `ListAggregationAuthorizations` | GET | `/v1/resource-manager/domains/{domain_id}/aggregators/aggregation-authorization` | 查询资源聚合器授权列表 |
| `ListAllResources` | GET | `/v1/resource-manager/domains/{domain_id}/all-resources` | 列举所有资源 |
| `ListAllTags` | GET | `/v1/resource-manager/domains/{domain_id}/all-resources/tags` | 列举资源标签 |
| `ListBuiltInPolicyDefinitions` | GET | `/v1/resource-manager/policy-definitions` | 列出内置策略 |
| `ListConfigurationAggregators` | GET | `/v1/resource-manager/domains/{domain_id}/aggregators` | 查询资源聚合器列表 |
| `ListOrganizationPolicyAssignments` | GET | `/v1/resource-manager/organizations/{organization_id}/policy-assignments` | 查询组织合规规则列表 |
| `ListPendingAggregationRequests` | GET | `/v1/resource-manager/domains/{domain_id}/aggregators/pending-aggregation-request` | 查询所有挂起的聚合请求列表 |
| `ListPolicyAssignments` | GET | `/v1/resource-manager/domains/{domain_id}/policy-assignments` | 列出合规规则 |
| `ListPolicyStatesByAssignmentId` | GET | `/v1/resource-manager/domains/{domain_id}/policy-assignments/{policy_assignment_id}/policy-states` | 获取规则的合规结果 |
| `ListPolicyStatesByDomainId` | GET | `/v1/resource-manager/domains/{domain_id}/policy-states` | 获取用户的合规结果 |
| `ListPolicyStatesByResourceId` | GET | `/v1/resource-manager/domains/{domain_id}/resources/{resource_id}/policy-states` | 获取资源的合规结果 |

... and 32 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
