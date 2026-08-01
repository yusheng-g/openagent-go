---
name: huaweicloud-iotda
description: HuaweiCloud IoTDA API guide. 173 APIs covering AMQP队列管理, OTA升级包管理, OTA模块管理, 产品管理, 域配置管理.
---

# HuaweiCloud IoTDA API Guide

173 APIs. Tags: AMQP队列管理, OTA升级包管理, OTA模块管理, 产品管理, 域配置管理, 安全态势感知配置管理, 导出任务, 广播消息, 批量任务, 批量任务的文件管理, 接入凭证管理, 数据流转规则管理, 数据转发流控策略管理, 数据转发积压策略管理, 服务器证书管理, 标签管理, 编解码函数管理, 网桥管理, 自定义鉴权, 设备CA证书管理, 设备代理, 设备命令, 设备属性, 设备异步命令, 设备影子, 设备消息, 设备策略管理, 设备管理, 设备组管理, 设备联动规则, 设备证书, 设备鉴权模板管理, 设备隧道管理, 资源空间管理, 预调配模板管理, 鸿蒙软总线管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddApplication` | POST | `/v5/iot/{project_id}/apps` | 创建资源空间 |
| `AddBridge` | POST | `/v5/iot/{project_id}/bridges` | 创建网桥 |
| `AddCertificate` | POST | `/v5/iot/{project_id}/certificates` | 上传设备CA证书 |
| `AddDevice` | POST | `/v5/iot/{project_id}/devices` | 创建设备 |
| `AddDeviceGroup` | POST | `/v5/iot/{project_id}/device-group` | 添加设备组 |
| `AddFunctions` | POST | `/v5/iot/{project_id}/product-functions` | 创建编解码函数 |
| `AddHarmonySoftBus` | POST | `/v5/iot/{project_id}/harmony-soft-bus` | 创建鸿蒙软总线 |
| `AddQueue` | POST | `/v5/iot/{project_id}/amqp-queues` | 创建AMQP队列 |
| `AddTunnel` | POST | `/v5/iot/{project_id}/tunnels` | 创建设备隧道 |
| `BatchShowQueue` | GET | `/v5/iot/{project_id}/amqp-queues` | 查询AMQP列表 |
| `BindDevicePolicy` | POST | `/v5/iot/{project_id}/device-policies/{policy_id}/bind` | 绑定设备策略 |
| `BroadcastMessage` | POST | `/v5/iot/{project_id}/broadcast-messages` | 下发广播消息 |
| `ChangeGateway` | POST | `/v5/iot/{project_id}/devices/{device_id}/change-gateway` | 修改设备网关 |
| `ChangeRuleStatus` | PUT | `/v5/iot/{project_id}/rules/{rule_id}/status` | 修改规则状态 |
| `CheckCertificate` | POST | `/v5/iot/{project_id}/certificates/{certificate_id}/action` | 验证设备CA证书 |
| `CloseDeviceTunnel` | PUT | `/v5/iot/{project_id}/tunnels/{tunnel_id}` | 关闭设备隧道 |
| `ConfirmBatchTask` | POST | `/v5/iot/{project_id}/batchtasks/{task_id}/confirm` | 确认执行批量任务 |
| `CountAsyncHistoryCommands` | POST | `/v5/iot/{project_id}/devices/{device_id}/async-commands-history/count` | 统计设备下的历史命令总数 |
| `CreateAccessCode` | POST | `/v5/iot/{project_id}/auth/accesscode` | 生成接入凭证 |
| `CreateAsyncCommand` | POST | `/v5/iot/{project_id}/devices/{device_id}/async-commands` | 下发异步设备命令 |
| `CreateBatchTask` | POST | `/v5/iot/{project_id}/batchtasks` | 创建批量任务 |
| `CreateCommand` | POST | `/v5/iot/{project_id}/devices/{device_id}/commands` | 下发设备命令 |
| `CreateDeviceAuthenticationTemplate` | POST | `/v5/iot/{project_id}/device-authentication-templates` | 创建设备鉴权模板 |
| `CreateDeviceAuthorizer` | POST | `/v5/iot/{project_id}/device-authorizers` | 创建自定义鉴权 |
| `CreateDevicePolicy` | POST | `/v5/iot/{project_id}/device-policies` | 创建设备策略 |
| `CreateDeviceProxy` | POST | `/v5/iot/{project_id}/device-proxies` | 创建设备代理 |
| `CreateDomainConfiguration` | POST | `/v5/iot/{project_id}/domain-configurations` | 添加域配置 |
| `CreateExportTask` | POST | `/v5/iot/{project_id}/export-tasks` | 创建导出任务 |
| `CreateMessage` | POST | `/v5/iot/{project_id}/devices/{device_id}/messages` | 下发设备消息 |
| `CreateOrDeleteDeviceInGroup` | POST | `/v5/iot/{project_id}/device-group/{group_id}/action` | 管理设备组中的设备 |

... and 143 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
