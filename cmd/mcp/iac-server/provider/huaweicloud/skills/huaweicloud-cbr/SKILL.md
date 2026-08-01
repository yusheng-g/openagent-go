---
name: huaweicloud-cbr
description: HuaweiCloud CBR API guide. 73 APIs covering 任务, 可保护性, 备份, 备份共享, 存储库.
---

# HuaweiCloud CBR API Guide

73 APIs. Tags: 任务, 可保护性, 备份, 备份共享, 存储库, 文件应用备份, 标签, 特性查询, 策略, 组织策略, 计量, 运营, 还原点, 项目

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddAgentPath` | POST | `/v3/{project_id}/agents/{agent_id}/add-path` | 新增备份路径 |
| `AddMember` | POST | `/v3/{project_id}/backups/{backup_id}/members` | 添加备份成员 |
| `AddVaultResource` | POST | `/v3/{project_id}/vaults/{vault_id}/addresources` | 添加资源 |
| `AssociateVaultPolicy` | POST | `/v3/{project_id}/vaults/{vault_id}/associatepolicy` | 设置存储库策略 |
| `BatchCreateAndDeleteVaultTags` | POST | `/v3/{project_id}/vault/{vault_id}/tags/action` | 批量添加删除存储库资源标签 |
| `BatchUpdateVault` | PUT | `/v3/{project_id}/vaults/batch-update` | 批量修改存储库 |
| `ChangeOrder` | POST | `/v3/{project_id}/orders/change` | 变更 |
| `ChangeVaultChargeMode` | POST | `/v3/{project_id}/vaults/change-charge-mode` | 修改付费模式 |
| `CheckAgent` | POST | `/v3/{project_id}/agent/check` | 查询agent状态 |
| `CopyBackup` | POST | `/v3/{project_id}/backups/{backup_id}/replicate` | 复制备份 |
| `CopyCheckpoint` | POST | `/v3/{project_id}/checkpoints/replicate` | 复制备份还原点 |
| `CreateCheckpoint` | POST | `/v3/{project_id}/checkpoints` | 创建备份还原点 |
| `CreateOrganizationPolicy` | POST | `/v3/{project_id}/organization-policies` | 创建组织策略 |
| `CreatePolicy` | POST | `/v3/{project_id}/policies` | 创建策略 |
| `CreatePostPaidVault` | POST | `/v3/{project_id}/vaults/order` | 创建包周期存储库 |
| `CreateVault` | POST | `/v3/{project_id}/vaults` | 创建存储库 |
| `CreateVaultTags` | POST | `/v3/{project_id}/vault/{vault_id}/tags` | 添加存储库资源标签 |
| `DeleteBackup` | DELETE | `/v3/{project_id}/backups/{backup_id}` | 删除备份 |
| `DeleteMember` | DELETE | `/v3/{project_id}/backups/{backup_id}/members/{member_id}` | 删除指定备份成员 |
| `DeleteOrganizationPolicy` | DELETE | `/v3/{project_id}/organization-policies/{organization_policy_id}` | 删除组织策略 |
| `DeletePolicy` | DELETE | `/v3/{project_id}/policies/{policy_id}` | 删除策略 |
| `DeleteVault` | DELETE | `/v3/{project_id}/vaults/{vault_id}` | 删除存储库 |
| `DeleteVaultTag` | DELETE | `/v3/{project_id}/vault/{vault_id}/tags/{key}` | 删除存储库资源标签 |
| `DisassociateVaultPolicy` | POST | `/v3/{project_id}/vaults/{vault_id}/dissociatepolicy` | 解除存储库策略 |
| `ImportBackup` | POST | `/v3/{project_id}/backups/sync` | 同步备份 |
| `ImportCheckpoint` | POST | `/v3/{project_id}/checkpoints/sync` | 同步备份还原点 |
| `ListAgent` | GET | `/v3/{project_id}/agents` | 查询客户端列表 |
| `ListBackups` | GET | `/v3/{project_id}/backups` | 查询所有备份 |
| `ListDomainProjects` | GET | `/v3/domain/{domain_name}/projects` | 查询租户项目列表 |
| `ListExternalVault` | GET | `/v3/{project_id}/vaults/external` | 查询其他区域存储库列表 |

... and 43 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
