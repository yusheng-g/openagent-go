---
name: huaweicloud-er
description: HuaweiCloud ER API guide. 49 APIs covering VPC连接, 企业路由器, 传播, 关联, 其他连接.
---

# HuaweiCloud ER API Guide

49 APIs. Tags: VPC连接, 企业路由器, 传播, 关联, 其他连接, 可用区, 标签, 流日志, 路由, 路由表, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptAttachment` | POST | `/v3/{project_id}/enterprise-router/{er_id}/attachments/{attachment_id}/accept` | 接受共享连接创建 |
| `AssociateRouteTable` | POST | `/v3/{project_id}/enterprise-router/{er_id}/route-tables/{route_table_id}/associate` | 创建路由关联 |
| `BatchCreateResourceTags` | POST | `/v3/{project_id}/{resource_type}/{resource_id}/tags/action` | 批量添加删除资源标签 |
| `ChangeAssociationRoutePolicy` | POST | `/v3/{project_id}/enterprise-router/{er_id}/route-tables/{route_table_id}/associations/{association_id}/change-route-policy` | 修改关联的路由策略 |
| `ChangeAvailabilityZone` | POST | `/v3/{project_id}/enterprise-router/instances/{er_id}/change-availability-zone-ids` | 更新企业路由器的可用区信息 |
| `CreateEnterpriseRouter` | POST | `/v3/{project_id}/enterprise-router/instances` | 创建企业路由器 |
| `CreateFlowLog` | POST | `/v3/{project_id}/enterprise-router/{er_id}/flow-logs` | 创建流日志 |
| `CreateResourceTag` | POST | `/v3/{project_id}/{resource_type}/{resource_id}/tags` | 创建资源标签 |
| `CreateRouteTable` | POST | `/v3/{project_id}/enterprise-router/{er_id}/route-tables` | 创建路由表 |
| `CreateStaticRoute` | POST | `/v3/{project_id}/enterprise-router/route-tables/{route_table_id}/static-routes` | 创建静态路由 |
| `CreateVpcAttachment` | POST | `/v3/{project_id}/enterprise-router/{er_id}/vpc-attachments` | 创建VPC连接 |
| `DeleteEnterpriseRouter` | DELETE | `/v3/{project_id}/enterprise-router/instances/{er_id}` | 删除企业路由器 |
| `DeleteFlowLog` | DELETE | `/v3/{project_id}/enterprise-router/{er_id}/flow-logs/{flow_log_id}` | 删除流日志 |
| `DeleteResourceTag` | DELETE | `/v3/{project_id}/{resource_type}/{resource_id}/tags/{key}` | 删除资源标签 |
| `DeleteRouteTable` | DELETE | `/v3/{project_id}/enterprise-router/{er_id}/route-tables/{route_table_id}` | 删除路由表 |
| `DeleteStaticRoute` | DELETE | `/v3/{project_id}/enterprise-router/route-tables/{route_table_id}/static-routes/{route_id}` | 删除静态路由 |
| `DeleteVpcAttachment` | DELETE | `/v3/{project_id}/enterprise-router/{er_id}/vpc-attachments/{vpc_attachment_id}` | 删除VPC连接 |
| `DisableFlowLog` | POST | `/v3/{project_id}/enterprise-router/{er_id}/flow-logs/{flow_log_id}/disable` | 关闭流日志 |
| `DisablePropagation` | POST | `/v3/{project_id}/enterprise-router/{er_id}/route-tables/{route_table_id}/disable-propagations` | 删除路由传播 |
| `DisassociateRouteTable` | POST | `/v3/{project_id}/enterprise-router/{er_id}/route-tables/{route_table_id}/disassociate` | 删除路由关联 |
| `EnableFlowLog` | POST | `/v3/{project_id}/enterprise-router/{er_id}/flow-logs/{flow_log_id}/enable` | 开启流日志 |
| `EnablePropagation` | POST | `/v3/{project_id}/enterprise-router/{er_id}/route-tables/{route_table_id}/enable-propagations` | 创建路由传播 |
| `ListAssociations` | GET | `/v3/{project_id}/enterprise-router/{er_id}/route-tables/{route_table_id}/associations` | 查询路由关联列表 |
| `ListAttachments` | GET | `/v3/{project_id}/enterprise-router/{er_id}/attachments` | 查询连接列表 |
| `ListAvailabilityZone` | GET | `/v3/{project_id}/enterprise-router/availability-zones` | 查询可用区列表 |
| `ListEffectiveRoutes` | GET | `/v3/{project_id}/enterprise-router/route-tables/{route_table_id}/routes` | 查询有效路由列表 |
| `ListEnterpriseRouters` | GET | `/v3/{project_id}/enterprise-router/instances` | 查询企业路由器列表 |
| `ListFlowLogs` | GET | `/v3/{project_id}/enterprise-router/{er_id}/flow-logs` | 查询流日志列表 |
| `ListProjectTags` | GET | `/v3/{project_id}/{resource_type}/tags` | 查询项目标签 |
| `ListPropagations` | GET | `/v3/{project_id}/enterprise-router/{er_id}/route-tables/{route_table_id}/propagations` | 查询路由传播列表 |

... and 19 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
