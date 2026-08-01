---
name: huaweicloud-codeartsrepo
description: HuaweiCloud CodeArtsRepo API guide. 100 APIs covering Commit, Discussion, File, Group, Member.
---

# HuaweiCloud CodeArtsRepo API Guide

100 APIs. Tags: Commit, Discussion, File, Group, Member, MergeRequest, Permission, Project, ProtectedRefs, Refs, RepoMember, Repository, SSHKey, Tenant, ThirdParty, User, V2Project, WebHook, v2仓库管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddDeployKey` | POST | `/v1/repositories/{repository_id}/deploy_keys` | 添加部署密钥 |
| `AddDeployKeyV2` | POST | `/v2/repositories/{repository_id}/deploy-keys` | 添加部署密钥 |
| `AddGroupWebhook` | POST | `/v4/groups/{group_id}/hooks` | 添加代码组下Webhook |
| `AddHooks` | POST | `/v1/repositories/{group_name}/{repository_name}/hooks` | 为指定仓库添加hook |
| `AddProjectWebhook` | POST | `/v4/projects/{project_id}/hooks` | 添加项目下Webhook |
| `AddProtectBranchV2` | PUT | `/v2/repositories/{repository_id}/branch/{branch_name}/protect` | 新建保护分支 |
| `AddRepoMembers` | POST | `/v1/repositories/{repository_uuid}/members` | 添加仓库成员 |
| `AddRepositoryMembers` | POST | `/v4/repositories/{repository_id}/members` | 批量添加仓库成员 |
| `AddRepositoryWebhook` | POST | `/v4/repositories/{repository_id}/hooks` | 添加仓库下Webhook |
| `AddSshKey` | POST | `/v1/users/sshkey` | 添加ssh key |
| `AddSubmodule` | POST | `/v4/repositories/{repository_id}/repository/submodules` | 创建子模块 |
| `AddTagV2` | POST | `/v2/repositories/{repository_id}/tags` | 新建标签 |
| `AddTenantTrustedIpAddress` | POST | `/v4/tenant/trusted-ip-addresses` | 添加租户ip白名单 |
| `AddTrustedIpAddress` | POST | `/v4/projects/{id}/trusted-ip-addresses` | 添加仓库ip白名单 |
| `ApprovalMergeRequest` | PUT | `/v4/repositories/{repository_id}/merge-requests/{merge_request_iid}/approval` | 审核合并请求 |
| `AssociateGroupUserGroup` | POST | `/v4/{project_id}/groups/{group_id}/user-group/{user_group_id}` | 关联代码组与成员组 |
| `AssociateIssues` | POST | `/v2/projects/issues` | 分支关联工作项 |
| `AssociateRemoteMirror` | POST | `/v4/repositories/{repository_id}/remote-mirror/associate` | 将普通仓库与远程镜像关联 |
| `AssociateRepositoryUserGroup` | POST | `/v4/{project_id}/repositories/{repository_id}/user-group/{user_group_id}` | 关联仓库与成员组 |
| `BatchCreateProtectedBranch` | POST | `/v4/repositories/{repository_id}/protected-branches` | 批量创建仓库保护分支 |
| `BatchCreateProtectedTags` | POST | `/v4/repositories/{repository_id}/protected-tags` | 批量创建仓库保护Tag |
| `BatchDeleteBranch` | POST | `/v4/repositories/{repository_id}/branches/batch-delete` | 批量删除分支 |
| `BatchDeleteProtectedBranches` | POST | `/v4/repositories/{repository_id}/protected-branches/bulk-deletion` | 批量删除仓库保护分支 |
| `BatchDeleteProtectedTags` | POST | `/v4/repositories/{repository_id}/protected-tags/bulk-deletion` | 批量删除仓库保护Tag |
| `BatchDeleteRepositoryFilePushPermissions` | POST | `/v4/repositories/{repository_id}/file-push-permissions/batch-delete` | 批量删除仓库文件推送权限 |
| `BatchUpdateProtectedBranches` | PUT | `/v4/repositories/{repository_id}/protected-branches` | 批量更新仓库保护分支 |
| `BatchUpdateProtectedTags` | PUT | `/v4/repositories/{repository_id}/protected-tags` | 批量更新仓库保护Tag |
| `BatchUpdateRepositoryFilePushPermissions` | PUT | `/v4/repositories/{repository_id}/file-push-permissions` | 批量更新仓库文件推送权限 |
| `BatchValidateRepoNames` | POST | `/v4/repository-names/validations` | 批量检查仓库名 |
| `BatchValidateUserGroupPermissions` | POST | `/v4/user/groups/group-permissions` | 获取当前用户指定的代码组列表中的权限 |

... and 70 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
