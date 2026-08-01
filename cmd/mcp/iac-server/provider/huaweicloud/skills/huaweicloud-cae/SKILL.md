---
name: huaweicloud-cae
description: HuaweiCloud CAE API guide. 56 APIs covering CAE环境访问VPC, 事件通知规则, 云存储, 任务, 凭据.
---

# HuaweiCloud CAE API Guide

56 APIs. Tags: CAE环境访问VPC, 事件通知规则, 云存储, 任务, 凭据, 域名, 委托, 定时启停规则, 应用, 弹性公网IP, 环境, 监控系统, 组件, 组件配置, 证书

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateAgency` | POST | `/v1/{project_id}/cae/agencies` | 创建委托 |
| `CreateApplication` | POST | `/v1/{project_id}/cae/applications` | 创建应用 |
| `CreateCertificate` | POST | `/v1/{project_id}/cae/certificates` | 创建证书 |
| `CreateComponent` | POST | `/v1/{project_id}/cae/applications/{application_id}/components` | 创建组件 |
| `CreateComponentConfiguration` | POST | `/v1/{project_id}/cae/applications/{application_id}/components/{component_id}/configurations` | 创建组件配置 |
| `CreateComponentWithConfiguration` | POST | `/v1/{project_id}/cae/applications/{application_id}/component-with-configurations` | 创建、生效配置并部署组件 |
| `CreateDomain` | POST | `/v1/{project_id}/cae/domains` | 创建域名 |
| `CreateEnvironment` | POST | `/v1/{project_id}/cae/environments` | 创建环境 |
| `CreateMonitorSystem` | POST | `/v1/{project_id}/cae/monitor-system` | 创建监控系统配置 |
| `CreateNoticeRule` | POST | `/v1/{project_id}/cae/notice-rules` | 创建事件通知规则。 |
| `CreateSecret` | POST | `/v1/{project_id}/cae/dew-secrets` | 关联租户已注册的凭据。 |
| `CreateTimerRule` | POST | `/v1/{project_id}/cae/timer-rules` | 创建定时启停规则 |
| `CreateVolume` | POST | `/v1/{project_id}/cae/volumes` | 授权云存储 |
| `CreateVpcEgress` | POST | `/v1/{project_id}/cae/vpc-egress` | 创建CAE环境访问VPC配置 |
| `DeleteApplication` | DELETE | `/v1/{project_id}/cae/applications/{application_id}` | 删除应用 |
| `DeleteCertificate` | DELETE | `/v1/{project_id}/cae/certificates/{certificate_id}` | 删除证书 |
| `DeleteComponent` | DELETE | `/v1/{project_id}/cae/applications/{application_id}/components/{component_id}` | 删除组件 |
| `DeleteComponentConfiguration` | DELETE | `/v1/{project_id}/cae/applications/{application_id}/components/{component_id}/configurations` | 删除组件配置 |
| `DeleteDomain` | DELETE | `/v1/{project_id}/cae/domains/{domain_id}` | 删除域名 |
| `DeleteEnvironment` | DELETE | `/v1/{project_id}/cae/environments/{environment_id}` | 删除环境 |
| `DeleteNoticeRule` | DELETE | `/v1/{project_id}/cae/notice-rules/{rule_id}` | 删除事件通知规则。 |
| `DeleteSecret` | DELETE | `/v1/{project_id}/cae/dew-secrets/{secret_id}` | 删除用户已在DEW服务上注册的凭据。 |
| `DeleteTimerRule` | DELETE | `/v1/{project_id}/cae/timer-rules/{timer_rule_id}` | 删除定时启停规则 |
| `DeleteVolume` | DELETE | `/v1/{project_id}/cae/volumes/{id}` | 解绑云存储 |
| `DeleteVpcEgress` | DELETE | `/v1/{project_id}/cae/vpc-egress/{vpc_egress_id}` | 删除CAE环境访问VPC配置 |
| `ExecuteAction` | POST | `/v1/{project_id}/cae/applications/{application_id}/components/{component_id}/action` | 操作组件 |
| `ListAgencies` | GET | `/v1/{project_id}/cae/agencies` | 获取委托列表 |
| `ListApplications` | GET | `/v1/{project_id}/cae/applications` | 获取应用列表 |
| `ListCertificates` | GET | `/v1/{project_id}/cae/certificates` | 获取证书列表 |
| `ListComponentConfigurations` | GET | `/v1/{project_id}/cae/applications/{application_id}/components/{component_id}/configurations` | 获取组件配置列表 |

... and 26 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
