---
name: huaweicloud-iamaccessanalyzer
description: HuaweiCloud IAMAccessAnalyzer API guide. 31 APIs covering 分析器, 分析结果, 存档规则, 标签, 消息通知配置.
---

# HuaweiCloud IAMAccessAnalyzer API Guide

31 APIs. Tags: 分析器, 分析结果, 存档规则, 标签, 消息通知配置, 策略校验, 访问预览, 资源分析配置

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ApplyArchiveRule` | POST | `/v5/analyzers/{analyzer_id}/archive-rules/{archive_rule_id}/apply` | 应用存档规则 |
| `CheckNoNewAccess` | POST | `/v5/policies/check-no-new-access` | 校验策略是否有新访问权限 |
| `CreateAccessPreview` | POST | `/v5/analyzers/{analyzer_id}/access-previews` | 创建访问预览 |
| `CreateAnalyzer` | POST | `/v5/analyzers` | 创建分析器 |
| `CreateArchiveRule` | POST | `/v5/analyzers/{analyzer_id}/archive-rules` | 为指定的分析器创建存档规则 |
| `CreateNotificationSetting` | POST | `/v5/notification-settings` | 创建消息通知配置 |
| `CreateResourceConfigurations` | POST | `/v5/analyzers/{analyzer_id}/resource-configurations/create` | 创建资源分析配置 |
| `DeleteAnalyzer` | DELETE | `/v5/analyzers/{analyzer_id}` | 删除指定的分析器 |
| `DeleteArchiveRule` | DELETE | `/v5/analyzers/{analyzer_id}/archive-rules/{archive_rule_id}` | 删除指定的存档规则 |
| `DeleteNotificationSetting` | DELETE | `/v5/notification-settings/{notification_setting_id}` | 删除消息通知配置 |
| `DeleteResourceConfigurations` | POST | `/v5/analyzers/{analyzer_id}/resource-configurations/delete` | 删除资源分析配置 |
| `ListAccessPreviewFindings` | POST | `/v5/analyzers/{analyzer_id}/access-previews/{access_preview_id}/findings` | 获取相关预览生成的分析结果 |
| `ListAccessPreviews` | GET | `/v5/analyzers/{analyzer_id}/access-previews` | 获取所有访问预览 |
| `ListAnalyzers` | GET | `/v5/analyzers` | 检索分析器的列表 |
| `ListArchiveRules` | GET | `/v5/analyzers/{analyzer_id}/archive-rules` | 检索为指定分析器创建的存档规则的列表 |
| `ListFindings` | POST | `/v5/analyzers/{analyzer_id}/findings` | 检索指定分析器生成的访问分析结果列表 |
| `ListNotificationSettings` | GET | `/v5/notification-settings` | 获取消息通知配置列表 |
| `ListResourceConfigurations` | GET | `/v5/analyzers/{analyzer_id}/resource-configurations` | 列举资源分析配置 |
| `ShowAccessPreview` | GET | `/v5/analyzers/{analyzer_id}/access-previews/{access_preview_id}` | 获取相关访问预览的信息 |
| `ShowAnalyzer` | GET | `/v5/analyzers/{analyzer_id}` | 显示指定的分析器 |
| `ShowArchiveRule` | GET | `/v5/analyzers/{analyzer_id}/archive-rules/{archive_rule_id}` | 检索有关存档规则的信息 |
| `ShowFinding` | GET | `/v5/analyzers/{analyzer_id}/findings/{finding_id}` | 检索有关指定结果的信息 |
| `ShowNotificationSetting` | GET | `/v5/notification-settings/{notification_setting_id}` | 获取消息通知配置 |
| `StartResourceScan` | POST | `/v5/analyzers/{analyzer_id}/scan` | 立即开始扫描应用于指定资源的策略 |
| `TagResource` | POST | `/v5/{resource_type}/{resource_id}/tags/create` | 向指定资源添加标签 |
| `UntagResource` | POST | `/v5/{resource_type}/{resource_id}/tags/delete` | 从指定资源中删除标签 |
| `UpdateAnalyzer` | PUT | `/v5/analyzers/{analyzer_id}` | 更新指定分析器的配置 |
| `UpdateArchiveRule` | PUT | `/v5/analyzers/{analyzer_id}/archive-rules/{archive_rule_id}` | 更新指定存档规则的条件和值 |
| `UpdateFindings` | PUT | `/v5/analyzers/{analyzer_id}/findings` | 更新指定结果的状态 |
| `UpdateNotificationSetting` | PUT | `/v5/notification-settings/{notification_setting_id}` | 更新消息通知配置 |

... and 1 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
