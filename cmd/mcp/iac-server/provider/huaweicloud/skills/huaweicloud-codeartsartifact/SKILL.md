---
name: huaweicloud-codeartsartifact
description: HuaweiCloud CodeArtsArtifact API guide. 59 APIs covering 中心仓, 仓库关联项目, 仓库容量, 仓库管理, 仓库详情.
---

# HuaweiCloud CodeArtsArtifact API Guide

59 APIs. Tags: 中心仓, 仓库关联项目, 仓库容量, 仓库管理, 仓库详情, 关注, 制品安全, 发布库仓库容量, 发布库仓库详情, 发布库套餐查询, 发布库文件查询, 发布库文件管理, 发布库权限管理, 发布库版本管理, 发布库设置, 回收站, 审计日志, 搜索, 文件管理, 权限查看, 权限管理, 用户管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchDeleteTrashes` | DELETE | `/cloudartifact/v5/trashes` | 批量删除回收站 |
| `BatchRestoreRepo` | PUT | `/cloudartifact/v5/trashes` | 批量还原回收站 |
| `CreateArtifactory` | POST | `/cloudartifact/v5/artifact/` | 创建非maven仓库 |
| `CreateAttention` | POST | `/cloudartifact/v5/attention` | 关注组件/取消关注组件 |
| `CreateDockerRepositories` | POST | `/cloudartifact/v5/repositories` | 创建docker仓库 |
| `CreateMavenRepo` | POST | `/cloudartifact/v5/maven/repositories` | 创建maven仓库 |
| `CreateProjectRelatedRepository` | POST | `/cloudartifact/v5/maven/project/repository` | 创建项目关联仓库 |
| `DeleteArtifactFile` | DELETE | `/cloudartifact/v5/file` | 非maven删除文件 |
| `DeleteCompletelyUpdateFileState` | DELETE | `/devreposerver/v5/files/compeletion` | 彻底删除文件/文件夹 |
| `DeleteRepository` | DELETE | `/cloudartifact/v5/repositories` | 删除仓库到回收站 |
| `ListAllRepositories` | GET | `/cloudartifact/v5/{tenant_id}/{project_id}/repositories` | 查询仓库详情,不会去统计仓库下的制品数量 |
| `ListArtifactoryComponent` | GET | `/cloudartifact/v5/{tenant_id}/{project_id}/{repo_name}/file-detail` | 查询仓库文件详情 |
| `ListArtifactoryStorageStatistic` | GET | `/cloudartifact/v5/{tenant_id}/{project_id}/storageinfo/statistic` | 查询存储容量趋势 |
| `ListAttentions` | GET | `/cloudartifact/v5/attention/artifacts` | 查询关注列表 |
| `ListCapacityMessageSetting` | GET | `/devreposerver/v5/capacity-notice/settings` | 查询租户容量消息通知设置信息 |
| `ListChildProxyRepositoriesList` | GET | `/cloudartifact/v5/repositories/proxy` | 获取聚合仓下的仓库代理列表 |
| `ListDomainIpConfig` | GET | `/cloudartifact/v5/domain/ipconfig` | 查询租户级IP白名单 |
| `ListFileBuildArchives` | GET | `/devreposerver/v5/files/archives` | 分页查询构建归档包列表 |
| `ListFiles` | POST | `/devreposerver/v5/files/list` | 查询文件/项目列表 |
| `ListLatestVersionFiles` | GET | `/devreposerver/v5/{project_id}/files/version` | 查询项目下所有文件的最新版本 |
| `ListMavenList` | GET | `/cloudartifact/v5/maven/list` | 查询Maven仓库列表 |
| `ListMavenUser` | GET | `/cloudartifact/v5/repositories/users` | 查询私有库用户列表 |
| `ListNetProxy` | GET | `/cloudartifact/v5/tree/net/proxy` | 查询网络代理列表 |
| `ListProjectRolePermissions` | GET | `/devreposerver/v5/project-role/permissions` | 查看项目的角色权限设置 |
| `ListProjectUsers` | GET | `/cloudartifact/v5/projects/{project_id}/users` | 查询项目下的用户 |
| `ListSecGuardList` | GET | `/cloudartifact/v5/sec-guard/task/list` | 查询制品安全扫描任务列表 |
| `ListUserPrivileges` | GET | `/v5/user/{project_id}/privileges` | 查询用户权限 |
| `ModifyRepository` | PUT | `/cloudartifact/v5/repositories/tab/{tab_id}` | 编辑仓库 |
| `ResetUserPassword` | POST | `/cloudartifact/v5/maven/users/me` | 重置用户密码 |
| `SearchArtifacts` | POST | `/cloudartifact/v5/tree/repos/artifacts` | 统筹搜索 |

... and 29 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
