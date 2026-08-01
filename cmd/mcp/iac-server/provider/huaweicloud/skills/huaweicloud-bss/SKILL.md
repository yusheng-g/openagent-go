---
name: huaweicloud-bss
description: HuaweiCloud BSS (Business Support System) API guide. Covers pricing (pay-as-you-go via ListOnDemandResourceRatings, yearly/monthly via ListRateOnPeriodDetail), billing and costs (ListCosts, ListCustomerBillsMonthlyBreakDown, ShowCustomerMonthlySum, ListCustomerBillsFeeRecords), resource specs (flavor, size, region availability), orders, subscriptions, and enterprise project management. 91 BSS APIs with full swagger definitions under references/, pricing flow (ListServiceTypes → ListResourceTypes → ListResourceSpecs → rating query), and spec-to-price mapping.
---

# HuaweiCloud BSS API Guide

You are a HuaweiCloud pricing and billing expert. You can query resource prices, check billing/cost data, look up resource specs, and manage orders/subscriptions.

## BSS API

HuaweiCloud billing and pricing is exposed through the BSS (Business Support System) API. The base URL is `https://bss.myhuaweicloud.com`.

### Pricing-core APIs

These are the APIs you will most likely use for `estimate_cost`:

| API | Method | URI | Description |
|---|---|---|---|
| `ListServiceTypes` | GET | `/v2/products/service-types` | List all cloud service types (ECS, RDS, OBS, ...) |
| `ListResourceTypes` | GET | `/v2/products/resource-types` | List resource types for a service type |
| `ListResourceSpecs` | POST | `/v2/products/resource-specs-query` | Query resource specs (flavor, size, etc.) |
| `ListOnDemandResourceRatings` | POST | `/v2/bills/ratings/on-demand-resources` | Query pay-as-you-go (on-demand) prices |
| `ListRateOnPeriodDetail` | POST | `/v2/bills/ratings/period-resources/subscribe-rate` | Query monthly/yearly subscription prices |
| `ListCosts` | POST | `/v4/costs/cost-analysed-bills/query` | Query cost data |
| `ListCustomerBillsMonthlyBreakDown` | GET | `/v2/costs/cost-analysed-bills/monthly-breakdown` | Query monthly cost breakdown |
| `ListCustomerBillsFeeRecords` | GET | `/v2/bills/customer-bills/fee-records` | Query billing fee records |
| `ShowCustomerMonthlySum` | GET | `/v2/bills/customer-bills/monthly-sum` | Query monthly billing summary |

### Pricing flow

1. Use `ListServiceTypes` to find the `cloud_service_type` code for the resource (e.g. `hws.service.type.ec2` for ECS)
2. Use `ListResourceTypes` to find the `resource_type` code (e.g. `hws.resource.type.vm`)
3. Use `ListResourceSpecs` to confirm the resource spec (flavor_id, etc.) exists in the target region
4. Use `ListOnDemandResourceRatings` for pay-as-you-go prices, or `ListRateOnPeriodDetail` for monthly/yearly subscription prices

### Full API catalog

All 91 BSS APIs. APIs marked **[pricing-core]** are the most relevant for `estimate_cost`. For detailed request/response schemas, `read` the corresponding file in `references/<APIName>.json`.

