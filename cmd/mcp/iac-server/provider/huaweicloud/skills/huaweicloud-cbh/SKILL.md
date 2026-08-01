---
name: huaweicloud-cbh
description: HuaweiCloud CBH API guide. 50 APIs covering 云堡垒机信息查询, 云堡垒机管理, 可用区查询, 委托授权, 操作管理.
---

# HuaweiCloud CBH API Guide

50 APIs. Tags: 云堡垒机信息查询, 云堡垒机管理, 可用区查询, 委托授权, 操作管理, 标签管理, 生命周期管理, 网络管理, 获取IAM登录实例链接, 规格管理, 订单管理, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateInstanceTag` | POST | `/v2/{project_id}/cbs/instance/{resource_id}/tags/action` | 操作堡垒机实例资源标签 |
| `ChangeInstanceNetwork` | POST | `/v1/{project_id}/cbs/{server_id}/network/change` | 修改实例网络 |
| `ChangeInstanceOrder` | GET | `/v1/{project_id}/cbs/{server_id}/alter/{instance_key}` | 创建变更云堡垒机实例订单 |
| `ChangeInstanceType` | PUT | `/v2/{project_id}/cbs/instance/type` | 修改单机堡垒机实例类型 |
| `CountInstancesByTag` | POST | `/v2/{project_id}/cbs/instance/count` | 统计符合标签条件的实例数量 |
| `CreateCbh` | POST | `/v1/{project_id}/cbs/instance/create` | 创建云堡垒机实例 |
| `CreateInstance` | POST | `/v2/{project_id}/cbs/instance` | 创建堡垒机实例 |
| `CreateInstanceOrder` | POST | `/v1/{project_id}/cbs/period/order` | 创建云堡垒机实例订单 |
| `DeleteInstance` | DELETE | `/v2/{project_id}/cbs/instance` | 删除故障云堡垒机实例 |
| `InstallCbhEip` | POST | `/v1/{project_id}/cbs/instance/{server_id}/eip/bind` | 绑定弹性公网IP |
| `InstallInstanceEip` | POST | `/v2/{project_id}/cbs/instance/{server_id}/eip/bind` | 堡垒机实例绑定弹性公网IP |
| `ListAvailableZones` | GET | `/v2/{project_id}/cbs/available-zone` | 获取服务可用区信息 |
| `ListCbhInstance` | GET | `/v1/{project_id}/cbs/instance/list` | 获取CBH实例列表 |
| `ListInstances` | GET | `/v2/{project_id}/cbs/instance/list` | 获取堡垒机实例列表 |
| `ListInstancesByTag` | POST | `/v2/{project_id}/cbs/instance/filter` | 使用标签过滤实例 |
| `ListQuotaStatus` | GET | `/v1/{project_id}/cbs/instance/ecs-quota` | 获取弹性云服务器配额 |
| `ListSpecifications` | GET | `/v2/{project_id}/cbs/instance/specification` | 查询云堡垒机规格信息 |
| `ListSwitchConfigInfo` | GET | `/v2/{project_id}/cbs/feature/config` | 获取后端开关控制信息列表 |
| `ListTags` | GET | `/v2/{project_id}/cbs/instance/tags` | 查询租户在项目中的资源标签集合 |
| `LoginCbh` | POST | `/v1/{project_id}/cbs/instance/login` | 获取IAM登录实例链接 |
| `LoginInstance` | POST | `/v2/{project_id}/cbs/instance/login` | IAM用户登录堡垒机实例console |
| `LoginInstanceAdmin` | GET | `/v2/{project_id}/cbs/instances/{server_id}/admin-url` | 用户登录堡垒机实例admin的console |
| `RebootInstance` | POST | `/v2/{project_id}/cbs/instance/reboot` | 重启堡垒机实例 |
| `RegisterAuthorization` | POST | `/v2/{project_id}/cbs/agency/authorization` | 租户创建或取消云堡垒机服务的委托授权 |
| `ResetInstanceLoginMethod` | PUT | `/v2/{project_id}/cbs/instance/login-method` | 重置堡垒机实例admin登录方式 |
| `ResetInstancePassword` | PUT | `/v2/{project_id}/cbs/instance/password` | 重置堡垒机实例admin密码 |
| `ResetLoginMethod` | PUT | `/v1/{project_id}/cbs/instance/{server_id}/login-method` | 重置admin用户多因子认证方式 |
| `ResetPassword` | PUT | `/v1/{project_id}/cbs/instance/password` | 修改admin用户密码 |
| `ResizeInstance` | PUT | `/v2/{project_id}/cbs/instance` | 变更堡垒机实例 |
| `RestartCbhInstance` | POST | `/v1/{project_id}/cbs/instance/reboot` | 重启云堡垒机实例 |

... and 20 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
