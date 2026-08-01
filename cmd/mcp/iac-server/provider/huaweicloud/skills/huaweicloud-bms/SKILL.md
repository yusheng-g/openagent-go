---
name: huaweicloud-bms
description: HuaweiCloud BMS API guide. 35 APIs covering Job管理, 查询API版本信息, 裸金属服务器二维标签管理, 裸金属服务器云硬盘管理, 裸金属服务器元数据管理.
---

# HuaweiCloud BMS API Guide

35 APIs. Tags: Job管理, 查询API版本信息, 裸金属服务器二维标签管理, 裸金属服务器云硬盘管理, 裸金属服务器元数据管理, 裸金属服务器元数据配置管理, 裸金属服务器密码管理, 裸金属服务器状态管理, 裸金属服务器生命周期管理, 裸金属服务器租户配额管理, 裸金属服务器网卡管理, 裸金属服务器规格管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddServerNics` | POST | `/v1/{project_id}/baremetalservers/{server_id}/nics` | 裸金属服务器绑定弹性网卡 |
| `AttachBaremetalServerVolume` | POST | `/v1/{project_id}/baremetalservers/{server_id}/attachvolume` | 裸金属服务器挂载云硬盘 |
| `BatchCreateBaremetalServerTags` | POST | `/v1/{project_id}/baremetalservers/{server_id}/tags/action` | 批量添加裸金属服务器标签 |
| `BatchDeleteBaremetalServerTags` | POST | `/v1/{project_id}/baremetalservers/{server_id}/tags/action` | 批量删除裸金属服务器标签 |
| `BatchRebootBaremetalServers` | POST | `/v1/{project_id}/baremetalservers/action` | 重启裸金属服务器 |
| `BatchStartBaremetalServers` | POST | `/v1/{project_id}/baremetalservers/action` | 启动裸金属服务器 |
| `BatchStopBaremetalServers` | POST | `/v1/{project_id}/baremetalservers/action` | 关闭裸金属服务器 |
| `ChangeBaremetalServerName` | PUT | `/v1/{project_id}/baremetalservers/{server_id}` | 修改裸金属服务器名称 |
| `ChangeBaremetalServerOs` | POST | `/v1/{project_id}/baremetalservers/{server_id}/changeos` | 切换裸金属服务器的操作系统 |
| `CreateBareMetalServers` | POST | `/v1/{project_id}/baremetalservers` | 创建裸金属服务器 |
| `DeleteServerNics` | POST | `/v1/{project_id}/baremetalservers/{server_id}/nics/delete` | 裸金属服务器解绑弹性网卡 |
| `DeleteWindowsBareMetalServerPassword` | DELETE | `/v1/{project_id}/baremetalservers/{server_id}/os-server-password` | Windows裸金属服务器清除密码 |
| `DetachBaremetalServerVolume` | DELETE | `/v1/{project_id}/baremetalservers/{server_id}/detachvolume/{attachment_id}` | 裸金属服务器卸载云磁盘 |
| `ListBaremetalFlavorDetailExtends` | GET | `/v1/{project_id}/baremetalservers/flavors` | 查询规格详情和规格扩展信息列表 |
| `ListBareMetalServerDetails` | GET | `/v1/{project_id}/baremetalservers/{server_id}` | 查询裸金属服务器详情 |
| `ListBareMetalServers` | GET | `/v1/{project_id}/baremetalservers/detail` | 查询裸金属服务器详情列表 |
| `ListBareMetalServersDetail` | GET | `/v1.1/{project_id}/baremetalservers/detail` | 查询裸金属服务器列表 |
| `ModifyVmNic` | PUT | `/v1/{project_id}/baremetalservers/nics/{nic_id}` | 编辑port |
| `ReinstallBaremetalServerOs` | POST | `/v1/{project_id}/baremetalservers/{server_id}/reinstallos` | 重装裸金属服务器操作系统 |
| `ResetPwdOneClick` | PUT | `/v1/{project_id}/baremetalservers/{server_id}/os-reset-password` | 一键重置裸金属服务器密码 |
| `ShowAvailableResource` | GET | `/v1/{project_id}/baremetalservers/available_resource` | 查询可用资源 |
| `ShowBaremetalServerInterfaceAttachments` | GET | `/v1/{project_id}/baremetalservers/{server_id}/os-interface` | 查询裸金属服务器网卡信息 |
| `ShowBaremetalServerTags` | GET | `/v1/{project_id}/baremetalservers/{server_id}/tags` | 查询裸金属服务器标签 |
| `ShowBaremetalServerVolumeInfo` | GET | `/v1/{project_id}/baremetalservers/{server_id}/os-volume_attachments` | 查询裸金属服务器挂载的云硬盘信息 |
| `ShowJobInfos` | GET | `/v1/{project_id}/jobs/{job_id}` | 查询Job状态 |
| `ShowMetadataOptions` | GET | `/v1/{project_id}/baremetalservers/{server_id}/metadata-options` | 查询裸金属服务器元数据配置 |
| `ShowResetPwd` | GET | `/v1/{project_id}/baremetalservers/{server_id}/os-resetpwd-flag` | 查询是否支持一键重置密码 |
| `ShowServerRemoteConsole` | POST | `/v1/{project_id}/baremetalservers/{server_id}/remote_console` | 获取裸金属服务器远程登录地址 |
| `ShowSpecifiedVersion` | GET | `/{api_version}` | 查询指定API版本信息 |
| `ShowTenantQuota` | GET | `/v1/{project_id}/baremetalservers/limits` | 查询租户配额 |

... and 5 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
