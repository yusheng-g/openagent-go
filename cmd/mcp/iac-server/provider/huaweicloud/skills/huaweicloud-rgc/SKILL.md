---
name: huaweicloud-rgc
description: HuaweiCloud RGC API guide. 51 APIs covering Landing Zone治理, Landing Zone管理, 模板治理, 治理成熟度检测, 组织管理.
---

# HuaweiCloud RGC API Guide

51 APIs. Tags: Landing Zone治理, Landing Zone管理, 模板治理, 治理成熟度检测, 组织管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CheckLaunch` | POST | `/v1/landing-zone/pre-launch-check` | 设置Landing Zone前检查 |
| `CreateAccount` | POST | `/v1/managed-organization/managed-accounts` | 创建账号 |
| `CreateBestPracticeDetect` | POST | `/v1/best-practice/detect` | 检测八大场景治理成熟度 |
| `CreateManagedOrganizationalUnit` | POST | `/v1/managed-organization/managed-organizational-units` | 创建OU |
| `CreateTemplate` | POST | `/v1/rgc/templates` | 创建模板 |
| `DeleteLandingZone` | POST | `/v1/landing-zone/delete` | 删除Landing Zone |
| `DeleteManagedOrganizationalUnits` | DELETE | `/v1/managed-organization/managed-organizational-units/{managed_organizational_unit_id}` | 删除注册OU |
| `DeleteTemplate` | DELETE | `/v1/rgc/templates/{template_name}` | 删除模板 |
| `DeregisterOrganizationalUnit` | POST | `/v1/managed-organization/managed-organizational-units/{managed_organizational_unit_id}/de-register` | 取消注册OU |
| `DisableControl` | POST | `/v1/governance/controls/disable` | 关闭控制策略 |
| `EnableControl` | POST | `/v1/governance/controls/enable` | 开启控制策略 |
| `EnrollAccount` | POST | `/v1/managed-organization/accounts/{managed_account_id}/enroll` | 纳管账号 |
| `ListConfigRuleCompliances` | GET | `/v1/governance/managed-accounts/{managed_account_id}/config-rule-compliances` | 查询纳管账号的Config规则合规性信息 |
| `ListControls` | GET | `/v1/governance/controls` | 列出控制策略 |
| `ListControlsForAccount` | GET | `/v1/governance/managed-accounts/{managed_account_id}/controls` | 列出纳管账号下开启的控制策略 |
| `ListControlsForOrganizationalUnit` | GET | `/v1/governance/managed-organizational-units/{managed_organizational_unit_id}/controls` | 列出注册OU下开启的控制策略 |
| `ListControlViolations` | GET | `/v1/governance/control-violations` | 列出不合规信息 |
| `ListDriftDetails` | GET | `/v1/governance/drift-details` | 列出漂移信息 |
| `ListEnabledControls` | GET | `/v1/governance/enabled-controls` | 列出开启的控制策略 |
| `ListExternalConfigRuleCompliances` | GET | `/v1/governance/managed-accounts/{managed_account_id}/external-config-rule-compliances` | 查询纳管账号的外部Config规则合规性信息 |
| `ListManagedAccounts` | GET | `/v1/managed-organization/managed-accounts` | 列举控制策略生效的纳管账号信息 |
| `ListManagedAccountsForParent` | GET | `/v1/managed-organization/managed-organizational-units/{managed_organizational_unit_id}/managed-accounts` | 列出注册OU下的纳管账号信息 |
| `ListManagedOrganizationalUnits` | GET | `/v1/managed-organization/managed-organizational-units` | 列举控制策略生效的注册OU信息 |
| `ListOperation` | GET | `/v1/managed-organization` | 查询已注册OU和纳管账号操作过程信息列表 |
| `ListPredefinedTemplates` | GET | `/v1/rgc/predefined-templates` | 查询预置模板列表 |
| `RegisterOrganizationalUnit` | POST | `/v1/managed-organization/organizational-units/{organizational_unit_id}/register` | 注册OU |
| `ReRegisterOrganizationalUnit` | POST | `/v1/managed-organization/organizational-units/{organizational_unit_id}/re-register` | 重新注册OU |
| `SetupLandingZone` | POST | `/v1/landing-zone/setup` | 设置Landing Zone |
| `ShowAvailableUpdates` | GET | `/v1/landing-zone/available-updates` | 查询Landing Zone可更新状态 |
| `ShowBestPracticeAccountInfo` | GET | `/v1/best-practice/account-info` | 查询治理成熟度的账号详情 |

... and 21 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
