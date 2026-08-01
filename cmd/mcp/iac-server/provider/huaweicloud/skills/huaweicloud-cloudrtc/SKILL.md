---
name: huaweicloud-cloudrtc
description: HuaweiCloud CloudRTC API guide. 42 APIs covering OBS桶管理, 单流任务管理, 合流任务管理, 应用回调管理, 应用管理.
---

# HuaweiCloud CloudRTC API Guide

42 APIs. Tags: OBS桶管理, 单流任务管理, 合流任务管理, 应用回调管理, 应用管理, 录制规则管理, 房间管理, 数据统计分析, 自动录制配置

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateApp` | POST | `/apps` | 创建应用 |
| `CreateIndividualStreamJob` | POST | `/apps/{app_id}/individual-stream-jobs` | 启动单流任务 |
| `CreateMixJob` | POST | `/apps/{app_id}/mix-stream-jobs` | 启动合流任务 |
| `CreateRecordRule` | POST | `/apps/{app_id}/record-rules` | 创建或更新录制规则 |
| `DeleteApp` | DELETE | `/apps/{app_id}` | 删除应用 |
| `DeleteRecordRule` | DELETE | `/apps/{app_id}/record-rules/{rule_id}` | 删除录制规则 |
| `ListApps` | GET | `/apps` | 查询应用列表 |
| `ListObsBucketObjects` | GET | `/rtc-ops/buckets/objects` | 查询OBS桶下对象列表 |
| `ListObsBuckets` | GET | `/rtc-ops/buckets` | 查询OBS桶列表 |
| `ListRecordRules` | GET | `/apps/{app_id}/record-rules` | 查询录制规则列表 |
| `ListRtcAbnormalEvent` | GET | `/v1/{project_id}/rtc/client/abnormalevent` | 查询用户异常体验事件接口 |
| `ListRtcAbnormalEventDimension` | GET | `/v1/rtc/data/abnormal-event/dimension` | 查询异常事件用户分布 |
| `ListRtcAbnormalEvents` | GET | `/v1/rtc/data/abnormal-events` | 查询用户异常体验事件 |
| `ListRtcClientQosDetails` | GET | `/v1/{project_id}/rtc/client/qos/details` | 查询用户通话指标 |
| `ListRtcEvent` | GET | `/v1/{project_id}/rtc/client/event` | 查询详情事件接口 |
| `ListRtcHistoryQuality` | GET | `/v1/{project_id}/rtc/history/quality` | 查询历史质量 |
| `ListRtcHistoryScale` | GET | `/v1/{project_id}/rtc/history/scale` | 查询历史规模 |
| `ListRtcHistoryUsage` | GET | `/v1/{project_id}/rtc/history/usage` | 查询用量 |
| `ListRtcRealtimeNetwork` | GET | `/v1/{project_id}/rtc/realtime/network` | 查询实时网络 |
| `ListRtcRealtimeQuality` | GET | `/v1/{project_id}/rtc/realtime/quality` | 查询实时质量数据 |
| `ListRtcRealtimeScale` | GET | `/v1/{project_id}/rtc/realtime/scale` | 查询实时规模 |
| `ListRtcRealtimeScaleDimension` | GET | `/v1/{project_id}/rtc/realtime/scale/dimension` | 查询实时规模分布 |
| `ListRtcRoomList` | GET | `/v1/{project_id}/rtc/rooms` | 查询房间列表 |
| `ListRtcUserList` | GET | `/v1/{project_id}/rtc/users` | 查询用户列表 |
| `RemoveRoom` | POST | `/apps/{app_id}/rooms/{room_id}/dismiss` | 解散房间 |
| `RemoveUsers` | POST | `/apps/{app_id}/rooms/{room_id}/batch-remove-users` | 踢除在线用户 |
| `ShowApp` | GET | `/apps/{app_id}` | 查询单个应用 |
| `ShowAutoRecord` | GET | `/apps/{app_id}/auto-record-mode` | 查询自动录制配置 |
| `ShowIndividualStreamJob` | GET | `/apps/{app_id}/individual-stream-jobs/{job_id}` | 查询单流任务状态 |
| `ShowMixJob` | GET | `/apps/{app_id}/mix-stream-jobs/{job_id}` | 查询合流任务 |

... and 12 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
