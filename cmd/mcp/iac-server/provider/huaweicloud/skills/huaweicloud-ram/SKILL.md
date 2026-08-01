---
name: huaweicloud-ram
description: HuaweiCloud RAM API guide. 30 APIs covering 共享资源, 共享资源权限, 其他操作, 标签管理, 组织共享.
---

# HuaweiCloud RAM API Guide

30 APIs. Tags: 共享资源, 共享资源权限, 其他操作, 标签管理, 组织共享, 绑定的共享资源权限, 绑定的资源使用者和共享资源, 资源使用者, 资源共享实例, 资源共享邀请

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptResourceShareInvitation` | POST | `/v1/resource-share-invitations/{resource_share_invitation_id}/accept` | 接受共享邀请 |
| `AssociateResourceShare` | POST | `/v1/resource-shares/{resource_share_id}/associate` | 绑定资源使用者和共享资源 |
| `AssociateResourceSharePermission` | POST | `/v1/resource-shares/{resource_share_id}/associate-permission` | 绑定或替换共享资源权限 |
| `BatchCreateResourceShareTags` | POST | `/v1/resource-shares/{resource_share_id}/tags/create` | 资源共享实例增加标签 |
| `BatchDeleteResourceShareTags` | POST | `/v1/resource-shares/{resource_share_id}/tags/delete` | 删除资源共享实例的标签 |
| `CreateResourceShare` | POST | `/v1/resource-shares` | 创建资源共享实例 |
| `DeleteResourceShare` | DELETE | `/v1/resource-shares/{resource_share_id}` | 删除资源共享实例 |
| `DisableOrganizationShare` | POST | `/v1/organization-share/disable` | 关闭与组织共享 |
| `DisassociateResourceShare` | POST | `/v1/resource-shares/{resource_share_id}/disassociate` | 移除资源使用者或共享资源 |
| `DisassociateResourceSharePermission` | POST | `/v1/resource-shares/{resource_share_id}/disassociate-permission` | 移除共享资源权限 |
| `EnableOrganizationShare` | POST | `/v1/organization-share/enable` | 启用与组织共享 |
| `ListPermissions` | GET | `/v1/permissions` | 检索共享资源权限列表 |
| `ListPermissionVersions` | GET | `/v1/permissions/{permission_id}/versions` | 获取权限的所有版本 |
| `ListQuota` | GET | `/v1/resource-shares/quotas` | 查询资源共享的配额 |
| `ListResourceSharePermissions` | GET | `/v1/resource-shares/{resource_share_id}/associated-permissions` | 检索绑定的共享资源权限 |
| `ListResourceSharesByTags` | POST | `/v1/resource-shares/resource-instances/filter` | 根据标签信息查询实例列表 |
| `ListResourceShareTags` | GET | `/v1/resource-shares/tags` | 查询已使用的标签列表 |
| `ListResourceTypes` | GET | `/v1/resource-types` | 检索云服务资源类型 |
| `RejectResourceShareInvitation` | POST | `/v1/resource-share-invitations/{resource_share_invitation_id}/reject` | 拒绝共享邀请 |
| `SearchDistinctPrincipals` | POST | `/v1/shared-principals/search-distinct-principal` | 检索不同的资源使用者或者资源所有者 |
| `SearchDistinctSharedResources` | POST | `/v1/shared-resources/search-distinct-resource` | 检索共享的不同资源 |
| `SearchResourceShareAssociations` | POST | `/v1/resource-share-associations/search` | 检索绑定的资源使用者和共享资源 |
| `SearchResourceShareCountByTags` | POST | `/v1/resource-shares/resource-instances/count` | 根据标签信息查询实例数量 |
| `SearchResourceShareInvitation` | POST | `/v1/resource-share-invitations/search` | 检索共享邀请 |
| `SearchResourceShares` | POST | `/v1/resource-shares/search` | 检索资源共享实例 |
| `SearchSharedPrincipals` | POST | `/v1/shared-principals/search` | 检索资源使用者或者资源所有者 |
| `SearchSharedResources` | POST | `/v1/shared-resources/search` | 检索共享的资源 |
| `ShowOrganizationShare` | GET | `/v1/organization-share` | 检索是否启用与组织共享 |
| `ShowPermission` | GET | `/v1/permissions/{permission_id}` | 检索资源共享权限内容 |
| `UpdateResourceShare` | PUT | `/v1/resource-shares/{resource_share_id}` | 更新资源共享实例 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
