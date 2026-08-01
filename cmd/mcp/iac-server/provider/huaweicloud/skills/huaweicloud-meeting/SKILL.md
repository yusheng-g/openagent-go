---
name: huaweicloud-meeting
description: HuaweiCloud Meeting API guide. 202 APIs covering 云会议室管理, 企业应用管理, 企业管理, 企业管理员管理, 企业级会议事件推送设置.
---

# HuaweiCloud Meeting API Guide

202 APIs. Tags: 云会议室管理, 企业应用管理, 企业管理, 企业管理员管理, 企业级会议事件推送设置, 企业资源管理, 企业部门管理, 会议QoS, 会议控制, 会议管理, 会议纪要, 会议统计, 信息窗发布管理, 信息窗素材管理, 信息窗节目管理, 查询企业通讯录, 激活码管理, 用户头像管理, 用户密码管理, 用户管理, 登录鉴权, 硬终端管理, 网络研讨会管理, 网络质量

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddAppId` | POST | `/v2/usg/acs/corp/app` | 添加企业应用 |
| `AddCorp` | POST | `/v1/usg/dcs/sp/corp` | SP管理员创建企业 |
| `AddCorpAdmin` | POST | `/v1/usg/dcs/corp/admin` | 添加企业管理员 |
| `AddDepartment` | POST | `/v1/usg/dcs/corp/dept` | 添加部门 |
| `AddDevice` | POST | `/v1/usg/dcs/corp/device` | 增加终端 |
| `AddMaterial` | POST | `/v1/usg/sss/materials` | 新增信息窗素材 |
| `AddProgram` | POST | `/v1/usg/sss/programs` | 新增信息窗节目 |
| `AddPublication` | POST | `/v1/usg/sss/publications` | 新增信息窗发布 |
| `AddResource` | POST | `/v1/usg/dcs/sp/corp/{corp_id}/resource` | SP管理员分配企业资源 |
| `AddToPersonalSpace` | POST | `/v1/usg/sss/meeting-files/save-to-personal-space` | 保存会议纪要到个人云空间 |
| `AddUser` | POST | `/v1/usg/dcs/corp/member` | 添加用户 |
| `AllowAudienceJoin` | PUT | `/v1/mmc/control/conferences/allowAudience` | 主持人允许观众入会 |
| `AllowClientRecord` | PUT | `/v1/mmc/control/conferences/allowClientRecord` | 允许客户端录制 |
| `AllowGuestUnmute` | PUT | `/v1/mmc/control/conferences/mute/guestUnMute` | 与会者自己解除静音 |
| `AllowWaitingParticipant` | PUT | `/v1/mmc/control/conferences/allowWaitingParticipant` | 准入等候者 |
| `AssociateVmr` | POST | `/v1/usg/dcs/corp/vmr/assign-to-member/{account}` | 分配云会议室 |
| `BatchDeleteCorpAdmins` | POST | `/v1/usg/dcs/corp/admin/delete` | 批量删除企业管理员 |
| `BatchDeleteDevices` | POST | `/v1/usg/dcs/corp/device/delete` | 批量删除终端 |
| `BatchDeleteMaterials` | POST | `/v1/usg/sss/materials/batch-delete` | 删除信息窗素材 |
| `BatchDeletePrograms` | POST | `/v1/usg/sss/programs/batch-delete` | 删除信息窗节目 |
| `BatchDeletePublications` | POST | `/v1/usg/sss/publications/batch-delete` | 删除信息窗发布 |
| `BatchDeleteUsers` | POST | `/v1/usg/dcs/corp/member/delete` | 批量删除用户 |
| `BatchHand` | PUT | `/v1/mmc/control/conferences/participants/batch/hands` | 批量举手 |
| `BatchMoveToWaitingRoom` | PUT | `/v1/mmc/control/conferences/batchMoveToWaitingRoom` | 批量移入等候室 |
| `BatchSearchAppId` | GET | `/v2/usg/acs/corp/apps` | 分页查询企业应用 |
| `BatchShowUserDetails` | POST | `/v1/usg/abs/users/list` | 批量查询用户详情 |
| `BatchUpdateDevicesStatus` | PUT | `/v1/usg/dcs/corp/device/status/{value}` | 批量修改终端状态 |
| `BatchUpdateUserStatus` | PUT | `/v1/usg/dcs/corp/member/status/{value}` | 批量修改用户状态 |
| `BroadcastParticipant` | PUT | `/v1/mmc/control/conferences/participants/broadcast` | 广播会场 |
| `CancelBroadcast` | PUT | `/v1/mmc/control/conferences/cancelBroadcast` | 取消广播 |

... and 172 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
