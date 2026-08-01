---
name: huaweicloud-workspace
description: HuaweiCloud Workspace API guide. 352 APIs covering ai功能, 云办公主机, 云办公服务, 产品套餐, 任务.
---

# HuaweiCloud Workspace API Guide

352 APIs. Tags: ai功能, 云办公主机, 云办公服务, 产品套餐, 任务, 协同桌面, 可用分区, 告警, 委托, 定时任务, 导出中心, 应用中心, 应用管控, 录屏审计, 快照, 报表统计, 桌面, 桌面名称策略, 桌面标签, 桌面池, 用户, 用户操作日志, 用户组, 磁盘, 租户配置, 策略组, 组织单元, 终端与桌面绑定, 网络, 脚本, 订单, 证书管理, 连接信息, 连接记录, 配额, 镜像

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddDesktopPoolVolumes` | POST | `/v2/{project_id}/desktop-pools/{pool_id}/volumes/batch-add` | 桌面池批量添加磁盘 |
| `AddDesktopSubResources` | POST | `/v2/{project_id}/desktop/sub-resources` | 桌面购买附属资源 |
| `AddDesktopVolumes` | POST | `/v2/{project_id}/desktops/{desktop_id}/volumes` | 增加桌面磁盘 |
| `AddMetricNotifyRule` | POST | `/v2/{project_id}/statistics/notify-rules` | 新增通知规则 |
| `AddOu` | POST | `/v2/{project_id}/ous` | 新增OU信息 |
| `AddRestrictedRule` | POST | `/v1/{project_id}/app-center/app-restricted-rules` | 增加管控规则 |
| `AddSite` | POST | `/v2/{project_id}/sites` | 新增站点 |
| `AddVolumes` | POST | `/v2/{project_id}/volumes` | 批量增加桌面磁盘 |
| `ApplyDesktopsInternet` | POST | `/v2/{project_id}/eips` | 开通桌面上网功能 |
| `ApplyInternet` | POST | `/v2/{project_id}/internet` | 开通上网功能 |
| `ApplySubnetBandwidth` | POST | `/v2/{project_id}/bandwidths` | 创建云办公带宽 |
| `ApplyWorkspace` | POST | `/v2/{project_id}/workspaces` | 开通云办公服务 |
| `AssociateDesktopsEip` | POST | `/v2/{project_id}/eips/binding` | 桌面绑定EIP |
| `AttachInstances` | POST | `/v2/{project_id}/desktops/attach` | 分配用户 |
| `BatchAddDesktopsTags` | POST | `/v2/{project_id}/desktops/batch-tags` | 批量添加多个桌面标签 |
| `BatchAssociateInstances` | POST | `/v2/{project_id}/desktops/pre-batch-attach` | 预批量分配用户 |
| `BatchAttachInstances` | POST | `/v2/{project_id}/desktops/batch-attach` | 批量分配用户 |
| `BatchChangeDesktopNetwork` | POST | `/v2/{project_id}/desktops/networks/batch-change` | 批量切换桌面网络 |
| `BatchChangeTags` | POST | `/v2/{project_id}/desktops/{desktop_id}/tags/action` | 批量添加或删除标签 |
| `BatchCreateDesktopSnapshot` | POST | `/v2/{project_id}/snapshots/batch-create` | 批量创建快照 |
| `BatchCreateUsers` | POST | `/v2/{project_id}/users/batch-create` | 批量创建用户 |
| `BatchDeleteAccessPolicies` | DELETE | `/v2/{project_id}/access-policy` | 删除接入策略 |
| `BatchDeleteAppRules` | POST | `/v1/{project_id}/app-center/app-rules/batch-delete` | 批量删除规则 |
| `BatchDeleteApps` | POST | `/v1/{project_id}/app-center/apps/actions/batch-delete` | 批量删除应用 |
| `BatchDeleteDesktopNamePolicy` | POST | `/v2/{project_id}/desktop-name-policies/batch-delete` | 批量删除桌面名称策略 |
| `BatchDeleteDesktops` | POST | `/v2/{project_id}/desktops/batch-delete` | 批量删除桌面 |
| `BatchDeleteDesktopSnapshot` | POST | `/v2/{project_id}/snapshots/batch-delete` | 批量删除快照 |
| `BatchDeleteDesktopsTags` | DELETE | `/v2/{project_id}/desktops/batch-tags` | 批量删除多个桌面标签 |
| `BatchDeleteJobs` | POST | `/v1/{project_id}/app-center/jobs/actions/batch-delete` | 批量删除job |
| `BatchDeleteOtpDevices` | DELETE | `/v2/{project_id}/users/{user_id}/otp-devices` | 解绑OTP设备 |

... and 322 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
