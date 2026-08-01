---
name: huaweicloud-css
description: HuaweiCloud CSS API guide. 118 APIs covering Kibana公网访问接口, Logstash, Logstash接口, 公网访问接口, 参数配置接口.
---

# HuaweiCloud CSS API Guide

118 APIs. Tags: Kibana公网访问接口, Logstash, Logstash接口, 公网访问接口, 参数配置接口, 快照管理接口, 日志管理接口, 智能运维, 终端节点接口, 规格查询接口, 词库管理接口, 负载均衡, 集群管理接口

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddFavorite` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/lgsconf/favorite` | 添加到自定义模板 |
| `AddIndependentNode` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/type/{type}/independent` | 添加独立master、client |
| `ChangeClusterSubnet` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/subnet/change` | 切换集群子网 |
| `ChangeMode` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/mode/change` | 安全模式修改 |
| `ChangeSecurityGroup` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/sg/change` | 切换安全组 |
| `CloseAiOpsSetting` | PUT | `/v1.0/{project_id}/clusters/{cluster_id}/ai-ops/close` | 关闭智能运维定时检测 |
| `CreateAgency` | POST | `/v1.0/{project_id}/agency/create` | 自动创建委托 |
| `CreateAiOps` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/ai-ops` | 创建一次集群检测任务 |
| `CreateAutoCreatePolicy` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/index_snapshot/policy` | 设置自动创建快照策略 |
| `CreateBindPublic` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/public/open` | 开启公网访问 |
| `CreateCluster` | POST | `/v2.0/{project_id}/clusters` | 创建集群V2 |
| `CreateClustersTags` | POST | `/v1.0/{project_id}/{resource_type}/{cluster_id}/tags` | 添加指定集群标签 |
| `CreateCnf` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/lgsconf/submit` | 创建配置文件 |
| `CreateElbListener` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/es-listeners` | 集群负载均衡监听器配置。 |
| `CreateLoadIkThesaurus` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/thesaurus` | 加载自定义词库 |
| `CreateLogBackup` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/logs/collect` | 备份日志 |
| `CreateSnapshot` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/index_snapshot` | 手动创建快照 |
| `DeleteAiOps` | DELETE | `/v1.0/{project_id}/clusters/{cluster_id}/ai-ops/{aiops_id}` | 删除一个检测任务记录 |
| `DeleteCerts` | DELETE | `/v1.0/{project_id}/clusters/{cluster_id}/certs/{cert_id}/delete` | 删除证书文件 |
| `DeleteCluster` | DELETE | `/v1.0/{project_id}/clusters/{cluster_id}` | 删除集群 |
| `DeleteClustersTags` | DELETE | `/v1.0/{project_id}/{resource_type}/{cluster_id}/tags/{key}` | 删除集群标签 |
| `DeleteConf` | DELETE | `/v1.0/{project_id}/clusters/{cluster_id}/lgsconf/delete` | 删除配置文件 |
| `DeleteIkThesaurus` | DELETE | `/v1.0/{project_id}/clusters/{cluster_id}/thesaurus` | 删除自定义词库 |
| `DeleteLogstashConf` | POST | `/v2.0/{project_id}/clusters/{cluster_id}/lgsconf/delete` | 删除配置文件V2 |
| `DeleteLogstashTemplate` | POST | `/v2.0/{project_id}/lgsconf/deletetemplate` | 删除自定义模板V2 |
| `DeleteSnapshot` | DELETE | `/v1.0/{project_id}/clusters/{cluster_id}/index_snapshot/{snapshot_id}` | 删除快照 |
| `DeleteTemplate` | DELETE | `/v1.0/{project_id}/lgsconf/deletetemplate` | 删除自定义模板 |
| `DownloadCert` | GET | `/v1.0/{project_id}/cer/download` | 下载安全证书 |
| `EnableOrDisableElb` | POST | `/v1.0/{project_id}/clusters/{cluster_id}/loadbalancers/es-switch` | 为集群打开或关闭负载均衡器 |
| `ListActions` | GET | `/v1.0/{project_id}/clusters/{cluster_id}/lgsconf/listactions` | 查询操作记录 |

... and 88 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
