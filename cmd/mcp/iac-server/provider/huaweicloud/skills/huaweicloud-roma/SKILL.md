---
name: huaweicloud-roma
description: HuaweiCloud ROMA API guide. 320 APIs covering ACL策略管理, API分组管理, API管理, API绑定ACL策略, API绑定流控策略.
---

# HuaweiCloud ROMA API Guide

320 APIs. Tags: ACL策略管理, API分组管理, API管理, API绑定ACL策略, API绑定流控策略, APPLICATION_MANAGEMENT, APP授权管理, ASSET_MANAGEMENT, DICTIONARY_MANAGEMENT, INSTANCE_MANAGEMENT, MQS实例管理, OpenAPI接口, PUBLIC_MANAGEMENT, SSL证书管理, VPC通道管理, VPC通道管理-项目级, 主题管理, 产品模板, 产品管理, 任务监控管理, 任务管理, 域名管理, 实例特性管理, 实例管理, 客户端配置, 客户端配额, 应用权限管理, 应用配置管理, 插件管理, 数据源管理, 服务管理, 标签管理, 流控策略管理, 消息管理, 环境变量管理, 环境管理, 监控信息查询, 签名密钥管理, 签名密钥绑定关系管理, 自定义后端服务, 自定义认证管理, 规则引擎, 订阅管理操作, 设备分组管理, 设备管理, 设置特殊流控, 配置管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddingBackendInstancesV2` | POST | `/v2/{project_id}/apic/instances/{instance_id}/vpc-channels/{vpc_channel_id}/members` | 添加后端实例 |
| `AddSubsetsToGateway` | POST | `/v2/{project_id}/link/instances/{instance_id}/devices/{device_id}/subsets` | 添加子设备到网关 |
| `AddUserToApp` | PUT | `/v2/{project_id}/instances/{instance_id}/apps/{app_id}/users` | 设置用户成员 |
| `AssociateAppsForAppQuota` | POST | `/v2/{project_id}/apic/instances/{instance_id}/app-quotas/{app_quota_id}/binding-apps` | 客户端配额绑定客户端应用列表 |
| `AssociateCertificateV2` | POST | `/v2/{project_id}/apic/instances/{instance_id}/api-groups/{group_id}/domains/{domain_id}/certificate` | 绑定域名证书 |
| `AssociateDomainV2` | POST | `/v2/{project_id}/apic/instances/{instance_id}/api-groups/{group_id}/domains` | 绑定域名 |
| `AssociateRequestThrottlingPolicyV2` | POST | `/v2/{project_id}/apic/instances/{instance_id}/throttle-bindings` | 绑定流控策略 |
| `AssociateSignatureKeyV2` | POST | `/v2/{project_id}/apic/instances/{instance_id}/sign-bindings` | 绑定签名密钥 |
| `AttachApiToPlugin` | POST | `/v2/{project_id}/apic/instances/{instance_id}/plugins/{plugin_id}/attach` | 插件绑定API |
| `AttachPluginToApi` | POST | `/v2/{project_id}/apic/instances/{instance_id}/apis/{api_id}/plugins/attach` | API绑定插件 |
| `BatchAddDeviceToGroup` | POST | `/v2/{project_id}/link/instances/{instance_id}/device-groups/{group_id}/devices/batch-add` | 批量添加设备到设备分组 |
| `BatchAssociateCertsV2` | POST | `/v2/{project_id}/apic/instances/{instance_id}/api-groups/{group_id}/domains/{domain_id}/certificates/attach` | 域名绑定SSL证书 |
| `BatchAssociateDomainsV2` | POST | `/v2/{project_id}/apic/certificates/{certificate_id}/domains/attach` | SSL证书绑定域名 |
| `BatchDeleteAclV2` | PUT | `/v2/{project_id}/apic/instances/{instance_id}/acls` | 批量删除ACL策略 |
| `BatchDeleteApiAclBindingV2` | PUT | `/v2/{project_id}/apic/instances/{instance_id}/acl-bindings` | 批量解除API与ACL策略的绑定 |
| `BatchDeleteMqsInstanceTopic` | POST | `/v2/{project_id}/mqs/instances/{instance_id}/topics/delete` | 批量删除Topic |
| `BatchDeleteRules` | POST | `/v2/{project_id}/link/instances/{instance_id}/rules/batch-delete` | 批量删除规则 |
| `BatchDeleteThrottlingPolicyV2` | POST | `/v2/{project_id}/apic/instances/{instance_id}/throttles/batch-delete` | 批量删除流控策略 |
| `BatchDisableMembers` | POST | `/v2/{project_id}/apic/instances/{instance_id}/vpc-channels/{vpc_channel_id}/members/batch-disable` | 批量修改后端服务器状态不可用 |
| `BatchDisassociateCertsV2` | POST | `/v2/{project_id}/apic/instances/{instance_id}/api-groups/{group_id}/domains/{domain_id}/certificates/detach` | 域名解绑SSL证书 |
| `BatchDisassociateDomainsV2` | POST | `/v2/{project_id}/apic/certificates/{certificate_id}/domains/detach` | SSL证书解绑域名 |
| `BatchDisassociateThrottlingPolicyV2` | PUT | `/v2/{project_id}/apic/instances/{instance_id}/throttle-bindings` | 批量解绑流控策略 |
| `BatchEnableMembers` | POST | `/v2/{project_id}/apic/instances/{instance_id}/vpc-channels/{vpc_channel_id}/members/batch-enable` | 批量修改后端服务器状态可用 |
| `BatchFreezeDevices` | POST | `/v2/{project_id}/link/instances/{instance_id}/devices/force-offline` | 设备批量下线 |
| `BatchPublishOrOfflineApiV2` | POST | `/v2/{project_id}/apic/instances/{instance_id}/apis/publish` | 批量发布或下线API |
| `BatchStartOrStopTasks` | POST | `/v2/{project_id}/fdi/instances/{instance_id}/batch-operation/tasks` | 批量启动\停止任务 |
| `CancelingAuthorizationV2` | DELETE | `/v2/{project_id}/apic/instances/{instance_id}/app-auths/{app_auth_id}` | 解除授权 |
| `ChangeApiVersionV2` | PUT | `/v2/{project_id}/apic/instances/{instance_id}/apis/publish/{api_id}` | 切换API版本 |
| `CheckApiGroupsV2` | POST | `/v2/{project_id}/apic/instances/{instance_id}/api-groups/check` | 校验API分组名称是否存在 |
| `CheckApisV2` | POST | `/v2/{project_id}/apic/instances/{instance_id}/apis/check` | 校验API定义 |

... and 290 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