| API | Method | URI | Description |
|---|---|---|---|
| `AutoRenewalResources` | POST | `/v2/orders/subscriptions/resources/autorenew/{resource_id}` | 设置包年/包月资源自动续费 |
| `CancelAutoRenewalResources` | DELETE | `/v2/orders/subscriptions/resources/autorenew/{resource_id}` | 取消包年/包月资源自动续费 |
| `CancelCustomerOrder` | PUT | `/v2/orders/customer-orders/cancel` | 取消待支付订单 |
| `CancelResourcesSubscription` | POST | `/v2/orders/subscriptions/resources/unsubscribe` | 退订包年/包月资源 |
| `ChangeEnterpriseRealnameAuthentication` | PUT | `/v2/customers/realname-auths/enterprise` | 申请实名认证变更 |
| `CheckUserIdentity` | POST | `/v2/partners/sub-customers/users/check-identity` | 校验客户注册信息 |
| `ClaimEnterpriseMultiAccountCoupon` | POST | `/v2/enterprises/multi-accounts/transfer-coupon` | 企业主账号向企业子账号拨款优惠券 |
| `CreateEnterpriseProjectAuth` | POST | `/v2/enterprises/enterprise-projects/authority` | 开通客户企业项目权限 |
| `CreateEnterpriseRealnameAuthentication` | POST | `/v2/customers/realname-auths/enterprise` | 申请企业实名认证 |
| `CreatePartnerCoupons` | POST | `/v2/promotions/benefits/partner-coupons` | 发放优惠券 |
| `CreatePersonalRealnameAuth` | POST | `/v2/customers/realname-auths/individual` | 申请个人实名认证 |
| `CreateSubCustomer` | POST | `/v2/partners/sub-customers` | 创建客户 |
| `CreateSubEnterpriseAccount` | POST | `/v2/enterprises/multi-accounts/sub-customers` | 创建企业子账号 |
| `ListCities` | GET | `/v2/systems/configs/cities` | 查询城市信息 |
| `ListConsumeSubCustomers` | POST | `/v2/bills/subcustomer-bills/res-fee-records/sub-customers/query` | 查询伙伴消费子客户列表 |
| `ListConversions` | GET | `/v2/bases/conversions` | 查询度量单位进制 |
| `ListCosts` | POST | `/v4/costs/cost-analysed-bills/query` | 查询成本数据 **[pricing-core]** |
| `ListCounties` | GET | `/v2/systems/configs/counties` | 查询区县信息 |
| `ListCouponQuotasRecords` | GET | `/v2/partners/coupon-quotas/records` | 查询代金券额度的发放回收记录 |
| `ListCustomerAccountChangeRecords` | GET | `/v2/accounts/customer-accounts/account-change-records` | 查询收支明细(客户) |
| `ListCustomerBillsFeeRecords` | GET | `/v2/bills/customer-bills/fee-records` | 查询流水账单 **[pricing-core]** |
| `ListCustomerBillsMonthlyBreakDown` | GET | `/v2/costs/cost-analysed-bills/monthly-breakdown` | 查询月度成本 **[pricing-core]** |
| `ListCustomerCouponChangeRecords` | GET | `/v2/promotions/benefits/account-change-records` | 查询优惠券收支明细 |
| `ListCustomerOnDemandResources` | POST | `/v2/partners/sub-customers/on-demand-resources/query` | 查询客户按需资源列表 |
| `ListCustomerOrders` | GET | `/v2/orders/customer-orders` | 查询订单列表 |
| `ListCustomersBalancesDetail` | POST | `/v2/accounts/customer-accounts/balances/batch-query` | 查询客户账户余额 |
| `ListCustomerselfResourceRecordDetails` | POST | `/v2/bills/customer-bills/res-records/query` | 查询资源详单 |
| `ListCustomerselfResourceRecords` | GET | `/v2/bills/customer-bills/res-fee-records` | 查询资源消费记录 |
| `ListEnterpriseMultiAccount` | GET | `/v2/enterprises/multi-accounts/retrieve-amount` | 查询企业子账号可回收余额 |
| `ListEnterpriseOrganizations` | GET | `/v2/enterprises/multi-accounts/enterprise-organizations` | 查询企业组织结构 |
| `ListEnterpriseSubCustomers` | GET | `/v2/enterprises/multi-accounts/sub-customers` | 查询企业子账号列表 |
| `ListFreeResourceInfos` | POST | `/v3/payments/free-resources/query` | 查询资源包列表 |
| `ListFreeResourcesUsageRecords` | GET | `/v2/bills/customer-bills/free-resources-usage-records` | 查询资源包使用明细 |
| `ListFreeResourceUsages` | POST | `/v2/payments/free-resources/usages/details/query` | 查询资源包使用量 |
| `ListIncentiveDiscountPolicies` | GET | `/v2/products/incentive-discount-policies` | 查询产品的折扣和激励策略 |
| `ListIndirectPartners` | POST | `/v2/partners/indirect-partners/query` | 查询云经销商列表 |
| `ListIssuedCouponQuotas` | GET | `/v2/partners/issued-coupon-quotas` | 查询已发放的代金券额度 |
| `ListIssuedPartnerCoupons` | GET | `/v2/promotions/benefits/partner-coupons` | 查询已发放的优惠券 |
| `ListMeasureUnits` | GET | `/v2/bases/measurements` | 查询度量单位列表 |
| `ListMultiAccountRetrieveCoupons` | GET | `/v2/enterprises/multi-accounts/retrieve-coupons` | 查询企业子账号可回收优惠券列表 |
| `ListMultiAccountTransferCoupons` | GET | `/v2/enterprises/multi-accounts/transfer-coupons` | 查询企业主账号可拨款优惠券列表 |
| `ListOnDemandResourceRatings` | POST | `/v2/bills/ratings/on-demand-resources` | 查询按需产品价格 **[pricing-core]** |
| `ListOrderCouponsByOrderId` | GET | `/v2/orders/customer-orders/order-coupons` | 查询订单可用优惠券 |
| `ListOrderDiscounts` | GET | `/v2/orders/customer-orders/order-discounts` | 查询订单可用折扣 |
| `ListPartnerAccountChangeRecords` | GET | `/v2/accounts/partner-accounts/account-change-records` | 查询收支明细 |
| `ListPartnerAdjustRecords` | GET | `/v3/accounts/partner-accounts/adjust-records` | 查询调账记录 |
| `ListPartnerBalances` | GET | `/v2/accounts/partner-accounts/balances` | 查询云经销商账户余额 |
| `ListPartnerCouponsRecord` | GET | `/v2/promotions/benefits/partner-coupons/records/query` | 查询优惠券的发放回收记录 |
| `ListPayPerUseCustomerResources` | POST | `/v2/orders/suscriptions/resources/query` | 查询客户包年/包月资源列表 |
| `ListProvinces` | GET | `/v2/systems/configs/provinces` | 查询省份信息 |
| `ListQuotaCoupons` | POST | `/v2/partners/coupon-quotas/query` | 查询优惠券额度 |
| `ListRateOnPeriodDetail` | POST | `/v2/bills/ratings/period-resources/subscribe-rate` | 查询包年/包月产品价格 **[pricing-core]** |
| `ListRenewRateOnPeriod` | POST | `/v2/bills/ratings/period-resources/renew-rate` | 查询待续订包年包月资源的续订金额 |
| `ListResourceSpecs` | POST | `/v2/products/resource-specs-query` | 查询云服务类型资源规格 **[pricing-core]** |
| `ListResourceTypes` | GET | `/v2/products/resource-types` | 查询资源类型列表 **[pricing-core]** |
| `ListResourceUsage` | GET | `/v2/bills/customer-bills/resources/usage/details` | 查询95计费资源用量明细 |
| `ListResourceUsageSummary` | GET | `/v2/bills/customer-bills/resources/usage/summary` | 查询95计费资源用量汇总 |
| `ListServiceResources` | GET | `/v2/products/service-resources` | 根据云服务类型查询资源类型列表 |
| `ListServiceTypes` | GET | `/v2/products/service-types` | 查询云服务类型列表 **[pricing-core]** |
| `ListStoredValueCards` | GET | `/v2/promotions/benefits/stored-value-cards` | 查询储值卡列表 |
| `ListSubCustomerBillDetail` | GET | `/v2/bills/subcustomer-bills/res-fee-records` | 查询伙伴子客户消费记录 |
| `ListSubCustomerCoupons` | GET | `/v2/promotions/benefits/coupons` | 查询优惠券列表 |
| `ListSubcustomerMonthlyBills` | GET | `/v2/bills/partner-bills/subcustomer-bills/monthly-sum` | 查询客户月度消费账单 |
| `ListSubCustomerNewTag` | POST | `/v2/partners/sub-customers/new-customers-tags/batch-query` | 查询客户新客标签 |
| `ListSubCustomers` | POST | `/v2/partners/sub-customers/query` | 查询客户列表 |
| `ListUsageTypes` | GET | `/v2/products/usage-types` | 查询使用量类型列表 |
| `PayOrders` | POST | `/v3/orders/customer-orders/pay` | 支付包年/包月产品订单 |
| `ReclaimCouponQuotas` | POST | `/v2/partners/coupon-quotas/indirect-partner-reclaim` | 回收云经销商的代金券额度 |
| `ReclaimEnterpriseMultiAccountCoupon` | POST | `/v2/enterprises/multi-accounts/retrieve-coupon` | 企业主账号从企业子账号回收优惠券 |
| `ReclaimIndirectPartnerAccount` | POST | `/v2/accounts/partner-accounts/indirect-partner-reclaim` | 回收云经销商账户拨款 |
| `ReclaimPartnerCoupons` | POST | `/v2/promotions/benefits/partner-coupons/reclaim` | 回收优惠券 |
| `ReclaimSubEnterpriseAmount` | POST | `/v2/enterprises/multi-accounts/retrieve-amount` | 企业主账号从企业子账号回收拨款 |
| `ReclaimToPartnerAccount` | POST | `/v2/accounts/partner-accounts/reclaim` | 回收客户账户余额 |
| `RenewalResources` | POST | `/v2/orders/subscriptions/resources/renew` | 续订包年/包月资源 |
| `SendSmsVerificationCode` | POST | `/v2/enterprises/multi-accounts/sm-verification-code` | 发送短信验证码 |
| `SendVerificationMessageCode` | POST | `/v2/bases/verificationcode/send` | 发送验证码 |
| `SetResourcesRenewConfig` | POST | `/v2/orders/subscriptions/resources/renew/config` | 设置包年/包月资源自动续费扣款日和续费后资源统一到期日 |
| `ShowCustomerAccountBalances` | GET | `/v2/accounts/customer-accounts/balances` | 查询账户余额 |
| `ShowCustomerMonthlySum` | GET | `/v2/bills/customer-bills/monthly-sum` | 查询汇总账单 **[pricing-core]** |
| `ShowCustomerOrderDetails` | GET | `/v2/orders/customer-orders/details/{order_id}` | 查询订单详情 |
| `ShowMultiAccountTransferAmount` | GET | `/v2/enterprises/multi-accounts/transfer-amount` | 查询企业主账号可拨款余额 |
| `ShowRealnameAuthenticationReviewResult` | GET | `/v2/customers/realname-auths/result` | 查询实名认证审核结果 |
| `ShowRealNameAuthQrCode` | GET | `/v2/customers/real-name-auth-qrcode` | 获取人脸实名认证二维码 |
| `ShowRealNameAuthStatus` | GET | `/v2/customers/real-name-auth-status` | 查询实名认证状态 |
| `ShowRefundOrderDetails` | GET | `/v2/orders/customer-orders/refund-orders` | 查询退款订单的金额详情 |
| `UpdateCouponQuotas` | POST | `/v2/partners/coupon-quotas/indirect-partner-adjust` | 向云经销商发放代金券额度 |
| `UpdateCustomerAccountAmount` | POST | `/v2/accounts/partner-accounts/adjust-amount` | 向客户账户拨款 |
| `UpdateIndirectPartnerAccount` | POST | `/v2/accounts/partner-accounts/indirect-partner-adjust` | 向云经销商账户拨款 |
| `UpdatePeriodToOnDemand` | POST | `/v2/orders/subscriptions/resources/to-on-demand` | 设置或取消包年/包月资源到期转按需 |
| `UpdatePeriodToOnDemandInstantly` | POST | `/v2/orders/subscriptions/resources/to-on-demand/instantly` | 设置包年/包月资源即时转按需 |
| `UpdateSubEnterpriseAmount` | POST | `/v2/enterprises/multi-accounts/transfer-amount` | 企业主账号向企业子账号拨款 |

