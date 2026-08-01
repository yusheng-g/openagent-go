---
name: huaweicloud-codeartsinspector
description: HuaweiCloud CodeArtsInspector API guide. 28 APIs covering 主机管理, 主机组管理, 任务管理, 报告管理, 网站任务管理.
---

# HuaweiCloud CodeArtsInspector API Guide

28 APIs. Tags: 主机管理, 主机组管理, 任务管理, 报告管理, 网站任务管理, 网站报告管理, 网站管理, 购买管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddGroup` | POST | `/{project_id}/hostscan/groups` | 批量创建主机组 |
| `AuthorizeDomains` | POST | `/{project_id}/webscan/domains/authenticate` | 认证网站资产 |
| `BatchCreateHosts` | POST | `/{project_id}/hostscan/hosts` | 批量创建主机资产 |
| `BatchStartHostTasks` | POST | `/{project_id}/hostscan/hosts/scan` | 批量启动或取消主机扫描任务 |
| `CancelTasks` | PUT | `/{project_id}/webscan/tasks` | 取消或重启网站扫描任务 |
| `CreateDomains` | POST | `/{project_id}/webscan/domains` | 创建网站资产 |
| `CreatePurchaseOrder` | POST | `/{project_id}/{service}/subscription/purchase` | 订购下单接口 |
| `CreateTasks` | POST | `/{project_id}/webscan/tasks` | 创建网站扫描任务并启动 |
| `DeleteDomains` | DELETE | `/{project_id}/webscan/domains` | 删除网站资产 |
| `DeleteGroup` | DELETE | `/{project_id}/hostscan/groups/{group_id}` | 删除主机组 |
| `DeleteHost` | DELETE | `/{project_id}/hostscan/hosts/delete/{host_id}` | 删除主机资产 |
| `DownloadTaskReport` | GET | `/{project_id}/webscan/report` | 下载网站扫描报告 |
| `ExecuteGenerateReport` | POST | `/{project_id}/webscan/report` | 生成网站扫描报告 |
| `ListBusinessRisks` | GET | `/{project_id}/webscan/results/business-risk` | 获取网站业务风险扫描结果 |
| `ListDomains` | GET | `/{project_id}/webscan/domains` | 获取网站资产 |
| `ListGroups` | GET | `/{project_id}/hostscan/groups` | 获取主机组列表 |
| `ListHostResults` | GET | `/{project_id}/hostscan/hosts/{host_id}/sys-vulns` | 获取主机漏洞扫描结果 |
| `ListHosts` | GET | `/{project_id}/hostscan/hosts` | 获取主机资产 |
| `ListPortResults` | GET | `/{project_id}/webscan/results/ports` | 获取网站端口扫描结果 |
| `ListTaskHistories` | GET | `/{project_id}/webscan/tasks/histories` | 获取网站的历史扫描任务 |
| `ShowDomainSettings` | GET | `/{project_id}/webscan/domains/settings` | 获取网站配置 |
| `ShowReportStatus` | GET | `/{project_id}/webscan/report/status` | 获取网站扫描报告状态 |
| `ShowResults` | GET | `/{project_id}/webscan/results` | 获取网站扫描结果 |
| `ShowSubscription` | GET | `/{project_id}/{service}/subscription` | 资源版本查询接口 |
| `ShowTasks` | GET | `/{project_id}/webscan/tasks` | 获取网站扫描任务详情 |
| `UpdateDomainSettings` | POST | `/{project_id}/webscan/domains/settings` | 更新网站配置 |
| `UpdateFalsePositive` | POST | `/{project_id}/webscan/vulnerability/false-positive` | 更新网站漏洞的误报状态 |
| `UpdatePurchaseOrder` | POST | `/{project_id}/{service}/subscription/alter` | 变更下单接口 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
