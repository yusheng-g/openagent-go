---
name: huaweicloud-cdn
description: HuaweiCloud CDN API guide. 90 APIs covering 刷新预热, 域名操作, 域名配置, 日志管理, 模板配置.
---

# HuaweiCloud CDN API Guide

90 APIs. Tags: 刷新预热, 域名操作, 域名配置, 日志管理, 模板配置, 租户配置, 统计分析, 计费管理, 配额中心

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ApplyDomainTemplate` | POST | `/v1.0/cdn/configuration/templates/{tml_id}/apply` | 应用域名模板。 |
| `BatchCopyDomain` | POST | `/v1.0/cdn/configuration/domains/batch-copy` | 批量域名复制 |
| `BatchDeleteTags` | POST | `/v1.0/cdn/configuration/tags/batch-delete` | 删除资源标签配置接口 |
| `BatchUpdateRuleStatus` | POST | `/v1.0/cdn/configuration/domains/{domain_name}/rules/batch-update` | 批量更新规则状态及优先级 |
| `CreateAccessControlTask` | POST | `/v1.0/cdn/content/access-control-urls/{action}` | 创建封禁/解禁URL任务 |
| `CreateDomain` | POST | `/v1.0/cdn/domains` | 创建加速域名 |
| `CreateDomainByDuplicate` | POST | `/v1.0/cdn/configuration/domains/duplicate` | 复制配置到新添加域名 |
| `CreateDomainTemplate` | POST | `/v1.0/cdn/configuration/templates` | 创建域名模板。 |
| `CreateExportTask` | POST | `/v1/cdn/statistics/export-tasks` | 创建统计数据异步导出任务 |
| `CreatePreheatingTasks` | POST | `/v1.0/cdn/content/preheating-tasks` | 创建预热缓存任务 |
| `CreateRefreshTasks` | POST | `/v1.0/cdn/content/refresh-tasks` | 创建刷新缓存任务 |
| `CreateRuleNew` | POST | `/v1.0/cdn/configuration/domains/{domain_name}/rules` | 创建规则引擎规则 |
| `CreateShareCacheGroups` | POST | `/v1.0/cdn/configuration/share-cache-groups` | 创建共享缓存组 |
| `CreateSubscriptionTask` | POST | `/v1/cdn/statistics/subscription-tasks` | 创建运营报表订阅任务 |
| `CreateTags` | POST | `/v1.0/cdn/configuration/tags` | 创建资源标签配置接口 |
| `DeleteDomain` | DELETE | `/v1.0/cdn/domains/{domain_id}` | 删除加速域名 |
| `DeleteDomainTemplate` | DELETE | `/v1.0/cdn/configuration/templates/{tml_id}` | 删除域名模板。 |
| `DeleteRuleNew` | DELETE | `/v1.0/cdn/configuration/domains/{domain_name}/rules/{rule_id}` | 删除规则引擎规则 |
| `DeleteShareCacheGroups` | DELETE | `/v1.0/cdn/configuration/share-cache-groups/{id}` | 删除共享缓存组 |
| `DeleteSubscriptionTask` | DELETE | `/v1/cdn/statistics/subscription-tasks/{id}` | 删除运营报表订阅任务 |
| `DisableDomain` | PUT | `/v1.0/cdn/domains/{domain_id}/disable` | 停用加速域名 |
| `DownloadRegionCarrierExcel` | GET | `/v1.0/cdn/statistics/region-carrier-excel` | 下载区域运营商指标数据表格文件 |
| `DownloadStatisticsExcel` | GET | `/v1.0/cdn/statistics/statistics-excel` | 下载统计指标数据表格文件 |
| `EnableDomain` | PUT | `/v1.0/cdn/domains/{domain_id}/enable` | 启用加速域名 |
| `ExportStatsOpen` | POST | `/v1.0/cdn/statistics/stats/export` | CDN数据导出 |
| `ListAccessControlTask` | GET | `/v1.0/cdn/content/access-control-tasks` | 查询封禁/解禁URL任务 |
| `ListBanUrl` | GET | `/v1.0/cdn/content/ban-urls` | 查询已封禁的URL |
| `ListCdnDomainTopIps` | GET | `/v1.0/cdn/statistics/top-ips` | 查询域名top ip统计分析数据 |
| `ListCdnDomainTopOriginUrl` | GET | `/v1.0/cdn/statistics/top-origin-urls` | 查询域名top回源URL数据 |
| `ListCdnDomainTopPath` | GET | `/v1.0/cdn/statistics/top-path` | 查询TOP100 Path访问明细 |

... and 60 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
