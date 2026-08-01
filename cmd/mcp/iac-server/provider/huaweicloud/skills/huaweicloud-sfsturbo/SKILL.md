---
name: huaweicloud-sfsturbo
description: HuaweiCloud SFSTurbo API guide. 55 APIs covering 任务管理, 共享标签, 名称管理, 存储联动管理, 文件系统管理.
---

# HuaweiCloud SFSTurbo API Guide

55 APIs. Tags: 任务管理, 共享标签, 名称管理, 存储联动管理, 文件系统管理, 权限管理, 查询文件系统类型和配额, 生命周期管理, 目录管理, 租户配额管理, 运营管理, 连接管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddActiveDirectoryDomain` | POST | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/active-directory-domain` | 加入AD域 |
| `BatchAddSharedTags` | POST | `/v1/{project_id}/sfs-turbo/{share_id}/tags/action` | 批量添加共享标签 |
| `ChangeSecurityGroup` | POST | `/v1/{project_id}/sfs-turbo/shares/{share_id}/action` | 修改文件系统绑定的安全组 |
| `ChangeShareChargeModeV2` | POST | `/v2/{project_id}/sfs-turbo/shares/{share_id}/change-charge-mode` | 修改文件系统计费模式由按需转为包周期 |
| `ChangeShareName` | POST | `/v1/{project_id}/sfs-turbo/shares/{share_id}/action` | 修改文件系统名称 |
| `CreateBackendTarget` | POST | `/v1/{project_id}/sfs-turbo/shares/{share_id}/targets` | 绑定后端存储 |
| `CreateFsDir` | POST | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/dir` | 创建目录 |
| `CreateFsDirQuota` | POST | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/dir-quota` | 创建目标文件夹配额 |
| `CreateFsTask` | POST | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/{feature}/tasks` | 创建文件系统异步任务 |
| `CreateHpcCacheTask` | POST | `/v1/{project_id}/sfs-turbo/{share_id}/hpc-cache/task` | 创建数据导入导出任务 |
| `CreateLdapConfig` | POST | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/ldap` | 创建并绑定LDAP配置 |
| `CreatePermRule` | POST | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/perm-rules` | 创建权限规则 |
| `CreateShare` | POST | `/v1/{project_id}/sfs-turbo/shares` | 创建文件系统 |
| `CreateSharedTag` | POST | `/v1/{project_id}/sfs-turbo/{share_id}/tags` | 创建共享标签 |
| `DeleteActiveDirectoryDomain` | DELETE | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/active-directory-domain` | 退出AD域 |
| `DeleteBackendTarget` | DELETE | `/v1/{project_id}/sfs-turbo/shares/{share_id}/targets/{target_id}` | 删除后端存储 |
| `DeleteFsDir` | DELETE | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/dir` | 删除文件系统目录 |
| `DeleteFsDirQuota` | DELETE | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/dir-quota` | 删除目标文件夹配额 |
| `DeleteFsTask` | DELETE | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/{feature}/tasks/{task_id}` | 取消/删除文件系统异步任务 |
| `DeleteHpcCacheTask` | DELETE | `/v1/{project_id}/sfs-turbo/{share_id}/hpc-cache/task/{task_id}` | 删除数据导入导出任务 |
| `DeleteLdapConfig` | DELETE | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/ldap` | 删除LDAP配置 |
| `DeletePermRule` | DELETE | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/perm-rules/{rule_id}` | 删除权限规则 |
| `DeleteShare` | DELETE | `/v1/{project_id}/sfs-turbo/shares/{share_id}` | 删除文件系统 |
| `DeleteSharedTag` | DELETE | `/v1/{project_id}/sfs-turbo/{share_id}/tags/{key}` | 删除共享标签 |
| `ExpandShare` | POST | `/v1/{project_id}/sfs-turbo/shares/{share_id}/action` | 扩容文件系统 |
| `ListBackendTargets` | GET | `/v1/{project_id}/sfs-turbo/shares/{share_id}/targets` | 查询后端存储列表 |
| `ListFsTasks` | GET | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/{feature}/tasks` | 获取文件系统异步任务列表 |
| `ListHpcCacheTasks` | GET | `/v1/{project_id}/sfs-turbo/{share_id}/hpc-cache/tasks` | 查询数据导入导出任务列表 |
| `ListPermRules` | GET | `/v1/{project_id}/sfs-turbo/shares/{share_id}/fs/perm-rules` | 查询文件系统的权限规则列表 |
| `ListSharedTags` | GET | `/v1/{project_id}/sfs-turbo/tags` | 查询租户所有共享的标签 |

... and 25 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
