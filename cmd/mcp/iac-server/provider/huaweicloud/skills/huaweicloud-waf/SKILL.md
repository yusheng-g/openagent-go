---
name: huaweicloud-waf
description: HuaweiCloud WAF API guide. 202 APIs covering Web防护规则查询, 事件管理, 云模式域名管理, 云模式防护网站管理, 可防护的资源管理.
---

# HuaweiCloud WAF API Guide

202 APIs. Tags: Web防护规则查询, 事件管理, 云模式域名管理, 云模式防护网站管理, 可防护的资源管理, 告警管理, 地址组管理, 域名dns解析, 安全总览, 安全报告管理, 实例组管理, 局点支持特性查询, 异步任务, 日志配置管理, 独享实例管理, 独享模式防护网站管理, 租户域名查询, 租户域名管理, 租户套餐管理, 租户订购管理, 租户资源, 租户防护域名管理, 策略规则管理, 系统管理, 证书管理, 防护事件管理, 防护策略管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ApplyCertificateToHost` | POST | `/v1/{project_id}/waf/certificate/{certificate_id}/apply-to-hosts` | 绑定证书到域名 |
| `BatchCreateAntileakageRule` | POST | `/v1/{project_id}/waf/rule/antileakage` | 选中多个策略批量添加防敏感信息泄漏规则 |
| `BatchCreateAntiTamperRule` | POST | `/v1/{project_id}/waf/rule/antitamper` | 选中多个策略批量添加网页防篡改规则 |
| `BatchCreateCcRule` | POST | `/v1/{project_id}/waf/rule/cc` | 选中多个策略为批量添加cc规则 |
| `BatchCreateCustomRule` | POST | `/v1/{project_id}/waf/rule/custom` | 选中多个策略批量添加精准防护规则 |
| `BatchCreateGeoIpRule` | POST | `/v1/{project_id}/waf/rule/geoip` | 选中多个策略批量添加地理位置访问控制规则 |
| `BatchCreateIgnoreRule` | POST | `/v1/{project_id}/waf/rule/ignore` | 选中多个策略批量添加全局白名单规则 |
| `BatchCreateIpReputationRule` | POST | `/v1/{project_id}/waf/rule/ip-reputation` | 为多个策略批量添加威胁情报访问控制规则 |
| `BatchCreatePrivacyRule` | POST | `/v1/{project_id}/waf/rule/privacy` | 选中多个策略批量添加隐私屏蔽防护防护规则 |
| `BatchCreateWhiteblackipRule` | POST | `/v1/{project_id}/waf/rule/whiteblackip` | 选中多个策略批量添加黑白名单防护规则 |
| `BatchDeleteAlertNoticeConfig` | POST | `/v2/{project_id}/waf/alert/batch-delete` | 批量删除告警通知 |
| `BatchDeleteCompositeHosts` | POST | `/v1/{project_id}/composite-waf/hosts/batch-delete` | 批量删除租户域名 |
| `BatchDeletePolicies` | POST | `/v1/{project_id}/waf/policies/batch-delete` | 批量删除防护策略 |
| `BatchDeleteRules` | POST | `/v1/{project_id}/waf/rule/{rule_type}/batch-delete` | 批量删除规则 |
| `BatchUpdateAntileakageRules` | POST | `/v1/{project_id}/waf/rule/antileakage/batch-update` | 批量更新防敏感信息泄露规则 |
| `BatchUpdateAntitamperRules` | POST | `/v1/{project_id}/waf/rule/antitamper/batch-update` | 批量更新网页防篡改规则 |
| `BatchUpdateCcRules` | POST | `/v1/{project_id}/waf/rule/cc/batch-update` | 批量修改CC防护规则 |
| `BatchUpdateCustomRules` | POST | `/v1/{project_id}/waf/rule/custom/batch-update` | 批量更新精准防护规则 |
| `BatchUpdateGeoipRules` | POST | `/v1/{project_id}/waf/rule/geoip/batch-update` | 批量修改地理位置访问控制规则 |
| `BatchUpdateIgnoreRules` | POST | `/v1/{project_id}/waf/rule/ignore/batch-update` | 批量更新全局白名单规则 |
| `BatchUpdateIpReputationRules` | POST | `/v1/{project_id}/waf/rule/ip-reputation/batch-update` | 批量更新威胁情报规则 |
| `BatchUpdatePrivacyRules` | POST | `/v1/{project_id}/waf/rule/privacy/batch-update` | 批量更新隐私屏蔽规则 |
| `BatchUpdateWhiteblackipRules` | POST | `/v1/{project_id}/waf/rule/whiteblackip/batch-update` | 批量更新黑白名单设置规则 |
| `ChangePrepaidCloudWaf` | POST | `/v1/{project_id}/waf/subscription/batchalter/prepaid-cloud-waf` | 变更包周期云模式waf规格 |
| `CheckAgency` | GET | `/v1/{project_id}/premium-waf/agency` | 查询独享引擎代理 |
| `ConfirmApplicationTypes` | GET | `/v1/{project_id}/waf/rules/application-types` | 按application规则类型获取内置规则类型 |
| `ConfirmAsyncJob` | GET | `/v1/{project_id}/waf/async-job/{job_id}` | 查询异步任务详情 |
| `ConfirmDnsDomain` | GET | `/v1/{project_id}/waf/dns-domain` | 查询用户托管在云解析上的域名 |
| `ConfirmIpReputationRule` | GET | `/v1/{project_id}/waf/policy/{policy_id}/ip-reputation/{rule_id}` | 根据Id查询威胁情报访问控制规则 |
| `ConfirmPolicyAntileakageMap` | GET | `/v1/{project_id}/waf/tag/antileakage/map` | 查询敏感信息选项的详细信息 |

... and 172 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
