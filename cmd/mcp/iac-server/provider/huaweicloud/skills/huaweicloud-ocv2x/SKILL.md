---
name: huaweicloud-ocv2x
description: HuaweiCloud ocv2x API guide. 4 APIs covering APIG-SN-DataAgingConfig, APIG-SN-HistoryTrafficEvents, APIG-SN-VehicleHistory.
---

# HuaweiCloud ocv2x API Guide

4 APIs. Tags: APIG-SN-DataAgingConfig, APIG-SN-HistoryTrafficEvents, APIG-SN-VehicleHistory

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `getDataAgingConfig` | GET | `/v1/{project_id}/v2x/{v2x_id}/aging-config` | 查询老化时间配置 |
| `getHistoryTrafficEvents` | GET | `/v1/{project_id}/v2x/{v2x_id}/history-traffic-events` | 条件查询交通事件 |
| `getVehicleHistory` | GET | `/v1/{project_id}/v2x/{v2x_id}/vehicle-safety-data` | 条件查询车辆数据 |
| `putDataAgingConfig` | PUT | `/v1/{project_id}/v2x/{v2x_id}/aging-config` | 修改历史数据老化时间 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
