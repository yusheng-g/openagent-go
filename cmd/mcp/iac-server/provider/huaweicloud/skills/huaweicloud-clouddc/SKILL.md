---
name: huaweicloud-clouddc
description: HuaweiCloud CloudDC API guide. 32 APIs covering 机房管理, 机架管理, 标签管理, 物理服务器管理, 物理服务器诊断.
---

# HuaweiCloud CloudDC API Guide

32 APIs. Tags: 机房管理, 机架管理, 标签管理, 物理服务器管理, 物理服务器诊断, 裸机实例管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateIrackTags` | POST | `/v1/{project_id}/iracks/{id}/tags/create` | 批量创建机柜标签 |
| `BatchCreateTags` | POST | `/v1/{project_id}/{resource_type}/{resource_id}/tags/create` | 批量创建资源标签 |
| `BatchDeleteIrackTags` | POST | `/v1/{project_id}/iracks/{id}/tags/delete` | 批量删除机柜标签 |
| `BatchDeleteTags` | POST | `/v1/{project_id}/{resource_type}/{resource_id}/tags/delete` | 批量删除资源标签 |
| `ChangeInstancePassword` | PUT | `/v1/{project_id}/instances/password` | 批量修改实例密码 |
| `ChangeServerPowerState` | PUT | `/v1/{project_id}/physicalservers/power-state` | 批量修改物理服务器电源状态 |
| `CreateInstance` | POST | `/v1/{project_id}/instances` | 创建实例 |
| `DeleteInstance` | DELETE | `/v1/{project_id}/instances/{id}` | 删除实例 |
| `DeleteInstances` | POST | `/v1/{project_id}/instances/batch-delete` | 批量删除实例 |
| `DownloadServerLogs` | GET | `/v1/{project_id}/physicalservers/{id}/logs/exports/{export_id}/content` | 下载日志文件 |
| `ExportServerLogs` | POST | `/v1/{project_id}/physicalservers/{id}/logs/exports` | 导出服务器日志请求 |
| `ListAlarms` | GET | `/v1/{project_id}/physicalservers/alarms` | 服务器告警列表 |
| `ListEvents` | GET | `/v1/{project_id}/physicalservers/events` | 服务器事件列表 |
| `ListIDcs` | GET | `/v1/{project_id}/idcs` | 查询 IDC 列表 |
| `ListInstances` | GET | `/v1/{project_id}/instances` | 批量查询实例 |
| `ListIRacks` | GET | `/v1/{project_id}/iracks` | 查询 iRack 实例列表 |
| `ListServers` | GET | `/v1/{project_id}/physicalservers` | 批量查询物理服务器 |
| `ModifyInstanceIp` | PUT | `/v1/{project_id}/instances/{id}/ip` | 修改实例ip |
| `ReinstallOs` | PUT | `/v1/{project_id}/instances/reinstall` | 批量重新安装OS |
| `RunInstances` | POST | `/v1/{project_id}/instances/batch-create` | 批量创建实例 |
| `ShowAlarmSummary` | GET | `/v1/{project_id}/physicalservers/alarms/summary` | 服务器告警概览 |
| `ShowAlarmTrend` | GET | `/v1/{project_id}/physicalservers/alarms/trend` | 服务器告警趋势 |
| `ShowEvent` | GET | `/v1/{project_id}/physicalservers/events/{event_id}` | 查询事件定义 |
| `ShowInstanceStatus` | GET | `/v1/{project_id}/instances/{id}/status` | 查询实例状态 |
| `ShowLogsExportStatus` | GET | `/v1/{project_id}/physicalservers/{id}/logs/exports/{export_id}` | 查询日志导出状态 |
| `ShowRemoteConsole` | GET | `/v1/{project_id}/physicalservers/{id}/remote-console-address` | 获取console地址信息 |
| `ShowServer` | GET | `/v1/{project_id}/physicalservers/{id}` | 查询物理服务器信息 |
| `ShowServerFirmwareAttributes` | GET | `/v1/{project_id}/physicalservers/{id}/firmware-attributes` | 查询服务器固件详细信息 |
| `ShowServerHardwareAttributes` | GET | `/v1/{project_id}/physicalservers/{id}/hardware-attributes` | 查询服务器硬件详细信息 |
| `ShowServerStatus` | GET | `/v1/{project_id}/physicalservers/status` | 服务器概览 |

... and 2 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
