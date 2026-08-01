---
name: huaweicloud-gsl
description: HuaweiCloud GSL API guide. 39 APIs covering SIM卡套餐实例管理, SIM卡管理, 三网卡策略管理, 三网卡管理, 业务受理管理.
---

# HuaweiCloud GSL API Guide

39 APIs. Tags: SIM卡套餐实例管理, SIM卡管理, 三网卡策略管理, 三网卡管理, 业务受理管理, 后向流量池管理, 套餐管理, 标签管理, 流量池管理, 短信套餐管理, 自定义属性管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddNetworkSwitchPolicy` | POST | `/v1/network-switch-policies` | 新增网络切换策略 |
| `BatchSetAttributes` | POST | `/v1/sim-cards/attributes/batch-set` | 批量设置自定义属性接口 |
| `BatchSetTags` | POST | `/v1/sim-tags/batch-set` | 批量设置/取消设置标签接口 |
| `CreateAttribute` | POST | `/v1/attributes` | 用户新增自定义属性接口 |
| `CreateTag` | POST | `/v1/tags` | 用户添加标签 |
| `DeleteRealName` | POST | `/v1/sim-cards/{sim_card_id}/clear-real-name` | 清除实名认证信息 |
| `DeleteTag` | DELETE | `/v1/tags/{tag_id}` | 删除标签 |
| `DisableAttribute` | POST | `/v1/attributes/{attribute_id}/disable` | 停用自定义属性接口 |
| `EnableAttribute` | POST | `/v1/attributes/{attribute_id}/enable` | 启用自定义属性接口 |
| `EnableSimCard` | POST | `/v1/sim-cards/{sim_card_id}/enable` | 激活实体卡 |
| `ListAttributes` | GET | `/v1/attributes` | 查询自定义属性列表接口 |
| `ListBackPoolMembers` | GET | `/v1/back-pools/{back_pool_id}/members` | 查询后向流量池成员列表 |
| `ListBackPools` | GET | `/v1/back-pools` | 查询后向流量池列表 |
| `ListFlowBySimCards` | POST | `/v1/sim-price-plans/usage/batch-query` | 批量查询实体卡流量 |
| `ListNetworkSwitchPolicies` | GET | `/v1/network-switch-policies` | 查询策略列表 |
| `ListProPricePlans` | GET | `/v1/price-plans` | 查询套餐列表信息 |
| `ListSimCardFlowPerDay` | POST | `/v1/sim-cards/batch-daily-flow` | 批量查询SIM卡日用量 |
| `ListSimCards` | GET | `/v1/sim-cards` | 查询SIM卡列表 |
| `ListSimDeviceMultiply` | GET | `/v1/sim-cards-multiply` | 查询三网卡列表 |
| `ListSimPoolMembers` | GET | `/v1/sim-pools/{sim_pool_id}/members` | 查询流量池成员列表 |
| `ListSimPools` | GET | `/v1/sim-pools` | 查询流量池列表 |
| `ListSimPricePlans` | GET | `/v1/sim-price-plans` | sim卡套餐列表查询 |
| `ListSmsDetails` | GET | `/v1/sms-send-infos/details` | 短信发送详情 |
| `ListTags` | GET | `/v1/tags` | 查询标签列表 |
| `ListWorkOrderDetails` | GET | `/v1/work-orders/{work_order_id}/details` | 分页查询业务受理明细 |
| `ListWorkOrders` | GET | `/v1/work-orders` | 分页查询业务受理单 |
| `RegisterImei` | POST | `/v1/sim-cards/{sim_card_id}/bind-device` | SIM卡机卡重绑 |
| `ResetSimCard` | POST | `/v1/sim-cards/{sim_card_id}/reset` | SIM卡单卡复机 |
| `SendSms` | POST | `/v1/sms-send-infos` | 发送短信 |
| `SetExceedCutNet` | POST | `/v1/sim-cards/{sim_card_id}/exceed-cut-net` | SIM卡达量断网/取消达量断网 |

... and 9 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
