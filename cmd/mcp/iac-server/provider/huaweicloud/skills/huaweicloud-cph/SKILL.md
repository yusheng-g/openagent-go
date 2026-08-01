---
name: huaweicloud-cph
description: HuaweiCloud CPH API guide. 51 APIs covering ADB命令, 云手机服务器管理, 任务管理, 密钥管理, 带宽管理.
---

# HuaweiCloud CPH API Guide

51 APIs. Tags: ADB命令, 云手机服务器管理, 任务管理, 密钥管理, 带宽管理, 手机实例管理, 标签管理, 编码服务管理, 自定义镜像管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddImageMember` | POST | `/v1/{project_id}/cloud-phone/images/{image_id}/members` | 共享镜像给指定账号 |
| `BatchCreateTags` | POST | `/v1/{project_id}/{resource_type}/{resource_id}/tags/action` | 批量添加标签 |
| `BatchDeleteTags` | POST | `/v1/{project_id}/{resource_type}/{resource_id}/tags/action` | 批量删除标签 |
| `BatchExportCloudPhoneData` | POST | `/v1/{project_id}/cloud-phone/phones/batch-storage` | 导出云手机数据 |
| `BatchImportCloudPhoneData` | POST | `/v1/{project_id}/cloud-phone/phones/batch-restore` | 恢复云手机数据 |
| `BatchShowPhoneConnectInfos` | POST | `/v1/{project_id}/cloud-phone/phones/batch-connection` | 获取云手机连接信息 |
| `ChangeCloudPhoneServer` | POST | `/v2/{project_id}/cloud-phone/servers/{server_id}/change` | 切换云手机服务器 |
| `ChangeCloudPhoneServerModel` | POST | `/v1/{project_id}/cloud-phone/servers/change-server-model` | 变更云手机服务器规格 |
| `CreateCloudPhoneSingleServer` | POST | `/v2.1/{project_id}/cloud-phone/servers` | 创建云手机裸服务器 |
| `CreateNet2CloudPhoneServer` | POST | `/v2/{project_id}/cloud-phone/servers` | 购买云手机服务器 |
| `DeleteImage` | DELETE | `/v1/{project_id}/cloud-phone/images/{image_id}` | 删除镜像 |
| `DeleteImageMember` | DELETE | `/v1/{project_id}/cloud-phone/images/{image_id}/members/{member_id}` | 删除共享镜像 |
| `DeleteShareApps` | DELETE | `/v1/{project_id}/cloud-phone/phones/share-apps` | 删除共享应用 |
| `DeleteShareFiles` | POST | `/v1/{project_id}/cloud-phone/phones/share-files` | 删除共享存储文件 |
| `ExpandPhoneDataVolumeSize` | POST | `/v1/{project_id}/cloud-phone/phones/expand-volume` | 扩容云手机数据盘大小 |
| `ImportTraffic` | POST | `/v1/{project_id}/cloud-phone/phones-traffic` | 云手机流量导流 |
| `InstallApk` | POST | `/v1/{project_id}/cloud-phone/phones/commands` | 安装apk |
| `ListCloudPhoneImages` | GET | `/v1/{project_id}/cloud-phone/phone-images` | 查询手机镜像 |
| `ListCloudPhoneModels` | GET | `/v1/{project_id}/cloud-phone/phone-models` | 查询云手机规格列表 |
| `ListCloudPhones` | GET | `/v1/{project_id}/cloud-phone/phones` | 查询云手机列表 |
| `ListCloudPhoneServerModels` | GET | `/v1/{project_id}/cloud-phone/server-models` | 查询云手机服务器规格列表 |
| `ListCloudPhoneServers` | GET | `/v1/{project_id}/cloud-phone/servers` | 查询云手机服务器列表 |
| `ListEncodeServers` | GET | `/v1/{project_id}/cloud-phone/encode-servers` | 查询编码服务 |
| `ListImageMembers` | GET | `/v1/{project_id}/cloud-phone/images/{image_id}/members` | 获取镜像已共享账号列表 |
| `ListImages` | GET | `/v1/{project_id}/cloud-phone/images` | 查询自定义镜像列表 |
| `ListJobs` | GET | `/v1/{project_id}/cloud-phone/jobs` | 查询任务执行状态列表 |
| `ListProjectTags` | GET | `/v1/{project_id}/{resource_type}/tags` | 查询项目标签 |
| `ListResourceInstances` | POST | `/v1/{project_id}/{resource_type}/resource_instances/action` | 查询资源实例 |
| `ListResourceTags` | GET | `/v1/{project_id}/{resource_type}/{resource_id}/tags` | 查询资源标签 |
| `ListShareFiles` | GET | `/v1/{project_id}/cloud-phone/servers/share-files` | 查询共享存储文件 |

... and 21 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