## Reference files

Detailed API definitions (swagger with parameters, responses, examples) are in `references/<APIName>.json`. Use `read` to load the specific API you need. Use `grep` to search across all definitions for a field name or resource type.

## Query Methods

Use the `http_request` tool to call BSS APIs directly. Authentication (SDK-HMAC-SHA256 signing with your AK/SK) is handled automatically — do NOT pass any credentials.

### Example: query on-demand resource price

```
http_request(
  method: "POST",
  url: "https://bss.myhuaweicloud.com/v2/bills/ratings/on-demand-resources",
  headers: {"Content-Type": "application/json"},
  body: "{\"cloud_service_type\": \"hws.service.type.ec2\", \"resource_type\": \"hws.resource.type.vm\", \"region_code\": \"cn-north-1\", \"charge_mode\": 3, ...}"
)
```

The tool returns:
```json
{"status": 200, "headers": {...}, "body": "...API response JSON..."}
```

### Fallback

Use WebSearch and WebFetch to query HuaweiCloud official pricing pages when the BSS API is unavailable or you need to look up a service type code.

## Rules

- Mark prices that cannot be determined as `null`, do not fabricate
- Note the price source and query time
- Distinguish between pay-as-you-go and monthly billing

## Return Format

```json
{
  "items": [
    {"resource": "resource address", "spec": "spec description", "monthly": monthly price or null}
  ],
  "total_monthly": total monthly cost or null,
  "currency": "CNY",
  "note": "price notes"
}
```
