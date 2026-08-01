---
name: huaweicloud-hcsecs
description: HuaweiCloud HCSECS API guide. 145 APIs covering ECS健康检查, HA管理, SSH密钥管理, hypervisors查询, 主机组管理.
---

# HuaweiCloud HCSECS API Guide

145 APIs. Tags: ECS健康检查, HA管理, SSH密钥管理, hypervisors查询, 主机组管理, 云服务器监控管理, 云服务器组管理, 元数据管理, 克隆弹性云服务器, 变更规格, 可用分区, 安全组管理, 密码, 快照管理, 批量操作云服务器, 操作云服务器, 整机快照, 查询API版本信息, 查询Job状态, 查询操作记录, 标签管理, 浮动IP管理, 生命周期管理, 磁盘管理, 租户配额管理, 网卡管理, 网络管理, 获取vnc地址, 规格管理, 重装切换操作系统, 镜像管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AttachExtServerVolume` | POST | `/v1/{tenant_id}/cloudservers/{server_id}/attachvolume` | 弹性云服务器挂载磁盘 |
| `AttachExtSharedVolume` | POST | `/v1/{tenant_id}/batchaction/attachvolumes/{volume_id}` | 批量挂载指定共享盘 |
| `AttachServerInterface` | POST | `/v2.1/{tenant_id}/servers/{server_id}/os-interface` | 添加云服务器网卡_社区兼容 |
| `AttachServerVolume` | POST | `/v2.1/{tenant_id}/servers/{server_id}/os-volume_attachments` | 弹性云服务器挂载磁盘_社区兼容 |
| `BatchAddExtServerNics` | POST | `/v1/{tenant_id}/cloudservers/{server_id}/nics` | 批量添加云服务器网卡 |
| `BatchDeleteExtServerNics` | POST | `/v1/{tenant_id}/cloudservers/{server_id}/nics/delete` | 批量删除云服务器网卡 |
| `BatchExecuteExtServerTags` | POST | `/v1/{tenant_id}/cloudservers/{server_id}/tags/action` | 批量管理虚拟机标签 |
| `BatchResetExtServersPwd` | PUT | `/v1/{tenant_id}/cloudservers/os-reset-passwords` | 批量重置弹性云服务器密码 |
| `BatchResizeExtLiveServer` | POST | `/v1/{tenant_id}/cloudservers/live-resize` | 批量在线变更规格 |
| `BatchResizeExtServer` | POST | `/v1/{tenant_id}/cloudservers/resize` | 批量变更规格 |
| `BatchUpdateExtServersName` | PUT | `/v1/{tenant_id}/cloudservers/server-name` | 批量修改弹性云服务器 |
| `CloneExtServer` | POST | `/v1/{tenant_id}/cloudservers/{server_id}/action` | 克隆弹性云服务器 |
| `CreateAggregate` | POST | `/v2.1/{tenant_id}/os-aggregates` | 创建主机组_社区兼容 |
| `CreateExtServerGroup` | POST | `/v1/{tenant_id}/cloudservers/os-server-groups` | 创建云服务器组 |
| `CreateExtServers` | POST | `/v1.1/{tenant_id}/cloudservers` | 创建云服务器 |
| `CreateFloatingIp` | POST | `/v2.1/{tenant_id}/os-floating-ips` | 创建浮动IP_社区兼容(废弃,不推荐使用) |
| `CreateKeypair` | POST | `/v2.1/{tenant_id}/os-keypairs` | 创建和导入SSH密钥_社区兼容 |
| `CreateSecurityGroup` | POST | `/v2.1/{tenant_id}/os-security-groups` | 创建安全组_社区兼容(废弃,不推荐使用) |
| `CreateSecurityRule` | POST | `/v2.1/{tenant_id}/os-security-group-rules` | 创建安全组规则_社区兼容(废弃,不推荐使用) |
| `CreateServers` | POST | `/v2.1/{tenant_id}/servers` | 创建云服务器_社区兼容 |
| `CreateVolume` | POST | `/v2.1/{tenant_id}/os-volumes` | 创建磁盘_社区兼容(废弃,不推荐使用) |
| `DeleteAggregate` | DELETE | `/v2.1/{tenant_id}/os-aggregates/{aggregate_id}` | 删除主机组_社区兼容 |
| `DeleteExtServerGroup` | DELETE | `/v1/{tenant_id}/cloudservers/os-server-groups/{server_group_id}` | 删除云服务器组 |
| `DeleteExtServerMetaItem` | DELETE | `/v1/{tenant_id}/cloudservers/{server_id}/metadata/{key}` | 删除云服务器指定元数据 |
| `DeleteExtServerPassword` | DELETE | `/v1/{tenant_id}/cloudservers/{server_id}/os-server-password` | Windows云服务器清除密码 |
| `DeleteExtServers` | POST | `/v1/{tenant_id}/cloudservers/delete` | 删除云服务器 |
| `DeleteFloatingIp` | DELETE | `/v2.1/{tenant_id}/os-floating-ips/{floating_ip_id}` | 删除浮动IP_社区兼容 |
| `DeleteInstanceSnapshot` | DELETE | `/v2/cloudimages` | 删除云服务器整机快照 |
| `DeleteKeypair` | DELETE | `/v2.1/{tenant_id}/os-keypairs/{keypair_name}` | 删除SSH密钥_社区兼容 |
| `DeleteSecurityRule` | DELETE | `/v2.1/{tenant_id}/os-security-group-rules/{id}` | 删除安全组规则_OpenStack原生 |

... and 115 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
