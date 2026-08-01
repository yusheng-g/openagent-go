---
name: huaweicloud-iec
description: HuaweiCloud IEC API guide. 90 APIs covering monitor, port, subnet, 子网, 安全组.
---

# HuaweiCloud IEC API Guide

90 APIs. Tags: monitor, port, subnet, 子网, 安全组, 密钥对, 带宽, 弹性公网IP, 端口, 网络ACL, 虚拟私有云, 路由表, 边缘业务, 边缘实例, 边缘硬盘, 边缘站点, 边缘规格, 边缘镜像, 配额

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddNics` | POST | `/v1/cloudservers/{instance_id}/nics` | 添加网卡 |
| `AssociateSubnet` | POST | `/v1/routetables/{routetable_id}/associate-subnets` | 路由表关联子网 |
| `AttachVipBandwidth` | POST | `/v1/vports/{vport_id}/bandwidth/attach` | 端口绑定带宽 |
| `BatchListMetricData` | POST | `/v1/batch-query-metric-data` | 批量查询监控数据 |
| `BatchRebootInstance` | POST | `/v1/cloudservers/action` | 批量重启边缘实例 |
| `BatchStartInstance` | POST | `/v1/cloudservers/action` | 批量启动边缘实例 |
| `BatchStopInstance` | POST | `/v1/cloudservers/action` | 批量关机边缘实例 |
| `ChangeOs` | POST | `/v1/cloudservers/{instance_id}/change-os` | 切换操作系统 |
| `CreateDeployment` | POST | `/v1/deployments` | 创建部署计划 |
| `CreateFirewall` | POST | `/v1/firewalls` | 创建网络ACL |
| `CreateImage` | POST | `/v1/images/create` | 从边缘实例创建边缘私有镜像 |
| `CreateInstance` | POST | `/v1/cloudservers` | 创建边缘实例 |
| `CreateKeypair` | POST | `/v1/os-keypairs` | 创建和导入密钥 |
| `CreatePort` | POST | `/v1/ports` | 创建端口 |
| `CreatePublicIp` | POST | `/v1/publicips` | 创建弹性公网IP |
| `CreateRoutes` | POST | `/v1/routetables/{routetable_id}/add-routes` | 创建路由 |
| `CreateRoutetable` | POST | `/v1/routetables` | 创建路由表 |
| `CreateSecurityGroup` | POST | `/v1/security-groups` | 创建边缘安全组 |
| `CreateSecurityGroupRule` | POST | `/v1/security-group-rules` | 创建安全组规则 |
| `CreateSubnet` | POST | `/v1/subnets` | 创建子网 |
| `CreateVpc` | POST | `/v1/vpcs` | 创建虚拟私有云 |
| `DeleteBandwidth` | DELETE | `/v1/bandwidths/{bandwidth_id}` | 删除带宽 |
| `DeleteDeployment` | DELETE | `/v1/deployments/{deployment_id}` | 删除部署计划 |
| `DeleteEdgeCloud` | DELETE | `/v1/edgeclouds/{edgecloud_id}` | 删除边缘业务 |
| `DeleteFirewall` | DELETE | `/v1/firewalls/{firewall_id}` | 删除网络ACL |
| `DeleteImage` | DELETE | `/v1/images/{image_id}` | 删除边缘私有镜像 |
| `DeleteInstances` | POST | `/v1/cloudservers/delete` | 批量删除边缘实例 |
| `DeleteKeypair` | DELETE | `/v1/os-keypairs/{keypair_name}` | 删除密钥 |
| `DeleteNics` | POST | `/v1/cloudservers/{instance_id}/nics/delete` | 删除网卡 |
| `DeletePort` | DELETE | `/v1/ports/{port_id}` | 删除端口 |

... and 60 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
