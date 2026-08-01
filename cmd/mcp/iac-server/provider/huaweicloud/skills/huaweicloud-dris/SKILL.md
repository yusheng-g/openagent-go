---
name: huaweicloud-dris
description: HuaweiCloud DRIS API guide. 58 APIs covering Edge应用版本管理, Edge应用管理, Edge管理, IPC管理, RSU型号管理.
---

# HuaweiCloud DRIS API Guide

58 APIs. Tags: Edge应用版本管理, Edge应用管理, Edge管理, IPC管理, RSU型号管理, RSU资源管理, 业务通道管理, 信号机管理, 即时交通事件分发, 历史交通消息管理, 数据转发配置管理, 车辆管理, 边缘应用管理, 长期交通事件管理, 雷达管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddForwardingConfigs` | POST | `/v1/{project_id}/forwarding-configs` | 创建数据转发配置 |
| `BatchShowEdgeApps` | GET | `/v1/{project_id}/v2x-edge-apps` | 查询应用列表 |
| `BatchShowEdgeAppVersions` | GET | `/v1/{project_id}/v2x-edge-apps/{edge_app_id}/versions` | 查询应用版本列表 |
| `BatchShowIpcs` | GET | `/v1/{project_id}/cameras` | 查询IPC列表 |
| `BatchShowRadars` | GET | `/v1/{project_id}/radars` | 查询雷达列表 |
| `BatchShowRsus` | GET | `/v1/{project_id}/rsus` | 查询RSU列表 |
| `BatchShowTrafficControllers` | GET | `/v1/{project_id}/traffic-controllers` | 查询信号机列表 |
| `BatchShowTrafficEvents` | GET | `/v1/{project_id}/traffic-events` | 查询长期交通事件列表 |
| `BatchShowVehicles` | GET | `/v1/{project_id}/vehicles` | 查询车辆列表 |
| `CreateDataChannel` | POST | `/v1/{project_id}/v2x-edges/{v2x_edge_id}/data-channel` | 创建业务通道 |
| `CreateEdgeApp` | POST | `/v1/{project_id}/v2x-edge-apps` | 创建应用 |
| `CreateEdgeApplicationVersion` | POST | `/v1/{project_id}/v2x-edge-apps/{edge_app_id}/versions` | 创建应用版本 |
| `CreateRsu` | POST | `/v1/{project_id}/rsus` | 创建RSU |
| `CreateRsuModel` | POST | `/v1/{project_id}/rsu-models` | 创建RSU型号 |
| `CreateTrafficController` | POST | `/v1/{project_id}/traffic-controllers` | 创建信号机 |
| `CreateTrafficEvent` | POST | `/v1/{project_id}/traffic-events` | 创建长期交通事件 |
| `CreateV2xEdge` | POST | `/v1/{project_id}/v2x-edges` | 创建Edge |
| `CreateV2xEdgeApp` | POST | `/v1/{project_id}/v2x-edges/{v2x_edge_id}/apps` | 部署边缘应用 |
| `CreateVehicle` | POST | `/v1/{project_id}/vehicles` | 创建车辆 |
| `DeleteDataChannel` | DELETE | `/v1/{project_id}/v2x-edges/{v2x_edge_id}/data-channel` | 删除业务通道 |
| `DeleteEdgeApp` | DELETE | `/v1/{project_id}/v2x-edge-apps/{edge_app_id}` | 删除应用 |
| `DeleteEdgeApplicationVersion` | DELETE | `/v1/{project_id}/v2x-edge-apps/{edge_app_id}/versions/{version}` | 删除应用版本 |
| `DeleteForwardingConfig` | DELETE | `/v1/{project_id}/forwarding-configs/{forwarding_config_id}` | 删除数据转发配置 |
| `DeleteRsu` | DELETE | `/v1/{project_id}/rsus/{rsu_id}` | 删除RSU |
| `DeleteRsuModel` | DELETE | `/v1/{project_id}/rsu-models/{rsu_model_id}` | 删除RSU型号 |
| `DeleteTrafficController` | DELETE | `/v1/{project_id}/traffic-controllers/{traffic_controller_id}` | 删除信号机 |
| `DeleteTrafficEvent` | DELETE | `/v1/{project_id}/traffic-events/{event_id}` | 删除长期交通事件 |
| `DeleteV2XEdgeAppByEdgeAppId` | DELETE | `/v1/{project_id}/v2x-edges/{v2x_edge_id}/apps/{edge_app_id}` | 删除边缘应用 |
| `DeleteV2XEdgeByV2xEdgeId` | DELETE | `/v1/{project_id}/v2x-edges/{v2x_edge_id}` | 删除Edge |
| `DeleteVehicle` | DELETE | `/v1/{project_id}/vehicles/{vehicle_id}` | 删除车辆 |

... and 28 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
