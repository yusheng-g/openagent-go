---
name: huaweicloud-vpc
description: HuaweiCloud VPC API guide. 192 APIs covering IP地址组, OpenStack - API版本信息, OpenStack - 子网, OpenStack - 安全组, OpenStack - 端口. Detailed swagger definitions in references/<APIName>.json.
---

# HuaweiCloud VPC API Guide

192 APIs. Tags: IP地址组, OpenStack - API版本信息, OpenStack - 子网, OpenStack - 安全组, OpenStack - 端口, OpenStack - 网络, OpenStack - 网络ACL, OpenStack - 路由器, VPC, VPC资源标签管理, VPC路由, 子网, 子网资源标签管理, 子网预留网段, 安全组, 安全组规则, 安全组资源标签管理, 对等连接, 查询网络IP使用情况, 流日志, 流量镜像会话, 流量镜像筛选条件, 流量镜像筛选规则, 私有IP, 端口, 端口资源标签管理, 网络ACL, 网络ACL资源标签管理, 虚拟子网, 路由表, 辅助弹性网卡, 配额

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptVpcPeering` | PUT | `/v2.0/vpc/peerings/{peering_id}/accept` | 接受对等连接请求 |
| `AddFirewallRules` | PUT | `/v3/{project_id}/vpc/firewalls/{firewall_id}/insert-rules` | 网络ACL插入规则 |
| `AddSecurityGroups` | PUT | `/v3/{project_id}/ports/{port_id}/insert-security-groups` | 端口插入安全组 |
| `AddSourcesToTrafficMirrorSession` | PUT | `/v3/{project_id}/vpc/traffic-mirror-sessions/{traffic_mirror_session_id}/add-sources` | 流量镜像会话添加镜像源 |
| `AddVpcExtendCidr` | PUT | `/v3/{project_id}/vpc/vpcs/{vpc_id}/add-extend-cidr` | 添加VPC扩展网段 |
| `AssociateRouteTable` | POST | `/v1/{project_id}/routetables/{routetable_id}/action` | 子网关联路由表 |
| `AssociateSubnetFirewall` | PUT | `/v3/{project_id}/vpc/firewalls/{firewall_id}/associate-subnets` | 网络ACL绑定子网 |
| `BatchCreateFirewallTags` | POST | `/v3/{project_id}/firewalls/{firewall_id}/tags/create` | 批量添加ACL资源标签 |
| `BatchCreatePortTags` | POST | `/v3/{project_id}/ports/{port_id}/tags/create` | 批量添加端口资源标签 |
| `BatchCreateSecurityGroupRules` | POST | `/v3/{project_id}/vpc/security-groups/{security_group_id}/security-group-rules/batch-create` | 批量创建安全组规则 |
| `BatchCreateSecurityGroupTags` | POST | `/v2.0/{project_id}/security-groups/{security_group_id}/tags/action` | 批量创建安全组资源标签 |
| `BatchCreateSubnetTags` | POST | `/v2.0/{project_id}/subnets/{subnet_id}/tags/action` | 批量创建子网资源标签 |
| `BatchCreateSubNetworkInterface` | POST | `/v3/{project_id}/vpc/sub-network-interfaces/batch-create` | 批量创建辅助弹性网卡 |
| `BatchCreateVpcTags` | POST | `/v2.0/{project_id}/vpcs/{vpc_id}/tags/action` | 批量创建VPC资源标签 |
| `BatchDeleteFirewallTags` | POST | `/v3/{project_id}/firewalls/{firewall_id}/tags/delete` | 批量删除ACL资源标签 |
| `BatchDeletePortTags` | POST | `/v3/{project_id}/ports/{port_id}/tags/delete` | 批量删除端口资源标签 |
| `BatchDeleteSecurityGroupTags` | POST | `/v2.0/{project_id}/security-groups/{security_group_id}/tags/action` | 批量删除安全组资源标签 |
| `BatchDeleteSubnetTags` | POST | `/v2.0/{project_id}/subnets/{subnet_id}/tags/action` | 批量删除子网资源标签 |
| `BatchDeleteVpcTags` | POST | `/v2.0/{project_id}/vpcs/{vpc_id}/tags/action` | 批量删除VPC资源标签 |
| `CountFirewallsByTags` | POST | `/v3/{project_id}/firewalls/resource-instances/count` | 查询ACL资源实例数量 |
| `CountPortsByTags` | POST | `/v3/{project_id}/ports/resource-instances/count` | 查询端口资源实例数量 |
| `CreateAddressGroup` | POST | `/v3/{project_id}/vpc/address-groups` | 创建地址组 |
| `CreateFirewall` | POST | `/v3/{project_id}/vpc/firewalls` | 创建网络ACL |
| `CreateFirewallTag` | POST | `/v3/{project_id}/firewalls/{firewall_id}/tags` | 添加ACL资源标签 |
| `CreateFlowLog` | POST | `/v1/{project_id}/fl/flow_logs` | 创建流日志 |
| `CreatePort` | POST | `/v1/{project_id}/ports` | 创建端口 |
| `CreatePortTag` | POST | `/v3/{project_id}/ports/{port_id}/tags` | 添加端口资源标签 |
| `CreatePrivateip` | POST | `/v1/{project_id}/privateips` | 申请私有IP |
| `CreateRouteTable` | POST | `/v1/{project_id}/routetables` | 创建路由表 |
| `CreateSecurityGroup` | POST | `/v1/{project_id}/security-groups` | 创建安全组 |

... and 162 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
