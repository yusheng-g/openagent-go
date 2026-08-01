---
name: huaweicloud-cse
description: HuaweiCloud CSE API guide. 35 APIs covering gateway, nacos, 引擎管理, 治理, 配置管理.
---

# HuaweiCloud CSE API Guide

35 APIs. Tags: gateway, nacos, 引擎管理, 治理, 配置管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateEngine` | POST | `/v2/{project_id}/enginemgr/engines` | 创建微服务引擎 |
| `CreateGovernancePolicy` | POST | `/v3/{project_id}/govern/governance/{kind}` | 创建治理策略 |
| `CreateHttp2Rpc` | POST | `/v2/{project_id}/enginemgr/gateways/{gateway_id}/http2Rpcs` | 创建http转rpc方法 |
| `CreateMicroserviceRouteRule` | PUT | `/v3/{project_id}/govern/route-rule/microservices/{service_name}` | 创建灰度发布策略 |
| `CreateNacosNamespaces` | POST | `/v1/{project_id}/nacos/v1/console/namespaces` | 创建nacos命名空间 |
| `CreatePlugin` | POST | `/v2/{project_id}/enginemgr/gateways/{gateway_id}/plugins` | 创建插件 |
| `DeleteEngine` | DELETE | `/v2/{project_id}/enginemgr/engines/{engine_id}` | 删除微服务引擎 |
| `DeleteGovernancePolicy` | DELETE | `/v3/{project_id}/govern/governance/{kind}/{policy_id}` | 删除治理策略 |
| `DeleteHttp2Rpc` | DELETE | `/v2/{project_id}/enginemgr/gateways/{gateway_id}/http2Rpcs/{http2Rpc_id}` | 删除http转rpc方法 |
| `DeleteMicroserviceRouteRule` | DELETE | `/v3/{project_id}/govern/route-rule/microservices/{service_name}` | 删除灰度发布策略 |
| `DeleteNacosNamespaces` | DELETE | `/v1/{project_id}/nacos/v1/console/namespaces` | 删除nacos命名空间 |
| `DeletePlugin` | DELETE | `/v2/{project_id}/enginemgr/gateways/{gateway_id}/plugins/{plugin_id}` | 删除插件 |
| `DownloadKie` | POST | `/v1/{project_id}/kie/download` | 导出kie配置 |
| `ListEngines` | GET | `/v2/{project_id}/enginemgr/engines` | 查询微服务引擎列表 |
| `ListFlavors` | GET | `/v2/{project_id}/enginemgr/flavors` | 查询微服务引擎的规格列表 |
| `ListGovernancePolicy` | GET | `/v3/{project_id}/govern/governance/{kind}` | 查询指定类型治理策略列表 |
| `ListGovernancePolicyByPolicyId` | GET | `/v3/{project_id}/govern/governance/{kind}/{policy_id}` | 查询治理策略详情 |
| `ListGovernancePolicys` | GET | `/v3/{project_id}/govern/governance/display` | 查询治理策略列表 |
| `ListMicroserviceRouteRule` | GET | `/v3/{project_id}/govern/route-rule/microservices/{service_name}` | 查询微服务的灰度发布规则 |
| `ListNacosNamespaces` | GET | `/v1/{project_id}/nacos/v1/console/namespaces` | 查询nacos命名空间 |
| `ModifyHttp2Rpc` | PUT | `/v2/{project_id}/enginemgr/gateways/{gateway_id}/http2Rpcs/{http2Rpc_id}` | 修改http转rpc方法 |
| `ModifyPlugin` | PUT | `/v2/{project_id}/enginemgr/gateways/{gateway_id}/plugins/{plugin_id}` | 修改插件 |
| `ResizeEngine` | PUT | `/v2/{project_id}/enginemgr/engines/{engine_id}/resize` | 变更微服务引擎规格 |
| `RetryEngine` | PUT | `/v2/{project_id}/enginemgr/engines/{engine_id}/actions` | 对微服务引擎进行重试 |
| `ShowEngine` | GET | `/v2/{project_id}/enginemgr/engines/{engine_id}` | 查询微服务引擎详情 |
| `ShowEngineJob` | GET | `/v2/{project_id}/enginemgr/engines/{engine_id}/jobs/{job_id}` | 查询微服务引擎任务详情 |
| `ShowEngineQuotas` | GET | `/v2/{project_id}/enginemgr/quotas` | 查询微服务引擎配额 |
| `ShowHttp2Rpcs` | GET | `/v2/{project_id}/enginemgr/gateways/{gateway_id}/http2Rpcs` | 查询http2rpc资源列表 |
| `ShowPlugins` | GET | `/v2/{project_id}/enginemgr/gateways/{gateway_id}/plugins` | 查询插件列表 |
| `ShowSinglePlugin` | GET | `/v2/{project_id}/enginemgr/gateways/{gateway_id}/plugins/{plugin_id}` | 查询单个插件 |

... and 5 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
