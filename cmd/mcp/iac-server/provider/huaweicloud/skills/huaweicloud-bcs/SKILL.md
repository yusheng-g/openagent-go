---
name: huaweicloud-bcs
description: HuaweiCloud BCS API guide. 32 APIs covering BCS监控, BCS管理, BCS联盟.
---

# HuaweiCloud BCS API Guide

32 APIs. Tags: BCS监控, BCS管理, BCS联盟

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchAddPeersToChannel` | POST | `/v2/{project_id}/blockchains/{blockchain_id}/channels/peers` | peer节点加入通道 |
| `BatchCreateChannels` | POST | `/v2/{project_id}/blockchains/{blockchain_id}/channels` | 创建通道 |
| `BatchInviteMembersToChannel` | POST | `/v2/{project_id}/members/invitations` | 邀请联盟成员 |
| `BatchRemoveOrgsFromChannel` | PUT | `/v2/{project_id}/blockchains/{blockchain_id}/{channel_id}/orgs/quit` | BCS组织退出某通道 |
| `BatchRemovePeersFromChannel` | PUT | `/v2/{project_id}/blockchains/{blockchain_id}/{channel_id}/peers/quit` | BCS某个组织中的节点退出某通道 |
| `CreateBlockchainCertByUserName` | POST | `/v2/{project_id}/blockchains/{blockchain_id}/orgs/{org_name}/usercert/{user_name}` | 生成用户证书 |
| `CreateNewBlockchain` | POST | `/v2/{project_id}/blockchains` | 创建服务实例 |
| `DeleteBlockchain` | DELETE | `/v2/{project_id}/blockchains/{blockchain_id}` | 删除服务实例 |
| `DeleteChannel` | DELETE | `/v2/{project_id}/blockchains/{blockchain_id}/channel/{channel_id}` | BCS删除某个通道 |
| `DeleteMemberInvite` | DELETE | `/v2/{project_id}/members/invitations` | 删除邀请成员信息 |
| `DownloadBlockchainCert` | GET | `/v2/{project_id}/blockchains/{blockchain_id}/cert` | 下载证书 |
| `DownloadBlockchainSdkConfig` | POST | `/v2/{project_id}/blockchains/{blockchain_id}/sdk-cfg` | 下载SDK配置 |
| `FreezeCert` | POST | `/v2/{project_id}/blockchains/{blockchain_id}/orgs/{org_name}/usercert/{user_name}/freeze` | 冻结用户证书 |
| `HandleNotification` | POST | `/v2/{project_id}/notification/handle` | 处理联盟邀请 |
| `HandleUnionMemberQuitList` | PUT | `/v2/{project_id}/members/quit` | 被邀请方退出指定联盟 |
| `ListBcsEvents` | POST | `/v2/{project_id}/blockchains/{blockchain_id}/events` | 查询服务实例告警信息 |
| `ListBcsEventsStatistic` | POST | `/v2/{project_id}/blockchains/{blockchain_id}/events/statistic` | 查询服务实例告警统计接口 |
| `ListBcsMetric` | POST | `/v2/{project_id}/blockchains/{blockchain_id}/metric/list` | 查询服务实例监控数据 |
| `ListBlockchainChannels` | GET | `/v2/{project_id}/blockchains/{blockchain_id}/channels` | 查询通道信息 |
| `ListBlockchains` | GET | `/v2/{project_id}/blockchains` | 查询服务实例列表 |
| `ListEntityMetric` | POST | `/v2/{project_id}/blockchains/{blockchain_id}/entity/metric/list` | 查询BCS组织监控数据列表 |
| `ListInstanceMetric` | POST | `/v2/{project_id}/blockchains/{blockchain_id}/entity/instance/metric/list` | 查询BCS组织实例监控数据详情 |
| `ListMembers` | GET | `/v2/{project_id}/members` | 获取联盟成员列表 |
| `ListNotifications` | GET | `/v2/{project_id}/notifications` | 获取全部通知 |
| `ListOpRecord` | GET | `/v2/{project_id}/operation/record` | 查询异步操作结果 |
| `ListQuotas` | GET | `/v2/{project_id}/quotas` | 查询配额 |
| `ShowBlockchainDetail` | GET | `/v2/{project_id}/blockchains/{blockchain_id}` | 查询实例信息 |
| `ShowBlockchainFlavors` | GET | `/v2/{project_id}/blockchains/flavors` | 查询规格 |
| `ShowBlockchainNodes` | GET | `/v2/{project_id}/blockchains/{blockchain_id}/nodes` | 查询节点信息 |
| `ShowBlockchainStatus` | GET | `/v2/{project_id}/blockchains/{blockchain_id}/status` | 查询创建状态 |

... and 2 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
