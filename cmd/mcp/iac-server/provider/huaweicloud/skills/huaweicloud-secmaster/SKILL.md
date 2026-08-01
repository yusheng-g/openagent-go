---
name: huaweicloud-secmaster
description: HuaweiCloud SecMaster API guide. 378 APIs covering SQL校验, 事件关系管理, 事件管理, 云日志资源, 云服务接入.
---

# HuaweiCloud SecMaster API Guide

378 APIs. Tags: SQL校验, 事件关系管理, 事件管理, 云日志资源, 云服务接入, 代码片段管理, 任务中心, 分析查询, 分析脚本管理, 分类映射, 剧本动作管理(待下线), 剧本实例管理, 剧本审核管理, 剧本打包管理, 剧本版本管理, 剧本管理, 剧本规则管理(待下线), 告警管理, 告警规则模板管理, 告警规则管理, 基线检查, 委托管理, 威胁情报管理, 字典管理, 安全分析, 安全报告, 工作空间管理, 应急策略, 指标定义, 指标管理, 插件管理, 插件配置模板管理, 操作连接管理, 数据加工作业管理, 数据对象管理, 数据投递, 数据消费管理, 数据空间管理, 数据管道, 数据类管理, 数据类类型, 数据统计, 查询分析, 查询条件, 检索脚本管理, 泛安全应用, 流程实例管理, 流程打包管理, 流程版本管理, 流程管理, 漏洞管理, 版本升级, 监控, 目录定制, 目录管理, 租户采集, 管道管理, 组件管理, 节点管理, 表管理, 计量计费管理, 订阅管理, 评论, 资产管理, 资源实例管理, 资源标签管理, 附件管理, 页面布局

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateDataobjectRelations` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/{dataclass_type}/{related_dataclass_type}/batch-create` | 批量关联数据对象 |
| `BatchCreateDatapanelObjects` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/datapanel/{dataclass}/data-objects/batch-create` | 批量创建数据对象 |
| `BatchSearchMetricHits` | POST | `/v1/{project_id}/workspaces/{workspace_id}/sa/metrics/hits` | 批量查询指标结果 |
| `BatchTagResources` | POST | `/v1/{project_id}/{resource_type}/{resource_id}/tags/create` | 批量添加资源标签 |
| `BatchUntagResources` | DELETE | `/v1/{project_id}/{resource_type}/{resource_id}/tags/delete` | 批量删除资源标签 |
| `BatchUpdateCatalogue` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/catalogues/batch-update` | 批量修改目录 |
| `ChangeAlert` | PUT | `/v1/{project_id}/workspaces/{workspace_id}/soc/alerts/{alert_id}` | 更新告警 |
| `ChangeAlerts` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/alerts/batch-update` | 批量更新告警 |
| `ChangeIncident` | PUT | `/v1/{project_id}/workspaces/{workspace_id}/soc/incidents/{incident_id}` | 更新事件 |
| `ChangeIncidents` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/incidents/batch-update` | 批量更新事件 |
| `ChangePlaybookInstance` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/playbooks/instances/{instance_id}/operation` | 操作剧本实例 |
| `ChangeResource` | PUT | `/v1/{project_id}/workspaces/{workspace_id}/sa/resources/{id}` | 更新资产信息 |
| `CopyMapping` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/mappings/{mapping_id}/clone` | 复制映射 |
| `CopyPlaybookVersion` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/playbooks/versions/{version_id}/clone` | 克隆剧本及版本 |
| `CountResourceInstance` | POST | `/v1/{project_id}/{resource_type}/resource-instances/count` | 查询资源实例数量 |
| `CreateAdhocQuery` | POST | `/v2/{project_id}/workspaces/{workspace_id}/siem/ad-hoc-queries` | 创建adhoc查询 |
| `CreateAlert` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/alerts` | 创建告警 |
| `CreateAlertRule` | POST | `/v1/{project_id}/workspaces/{workspace_id}/siem/alert-rules` | 创建告警规则 |
| `CreateAlertRuleSimulation` | POST | `/v1/{project_id}/workspaces/{workspace_id}/siem/alert-rules/simulation` | 模拟告警规则 |
| `CreateAnalysisScript` | POST | `/v2/{project_id}/workspaces/{workspace_id}/siem/analysis-scripts` | 创建分析脚本 |
| `CreateAopWorkflow` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/workflows` | 创建流程 |
| `CreateAopWorkflowVersion` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/workflows/{workflow_id}/versions` | 创建流程版本 |
| `CreateAopWorkflowVersionApprovel` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/workflows/versions/{version_id}/approval` | 审核流程版本的发布 |
| `CreateBatchOrderAlerts` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/alerts/batch-order` | 告警转事件 |
| `CreateCatalogue` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/catalogues` | 创建自定义目录 |
| `CreateClassifier` | POST | `/v1/{project_id}/workspaces/{workspace_id}/soc/mappings/classifiers` | 新增分类 |
| `CreateCodeSegment` | POST | `/v2/{project_id}/workspaces/{workspace_id}/siem/code-segments` | 创建代码片段 |
| `CreateCollectConfig` | POST | `/v2/{project_id}/collector/cloudlogs/config` | 保存云服务采集配置 |
| `CreateCollectorChannel` | POST | `/v1/{project_id}/workspaces/{workspace_id}/collector/channels` | 创建采集通道 |
| `CreateCollectorChannelGroup` | POST | `/v1/{project_id}/workspaces/{workspace_id}/collector/channels/groups` | 创建采集通道分组 |

... and 348 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
