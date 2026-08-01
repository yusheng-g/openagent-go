---
name: huaweicloud-organizations
description: HuaweiCloud Organizations API guide. 69 APIs covering 其他, 可信服务管理, 委托管理员管理, 标签管理, 策略管理.
---

# HuaweiCloud Organizations API Guide

69 APIs. Tags: 其他, 可信服务管理, 委托管理员管理, 标签管理, 策略管理, 策略试运行配置, 组织单元管理, 组织管理, 试运行策略管理, 账号管理, 邀请管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptHandshake` | POST | `/v1/received-handshakes/{handshake_id}/accept` | 接受邀请 |
| `AttachDryRunPolicy` | POST | `/v1/organizations/dry-run-policies/{policy_id}/attach` | 将试运行策略跟实体绑定 |
| `AttachPolicy` | POST | `/v1/organizations/policies/{policy_id}/attach` | 将策略跟实体绑定 |
| `CancelHandshake` | POST | `/v1/organizations/handshakes/{handshake_id}/cancel` | 取消邀请 |
| `CloseAccount` | POST | `/v1/organizations/accounts/{account_id}/close` | 关闭账号 |
| `CreateAccount` | POST | `/v1/organizations/accounts` | 创建账号 |
| `CreateDryRunPolicy` | POST | `/v1/organizations/dry-run-policies` | 创建试运行策略 |
| `CreateOrganization` | POST | `/v1/organizations` | 创建组织 |
| `CreateOrganizationalUnit` | POST | `/v1/organizations/organizational-units` | 创建组织单元 |
| `CreatePolicy` | POST | `/v1/organizations/policies` | 创建策略 |
| `CreateResourceAccount` | POST | `/v2/organizations/accounts` | 创建帐号 |
| `CreateTagResource` | POST | `/v1/organizations/{resource_type}/{resource_id}/tags/create` | 为指定资源类型添加标签 |
| `DeclineHandshake` | POST | `/v1/received-handshakes/{handshake_id}/decline` | 拒绝邀请 |
| `DeleteDryRunPolicy` | DELETE | `/v1/organizations/dry-run-policies/{policy_id}` | 删除试运行策略 |
| `DeleteOrganization` | DELETE | `/v1/organizations` | 删除组织 |
| `DeleteOrganizationalUnit` | DELETE | `/v1/organizations/organizational-units/{organizational_unit_id}` | 删除组织单元 |
| `DeletePolicy` | DELETE | `/v1/organizations/policies/{policy_id}` | 删除策略 |
| `DeleteTagResource` | POST | `/v1/organizations/{resource_type}/{resource_id}/tags/delete` | 从指定资源类型中删除指定主键标签 |
| `DeregisterDelegatedAdministrator` | POST | `/v1/organizations/delegated-administrators/deregister` | 注销服务的委托管理员 |
| `DetachDryRunPolicy` | POST | `/v1/organizations/dry-run-policies/{policy_id}/detach` | 将试运行策略跟实体解绑 |
| `DetachPolicy` | POST | `/v1/organizations/policies/{policy_id}/detach` | 将策略跟实体解绑 |
| `DisablePolicyType` | POST | `/v1/organizations/policies/disable` | 禁用根中的策略类型 |
| `DisableTrustedService` | POST | `/v1/organizations/trusted-services/disable` | 禁用受信任服务 |
| `EnablePolicyType` | POST | `/v1/organizations/policies/enable` | 在根中启用策略类型 |
| `EnableTrustedService` | POST | `/v1/organizations/trusted-services/enable` | 启用可信服务 |
| `InviteAccount` | POST | `/v1/organizations/accounts/invite` | 邀请账号加入组织 |
| `LeaveOrganization` | POST | `/v1/organizations/leave` | 离开当前组织 |
| `ListAccounts` | GET | `/v1/organizations/accounts` | 列出组织中的账号 |
| `ListCloseAccountStatuses` | GET | `/v1/organizations/close-account-status` | 列出关闭账号的状态 |
| `ListCreateAccountStatuses` | GET | `/v1/organizations/create-account-status` | 列出创建账号的状态 |

... and 39 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
