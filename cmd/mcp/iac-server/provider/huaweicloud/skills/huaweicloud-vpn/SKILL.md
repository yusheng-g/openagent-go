---
name: huaweicloud-vpn
description: HuaweiCloud VPN API guide. 89 APIs covering ClientCaCertificate, ConnectionMonitor, CustomerGateway, P2cVpnGateway, Tags.
---

# HuaweiCloud VPN API Guide

89 APIs. Tags: ClientCaCertificate, ConnectionMonitor, CustomerGateway, P2cVpnGateway, Tags, VpnAccessPolicy, VpnConnection, VpnConnectionsLogConfig, VpnGateway, VpnGatewayCertificate, VpnQuota, VpnServer, VpnUser, VpnUserGroup

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddVpnUsersToGroup` | POST | `/v5/{project_id}/p2c-vpn-gateways/vpn-servers/{vpn_server_id}/groups/{group_id}/add-users` | 添加VPN用户到组 |
| `BatchCreateResourceTags` | POST | `/v5/{project_id}/{resource_type}/{resource_id}/tags/create` | 批量添加资源标签 |
| `BatchCreateVpnConnection` | POST | `/v5/{project_id}/vpn-connections/batch-create` | 批量创建VPN连接 |
| `BatchCreateVpnUsers` | POST | `/v5/{project_id}/p2c-vpn-gateways/vpn-servers/{vpn_server_id}/users/batch-create` | 批量创建VPN用户 |
| `BatchDeleteResourceTags` | POST | `/v5/{project_id}/{resource_type}/{resource_id}/tags/delete` | 批量删除资源标签 |
| `BatchDeleteVpnUsers` | POST | `/v5/{project_id}/p2c-vpn-gateways/vpn-servers/{vpn_server_id}/users/batch-delete` | 批量删除VPN用户 |
| `CheckClientCaCertificate` | POST | `/v5/{project_id}/p2c-vpn-gateways/vpn-servers/client-ca-certificates/check` | 校验客户端CA |
| `CountResourcesByTags` | POST | `/v5/{project_id}/{resource_type}/resource-instances/count` | 查询资源实例数量 |
| `CreateCgw` | POST | `/v5/{project_id}/customer-gateways` | 创建对端网关 |
| `CreateConnectionMonitor` | POST | `/v5/{project_id}/connection-monitors` | 创建VPN连接监控 |
| `CreateP2cVgw` | POST | `/v5/{project_id}/p2c-vpn-gateways` | 创建P2C VPN网关 |
| `CreateVgw` | POST | `/v5/{project_id}/vpn-gateways` | 创建VPN网关 |
| `CreateVgwCertificate` | POST | `/v5/{project_id}/vpn-gateways/{vgw_id}/certificate` | 导入VPN网关证书 |
| `CreateVpnAccessPolicy` | POST | `/v5/{project_id}/p2c-vpn-gateways/vpn-servers/{vpn_server_id}/access-policies` | 创建VPN访问策略 |
| `CreateVpnConnection` | POST | `/v5/{project_id}/vpn-connection` | 创建VPN连接 |
| `CreateVpnServer` | POST | `/v5/{project_id}/p2c-vpn-gateways/{p2c_vgw_id}/vpn-servers` | 创建一个VPN 服务端 |
| `CreateVpnUser` | POST | `/v5/{project_id}/p2c-vpn-gateways/vpn-servers/{vpn_server_id}/users` | 创建VPN用户 |
| `CreateVpnUserGroup` | POST | `/v5/{project_id}/p2c-vpn-gateways/vpn-servers/{vpn_server_id}/groups` | 创建VPN用户组 |
| `DeleteCgw` | DELETE | `/v5/{project_id}/customer-gateways/{customer_gateway_id}` | 删除对端网关 |
| `DeleteClientCa` | DELETE | `/v5/{project_id}/p2c-vpn-gateways/vpn-servers/{vpn_server_id}/client-ca-certificates/{client_ca_certificate_id}` | 删除客户端的CA证书 |
| `DeleteConnectionMonitor` | DELETE | `/v5/{project_id}/connection-monitors/{connection_monitor_id}` | 删除VPN连接监控 |
| `DeleteP2cVgw` | DELETE | `/v5/{project_id}/p2c-vpn-gateways/{p2c_vgw_id}` | 删除P2C VPN网关 |
| `DeleteP2cVgwConnection` | POST | `/v5/{project_id}/p2c-vpn-gateways/{p2c_vgw_id}/connections/{connection_id}/disconnect` | 断开P2C VPN网关连接 |
| `DeleteP2cVpnGatewayJob` | DELETE | `/v5/{project_id}/p2c-vpn-gateways/jobs/{job_id}` | 删除指定任务的记录 |
| `DeleteVgw` | DELETE | `/v5/{project_id}/vpn-gateways/{vgw_id}` | 删除VPN网关 |
| `DeleteVpnAccessPolicy` | DELETE | `/v5/{project_id}/p2c-vpn-gateways/vpn-servers/{vpn_server_id}/access-policies/{policy_id}` | 删除VPN访问策略 |
| `DeleteVpnConnection` | DELETE | `/v5/{project_id}/vpn-connection/{vpn_connection_id}` | 删除VPN连接 |
| `DeleteVpnConnectionsLogConfig` | DELETE | `/v5/{project_id}/p2c-vpn-gateways/{p2c_vgw_id}/log-config` | 删除VPN连接日志配置 |
| `DeleteVpnGatewayJob` | DELETE | `/v5/{project_id}/vpn-gateways/jobs/{job_id}` | 删除指定任务的记录 |
| `DeleteVpnUser` | DELETE | `/v5/{project_id}/p2c-vpn-gateways/vpn-servers/{vpn_server_id}/users/{user_id}` | 删除VPN用户 |

... and 59 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
