---
name: huaweicloud-eds
description: HuaweiCloud EDS API guide. 11 APIs covering offer, user, 合约, 审计日志, 连接器.
---

# HuaweiCloud EDS API Guide

11 APIs. Tags: offer, user, 合约, 审计日志, 连接器

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddConnectorUser` | POST | `/v1/{project_id}/eds/instances/{instance_id}/connectors/{connector_id}/users` | 分配子账号 |
| `CancelContract` | DELETE | `/v1/{project_id}/eds/instances/{instance_id}/connectors/{connector_id}/contracts/{contract_id}` | 终止合约 |
| `CommitContract` | POST | `/v1/{project_id}/eds/instances/{instance_id}/connectors/{connector_id}/contracts` | 提交合约 |
| `DeleteConnectorUser` | DELETE | `/v1/{project_id}/eds/instances/{instance_id}/connectors/{connector_id}/users/{user_name}` | 账号回收 |
| `ListConnectorsByInstanceManger` | GET | `/v1/{project_id}/eds/instances/{instance_id}/connectors` | 按空间管理员查询连接器列表 |
| `ListConnectorsByInstanceUser` | GET | `/v1/{project_id}/eds/instances/{instance_id}/connectors/user-connectors` | 按空间用户查询连接器列表 |
| `ListOffers` | POST | `/v1/{project_id}/eds/instances/{instance_id}/connectors/{connector_id}/offers` | 查询指定连接器下的offer列表 |
| `ShowAuditLog` | GET | `/v1/{project_id}/eds/instances/{instance_id}/connectors/{connector_id}/audit-logs` | 查询数据资产的审计日志 |
| `ShowConnector` | GET | `/v1/{project_id}/eds/instances/{instance_id}/connectors/{connector_id}` | 查询指定租户的连接器详情 |
| `ShowContract` | GET | `/v1/{project_id}/eds/instances/{instance_id}/connectors/{connector_id}/contracts/{contract_id}` | 查询合约 |
| `ShowOffer` | GET | `/v1/{project_id}/eds/instances/{instance_id}/connectors/{connector_id}/offers/{offer_id}` | 查询指定offer详情 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
