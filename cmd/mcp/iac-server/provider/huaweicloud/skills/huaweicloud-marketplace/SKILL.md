---
name: huaweicloud-marketplace
description: HuaweiCloud Marketplace API guide. 6 APIs covering 查询订单, 购买商品, 资源实例.
---

# HuaweiCloud Marketplace API Guide

6 APIs. Tags: 查询订单, 购买商品, 资源实例

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CancelOrders` | POST | `/api/mkp-openapi-public/global/v1/order/batch-cancel` | 取消订单 |
| `CreateOrders` | POST | `/api/mkp-openapi-public/global/v1/order/batch-order` | 创建订单 |
| `ListOrders` | POST | `/api/mkp-openapi-public/global/v1/query-orders` | 查询订单列表 |
| `ShowOrderDetails` | POST | `/api/mkp-openapi-public/global/v1/order-detail/query` | 查询订单详情 |
| `ShowSaasInstance` | POST | `/api/mkp-openapi-public/global/v1/saas-instance/query` | 查询Saas资源实例 |
| `ShowSubscriptionAgreements` | POST | `/api/mkp-openapi-public/global/v1/subscription-agreements-query` | 查询下单待签署协议 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
