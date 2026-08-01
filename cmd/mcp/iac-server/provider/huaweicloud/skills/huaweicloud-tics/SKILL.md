---
name: huaweicloud-tics
description: HuaweiCloud TICS API guide. 19 APIs covering 作业实例管理, 可信节点管理, 审计日志管理, 数据集管理, 统计信息管理.
---

# HuaweiCloud TICS API Guide

19 APIs. Tags: 作业实例管理, 可信节点管理, 审计日志管理, 数据集管理, 统计信息管理, 联盟管理, 联邦分析作业管理, 联邦学习作业管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ListAgents` | GET | `/v1/{project_id}/agents` | 获取计算节点列表 |
| `ListAuditInfo` | GET | `/v1/{project_id}/leagues/{league_id}/audit-info` | 查询审计日志 |
| `ListFlJob` | GET | `/v1/{project_id}/leagues/{league_id}/fl-jobs` | 查询联邦学习作业列表 |
| `ListInstanceHistory` | GET | `/v1/{project_id}/leagues/{league_id}/job-instances` | 查询作业的历史实例列表 |
| `ListLeagueDatasets` | GET | `/v1/{project_id}/leagues/{league_id}/datasets` | 查询联盟已注册数据集列表 |
| `ListLeagues` | GET | `/v1/{project_id}/league-info` | 获取联盟列表 |
| `ListNodes` | GET | `/v1/{project_id}/leagues/{league_id}/nodes` | 查询联盟节点列表 |
| `ListNotices` | GET | `/v1/{project_id}/notices` | 查询通知管理列表 |
| `ListPartners` | GET | `/v1/{project_id}/leagues/{league_id}/partners` | 获取联盟组员信息 |
| `ListSqlJob` | GET | `/v1/{project_id}/leagues/{league_id}/sql-jobs` | 查询联邦分析作业列表 |
| `ShowAgentDetail` | GET | `/v1/{project_id}/agents/{agent_id}` | 获取计算节点详情信息 |
| `ShowDatasetStatistics` | GET | `/v1/{project_id}/leagues/{league_id}/datasets-statistics` | 数据集统计 |
| `ShowInstanceReport` | GET | `/v1/{project_id}/leagues/{league_id}/job-instances/{instance_id}/report` | 查询实例执行报告 |
| `ShowJobInstanceDag` | GET | `/v1/{project_id}/leagues/{league_id}/job-instances/{instance_id}/dag` | 获取实例执行图 |
| `ShowJobStatistics` | GET | `/v1/{project_id}/leagues/{league_id}/jobs-statistics` | 作业统计 |
| `ShowLeague` | GET | `/v1/{project_id}/leagues/{league_id}` | 获取联盟详细信息 |
| `ShowOverview` | GET | `/v1/{project_id}/overview/statistics` | 查询租户下统计信息 |
| `ShowPartnerStatistics` | GET | `/v1/{project_id}/leagues/{league_id}/partners-statistics` | 合作方统计 |
| `UpdateLeague` | PUT | `/v1/{project_id}/leagues/{league_id}` | 更新联盟信息 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
