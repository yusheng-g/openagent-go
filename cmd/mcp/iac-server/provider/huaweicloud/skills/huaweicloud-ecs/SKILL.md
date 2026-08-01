---
name: huaweicloud-ecs
description: HuaweiCloud ECS API guide. 131 APIs covering 云服务器操作管理, 云服务器组管理, 元数据管理, 元数据配置管理, 可用区管理. Detailed swagger definitions in references/<APIName>.json.
---

# HuaweiCloud ECS API Guide

131 APIs. Tags: 云服务器操作管理, 云服务器组管理, 元数据管理, 元数据配置管理, 可用区管理, 回收站, 安全组管理, 密钥密码管理, 批量操作, 查询API版本信息, 查询Job状态, 标签管理, 模板, 状态管理, 生命周期管理, 磁盘管理, 租户配额管理, 网卡管理, 规格管理, 计划事件

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptScheduledEvent` | POST | `/v3/{project_id}/instance-scheduled-events/{id}/actions/accept` | 接受并授权执行计划事件操作 |
| `AddServerGroupMember` | POST | `/v1/{project_id}/cloudservers/os-server-groups/{server_group_id}/action` | 添加云服务器组成员 |
| `AssociateServerVirtualIp` | PUT | `/v1/{project_id}/cloudservers/nics/{nic_id}` | 云服务器网卡配置虚拟IP地址 |
| `AttachServerVolume` | POST | `/v1/{project_id}/cloudservers/{server_id}/attachvolume` | 弹性云服务器挂载磁盘 |
| `BatchAddServerGroupMember` | POST | `/v1/{project_id}/cloudservers/os-server-groups/{server_group_id}/add_members` | 云服务器组批量添加成员 |
| `BatchAddServerNics` | POST | `/v1/{project_id}/cloudservers/{server_id}/nics` | 批量添加云服务器网卡 |
| `BatchAttachSharableVolumes` | POST | `/v1/{project_id}/batchaction/attachvolumes/{volume_id}` | 批量挂载指定共享盘 |
| `BatchCreateServerTags` | POST | `/v1/{project_id}/cloudservers/{server_id}/tags/action` | 批量添加云服务器标签 |
| `BatchDeleteServerGroupMember` | POST | `/v1/{project_id}/cloudservers/os-server-groups/{server_group_id}/remove_members` | 云服务器组批量删除成员 |
| `BatchDeleteServerNics` | POST | `/v1/{project_id}/cloudservers/{server_id}/nics/delete` | 批量删除云服务器网卡 |
| `BatchDeleteServerTags` | POST | `/v1/{project_id}/cloudservers/{server_id}/tags/action` | 批量删除云服务器标签 |
| `BatchDetachVolumes` | POST | `/v1/{project_id}/batchaction/detachvolumes/{volume_id}` | 批量卸载卷 |
| `BatchRebootServers` | POST | `/v1/{project_id}/cloudservers/action` | 批量重启云服务器 |
| `BatchResetServersPassword` | PUT | `/v1/{project_id}/cloudservers/os-reset-passwords` | 批量重置弹性云服务器密码 |
| `BatchResizeServers` | POST | `/v1/{project_id}/cloudservers/batch-resize` | 批量变更云服务器规格 |
| `BatchStartServers` | POST | `/v1/{project_id}/cloudservers/action` | 批量启动云服务器 |
| `BatchStopServers` | POST | `/v1/{project_id}/cloudservers/action` | 批量关闭云服务器 |
| `BatchUpdateServersName` | PUT | `/v1/{project_id}/cloudservers/server-name` | 批量修改弹性云服务器 |
| `ChangeServerChargeMode` | POST | `/v1/{project_id}/cloudservers/actions/change-charge-mode` | 更换云服务器计费模式 |
| `ChangeServerNetworkInterface` | POST | `/v1/{project_id}/cloudservers/{server_id}/os-interface/{port_id}/change-network-interface` | 更新云服务器指定网卡属性 |
| `ChangeServerOsWithCloudInit` | POST | `/v2/{project_id}/cloudservers/{server_id}/changeos` | 切换弹性云服务器操作系统(安装Cloud init) |
| `ChangeServerOsWithoutCloudInit` | POST | `/v1/{project_id}/cloudservers/{server_id}/changeos` | 切换弹性云服务器操作系统(未安装Cloud init) |
| `ChangeVpc` | POST | `/v1/{project_id}/cloudservers/{server_id}/changevpc` | 云服务器切换虚拟私有云 |
| `CreateLaunchTemplate` | POST | `/v3/{project_id}/launch-templates` | 创建模板 |
| `CreatePostPaidServers` | POST | `/v1/{project_id}/cloudservers` | 创建云服务器(按需) |
| `CreateServerGroup` | POST | `/v1/{project_id}/cloudservers/os-server-groups` | 创建云服务器组 |
| `CreateServers` | POST | `/v1.1/{project_id}/cloudservers` | 创建云服务器 |
| `DeleteLaunchTemplates` | DELETE | `/v3/{project_id}/launch-templates/{launch_template_id}` | 删除模板 |
| `DeleteRecycleBinServer` | DELETE | `/v1/{project_id}/recycle-bin/cloudservers/{server_id}` | 删除回收站中虚拟机 |
| `DeleteServerGroup` | DELETE | `/v1/{project_id}/cloudservers/os-server-groups/{server_group_id}` | 删除云服务器组 |

... and 101 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
