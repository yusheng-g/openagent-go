---
name: huaweicloud-ec
description: HuaweiCloud EC API guide. 54 APIs covering EcnAccessPoint, EnterpriseConnectNetwork, Equipment, EquipmentLan, EquipmentOspf.
---

# HuaweiCloud EC API Guide

54 APIs. Tags: EcnAccessPoint, EnterpriseConnectNetwork, Equipment, EquipmentLan, EquipmentOspf, EquipmentStaticRoute, EquipmentWan, EquipmentWlan, ErRelationship, IntelligentEnterpriseGateway, Quota, VpcRelationship, VrrpConfig

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddEcnWithEr` | POST | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/relationship/enterprise-router` | 关联企业路由器到企业连接网络 |
| `AddEcnWithIeg` | POST | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/relationship/intelligent-enterprise-gateway` | 绑定智能企业网关到企业连接网络 |
| `AddEcnWithVpc` | POST | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/relationship/vpc` | 关联虚拟私有云到企业连接网络 |
| `AddVrrpConfig` | POST | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/vrrp-config` | 创建vrrp配置 |
| `ChangeIegPassword` | PUT | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/password` | 修改IEG设备admin账户密码 |
| `CreateEcnAccessPoint` | POST | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/access-point` | 添加新的接入点 |
| `CreateEquipment` | POST | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/equipment` | 激活智能企业网关设备 |
| `CreateEquipmentLanConfig` | POST | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/equipment/{equipment_id}/lan-interface` | 创建智能企业网关设备LAN口配置 |
| `CreateEquipmentStaticRouteConfig` | POST | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/equipment/{equipment_id}/static-route` | 创建智能企业网关设备静态路由配置 |
| `DeleteEcnAccessPoint` | DELETE | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/access-point/{access_point_id}` | 删除接入点 |
| `DeleteEcnWithEr` | DELETE | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/relationship/enterprise-router/{relation_id}` | 解除企业路由器和企业连接网络的关联 |
| `DeleteEcnWithIeg` | DELETE | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/relationship/intelligent-enterprise-gateway/{relation_id}` | 解除智能企业网关和企业连接网络的绑定 |
| `DeleteEcnWithVpc` | DELETE | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/relationship/vpc/{relation_id}` | 解除虚拟私有云和企业连接网络的关联 |
| `DeleteEquipment` | DELETE | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/equipment/{equipment_id}` | 移除智能企业网关设备 |
| `DeleteEquipmentLanConfig` | DELETE | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/equipment/{equipment_id}/lan-interface` | 删除智能企业网关设备LAN口配置 |
| `DeleteEquipmentStaticRouteConfig` | DELETE | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/equipment/{equipment_id}/static-route` | 删除智能企业网关设备静态路由配置 |
| `DeleteVrrpConfig` | DELETE | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/vrrp-config/{virtual_router_id}` | 删除vrrp配置 |
| `GenerateInitialConfiguration` | POST | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/equipment/{equipment_id}/initial-configuration` | 生成智能企业网关设备初始配置 |
| `ListEcn` | GET | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network` | 查询企业连接网络列表 |
| `ListEcnAccessPointByEcnId` | GET | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/access-point` | 查询接入点 |
| `ListEcnWithEr` | GET | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/relationship/enterprise-router` | 查询企业连接网络网与企业路由器关联关系 |
| `ListEcnWithIeg` | GET | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/relationship/intelligent-enterprise-gateway` | 查询企业连接网络与智能企业网关绑定关系 |
| `ListEcnWithVpc` | GET | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/relationship/vpc` | 查询企业连接网络与虚拟私有云关联关系 |
| `ListEquipmentInterfaceName` | GET | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/equipment/{equipment_id}/interface-name` | 查询智能企业网关已配置的接口名字 |
| `ListEquipments` | GET | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/equipment` | 查询智能企业网关设备列表 |
| `ListIeg` | GET | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway` | 查询租户智能企业网关列表 |
| `RebootEquipment` | POST | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/equipment/{equipment_id}/reboot` | 重启智能企业网关设备 |
| `ShowEcnInfo` | GET | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}` | 查询企业连接网络 |
| `ShowEcnWithIeg` | GET | `/v1/{domain_id}/enterprise-connect/enterprise-connect-network/{ecn_id}/relationship/intelligent-enterprise-gateway/{relation_id}` | 查询企业连接网络与单个智能企业网关绑定关系 |
| `ShowEquipmentDnsInfo` | GET | `/v1/{domain_id}/enterprise-connect/intelligent-enterprise-gateway/{ieg_id}/equipment/{equipment_id}/lan-interface/dns` | 查询智能企业网关设备主备DNS配置 |

... and 24 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
