---
name: huaweicloud-mssi
description: HuaweiCloud MSSI API guide. 16 APIs covering 流, 流模板, 连接, 连接器.
---

# HuaweiCloud MSSI API Guide

16 APIs. Tags: 流, 流模板, 连接, 连接器

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateConnectionInfo` | POST | `/v1/{project_id}/connections` | 新建Connection |
| `CreateCustomConnectorFromOpenapi` | POST | `/v2/{project_id}/connectors/custom-connectors` | 新建自定义连接器(导入swagger方式) |
| `CreateFlow` | POST | `/v1/{project_id}/flows` | 创建flow |
| `CreateFlowTemplateFromFlow` | POST | `/v1/{project_id}/flows/{flow_id}/templates` | 根据流创建Flow模板 |
| `DeleteConnectionInfo` | DELETE | `/v1/{project_id}/connections/{connect_id}` | 删除Connection |
| `DeleteCustomConnector` | DELETE | `/v2/{project_id}/connectors/custom-connectors/{connector_id}` | 删除自定义连接器 |
| `DeleteFlow` | DELETE | `/v1/{project_id}/flows/{flow_id}` | 删除Flow |
| `SearchFlowById` | GET | `/v2/{project_id}/flows/{flow_id}` | 查询特定flow |
| `ShowAllConnections` | GET | `/v1/{project_id}/connections` | 查询Connection列表 |
| `ShowAllFlows` | GET | `/v1/{project_id}/flows` | 查询所有Flow |
| `ShowConnectors` | GET | `/v2/{project_id}/connectors` | 查询Connector列表 |
| `ShowCustomConnector` | POST | `/v2/{project_id}/connectors/custom-connectors/{connector_id}/release` | 发布连接器 |
| `ShowCustomConnectors` | GET | `/v2/{project_id}/connectors/custom-connectors` | 查询CustomConnector列表 |
| `ShowSingleConnection` | GET | `/v1/{project_id}/connections/{connect_id}` | 查询单个Connection |
| `UpdateConnectionInfo` | PUT | `/v1/{project_id}/connections/{connect_id}` | 修改连接配置内容 |
| `UpdateFlow` | PUT | `/v1/{project_id}/flows/{flow_id}` | 更新flow |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
