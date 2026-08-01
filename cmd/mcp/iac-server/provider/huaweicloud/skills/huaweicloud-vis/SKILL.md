---
name: huaweicloud-vis
description: HuaweiCloud VIS API guide. 33 APIs covering obs桶策略管理, 凭证管理, 服务开通管理, 视频流管理, 设备指标统计.
---

# HuaweiCloud VIS API Guide

33 APIs. Tags: obs桶策略管理, 凭证管理, 服务开通管理, 视频流管理, 设备指标统计, 设备管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateAkskCertificate` | POST | `/v1/{project_id}/credentials/aksks` | 创建AK/SK凭证 |
| `CreateDeviceCertificate` | POST | `/v1/{project_id}/credentials/codes` | 创建GB/T28181凭证 |
| `CreateDeviceChannel` | POST | `/v1/{project_id}/devices` | 创建GB/T28181设备通道 |
| `CreateStream` | POST | `/v1/{project_id}/streams` | 创建视频流 |
| `CreateSubscription` | POST | `/v1/{project_id}/subscription` | 开通服务 |
| `DeleteAkskCertificate` | DELETE | `/v1/{project_id}/credentials/aksks/{access_key}` | 删除AK/SK凭证 |
| `DeleteDevice` | DELETE | `/v1/{project_id}/devices/{device_id}` | 删除GB/T28181设备 |
| `DeleteDeviceCertificate` | DELETE | `/v1/{project_id}/credentials/codes/{username}` | 删除GB/T28181凭证 |
| `DeleteStream` | DELETE | `/v1/{project_id}/streams/{stream_name}` | 删除视频流 |
| `ListAkskCertificates` | GET | `/v1/{project_id}/credentials/aksks` | 获取AK/SK凭证列表 |
| `ListBucket` | GET | `/v1/{project_id}/buckets` | 获取桶信息列表 |
| `ListDevices` | GET | `/v1/{project_id}/devices` | 获取设备列表 |
| `ListDevicesCertificates` | GET | `/v1/{project_id}/credentials/codes` | 获取GB/T28181凭证列表 |
| `ListLongTermOfflineDevices` | GET | `/v1/{project_id}/operation/longTermOfflineDeviceList` | 获取长期不在线设备列表 |
| `ListNotSendDataDevices` | GET | `/v1/{project_id}/operation/notSendDataDeviceList` | 获取在线未推流设备列表 |
| `ListNvrChannels` | GET | `/v1/{project_id}/devices/{device_id}` | 获取NVR设备通道列表 |
| `ListPastOnlineDevices` | GET | `/v1/{project_id}/operation/pastOnlineDeviceList` | 获取曾经上线的设备列表 |
| `ListShortTermOfflineDevices` | GET | `/v1/{project_id}/operation/shortTermOfflineDeviceList` | 获取近期掉线的设备列表 |
| `ListStreamInfos` | GET | `/v1/{project_id}/streams/{stream_name}` | 获取视频流信息 |
| `ListStreams` | GET | `/v1/{project_id}/streams` | 获取视频流列表 |
| `ListStreamsAddresses` | GET | `/v1/{project_id}/streams/{stream_name}/endpoint` | 获取视频流地址 |
| `ListSubscriptions` | GET | `/v1/{project_id}/subscription` | 获取服务开通信息 |
| `ListTodayDroppedDevices` | GET | `/v1/{project_id}/operation/todayDroppedDeviceList` | 获取新掉线设备列表 |
| `ListTodayNewOnlineDevices` | GET | `/v1/{project_id}/operation/todayNewOnlineDeviceList` | 获取新上线设备列表 |
| `ListTodayPacketReceivedRateDevices` | GET | `/v1/{project_id}/operation/todayPacketReceivedRateDeviceList` | 获取视频包接收率 |
| `UpdateAksk` | PUT | `/v1/{project_id}/credentials/aksks/{access_key}` | 更新AK/SK凭证 |
| `UpdateAuthority` | PUT | `/v1/{project_id}/buckets/authority` | 更新桶授权 |
| `UpdateDeviceCertificate` | PUT | `/v1/{project_id}/credentials/codes/{username}` | 更新GB/T28181凭证 |
| `UpdateDeviceChannel` | PUT | `/v1/{project_id}/devices/{device_id}/channels/{channel_id}` | 更新GB/T28181设备通道信息 |
| `UpdateDeviceChannelStrategy` | PUT | `/v1/{project_id}/devices/{device_id}/channels/{channel_id}/access-strategy` | 更新GB/T28181设备通道接入策略 |

... and 3 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
