---
name: huaweicloud-apig
description: HuaweiCloud APIG API guide. 277 APIs covering 专享版-ACL策略管理, 专享版-API分组管理, 专享版-API管理, 专享版-API绑定ACL策略, 专享版-API绑定流控策略.
---

# HuaweiCloud APIG API Guide

277 APIs. Tags: 专享版-ACL策略管理, 专享版-API分组管理, 专享版-API管理, 专享版-API绑定ACL策略, 专享版-API绑定流控策略, 专享版-APP授权管理, 专享版-OpenAPI接口, 专享版-SSL证书管理, 专享版-VPC通道管理, 专享版-凭据管理, 专享版-凭据配额管理, 专享版-分组自定义响应管理, 专享版-域名管理, 专享版-实例标签管理, 专享版-实例特性管理, 专享版-实例管理, 专享版-实例终端节点管理, 专享版-实例自定义入方向端口管理, 专享版-异步任务管理, 专享版-微服务中心管理, 专享版-插件管理, 专享版-标签管理, 专享版-概要查询, 专享版-流控策略管理, 专享版-环境变量管理, 专享版-环境管理, 专享版-监控信息查询, 专享版-签名密钥管理, 专享版-签名密钥绑定关系管理, 专享版-编排规则管理, 专享版-自定义认证管理, 专享版-设置特殊流控, 专享版-配置管理, 共享版-API分组管理, 共享版-API管理, 共享版-API绑定流控策略, 共享版-APP授权管理, 共享版-APP管理, 共享版-域名管理, 共享版-概要查询, 共享版-流控策略管理, 共享版-环境变量管理, 共享版-环境管理, 共享版-签名密钥管理, 共享版-签名密钥绑定关系管理, 共享版-设置特殊流控

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptOrRejectEndpointConnections` | POST | `/{project_id}/apigw/instances/{instance_id}/vpc-endpoint/connections/action` | 接受或拒绝终端节点连接 |
| `AddCustomIngressPort` | POST | `/{project_id}/apigw/instances/{instance_id}/custom-ingress-ports` | 新增实例的自定义入方向端口 |
| `AddEipV2` | PUT | `/{project_id}/apigw/instances/{instance_id}/eip` | 实例更新或绑定EIP |
| `AddEndpointPermissions` | POST | `/{project_id}/apigw/instances/{instance_id}/vpc-endpoint/permissions/batch-add` | 批量添加实例终端节点连接白名单 |
| `AddEngressEipV2` | POST | `/{project_id}/apigw/instances/{instance_id}/nat-eip` | 开启实例公网出口 |
| `AddingBackendInstancesV2` | POST | `/{project_id}/apigw/instances/{instance_id}/vpc-channels/{vpc_channel_id}/members` | 添加或更新后端实例 |
| `AddIngressEipV2` | POST | `/{project_id}/apigw/instances/{instance_id}/ingress-eip` | 开启实例公网入口 |
| `AssociateAppsForAppQuota` | POST | `/{project_id}/apigw/instances/{instance_id}/app-quotas/{app_quota_id}/binding-apps` | 凭据配额绑定凭据列表 |
| `AssociateCertificate` | POST | `/apigw/api-groups/{group_id}/domains/{domain_id}/certificate` | 绑定域名证书 |
| `AssociateCertificateV2` | POST | `/{project_id}/apigw/instances/{instance_id}/api-groups/{group_id}/domains/{domain_id}/certificate` | 绑定域名证书 |
| `AssociateDomain` | POST | `/apigw/api-groups/{group_id}/domains` | 绑定域名 |
| `AssociateDomainV2` | POST | `/{project_id}/apigw/instances/{instance_id}/api-groups/{group_id}/domains` | 绑定域名 |
| `AssociateRequestThrottlingPolicy` | POST | `/apigw/throttle-bindings` | 绑定流控策略 |
| `AssociateRequestThrottlingPolicyV2` | POST | `/{project_id}/apigw/instances/{instance_id}/throttle-bindings` | 绑定流控策略 |
| `AssociateSignatureKey` | POST | `/apigw/sign-bindings` | 绑定签名密钥 |
| `AssociateSignatureKeyV2` | POST | `/{project_id}/apigw/instances/{instance_id}/sign-bindings` | 绑定签名密钥 |
| `AttachApiToPlugin` | POST | `/{project_id}/apigw/instances/{instance_id}/plugins/{plugin_id}/attach` | 插件绑定API |
| `AttachPluginToApi` | POST | `/{project_id}/apigw/instances/{instance_id}/apis/{api_id}/plugins/attach` | API绑定插件 |
| `BatchAssociateCertsV2` | POST | `/{project_id}/apigw/instances/{instance_id}/api-groups/{group_id}/domains/{domain_id}/certificates/attach` | 域名绑定SSL证书 |
| `BatchAssociateDomainsV2` | POST | `/{project_id}/apigw/certificates/{certificate_id}/domains/attach` | SSL证书绑定域名 |
| `BatchCreateOrDeleteInstanceTags` | POST | `/{project_id}/apigw/instances/{instance_id}/instance-tags/action` | 批量添加或删除单个实例的标签 |
| `BatchDeleteAclV2` | PUT | `/{project_id}/apigw/instances/{instance_id}/acls` | 批量删除ACL策略 |
| `BatchDeleteApiAclBindingV2` | PUT | `/{project_id}/apigw/instances/{instance_id}/acl-bindings` | 批量解除API与ACL策略的绑定 |
| `BatchDisableMembers` | POST | `/{project_id}/apigw/instances/{instance_id}/vpc-channels/{vpc_channel_id}/members/batch-disable` | 批量修改后端服务器状态不可用 |
| `BatchDisassociateCertsV2` | POST | `/{project_id}/apigw/instances/{instance_id}/api-groups/{group_id}/domains/{domain_id}/certificates/detach` | 域名解绑SSL证书 |
| `BatchDisassociateDomainsV2` | POST | `/{project_id}/apigw/certificates/{certificate_id}/domains/detach` | SSL证书解绑域名 |
| `BatchDisassociateThrottlingPolicyV2` | PUT | `/{project_id}/apigw/instances/{instance_id}/throttle-bindings` | 批量解绑流控策略 |
| `BatchEnableMembers` | POST | `/{project_id}/apigw/instances/{instance_id}/vpc-channels/{vpc_channel_id}/members/batch-enable` | 批量修改后端服务器状态可用 |
| `BatchPublishOrOfflineApiV2` | POST | `/{project_id}/apigw/instances/{instance_id}/apis/publish` | 批量发布或下线API |
| `CancelingAuthorization` | DELETE | `/apigw/app-auths/{app_auth_id}` | 解除授权 |

... and 247 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
