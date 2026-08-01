---
name: huaweicloud-cloudtable
description: HuaweiCloud CloudTable API guide. 10 APIs covering CloudTable集群管理接口, CloudTable集群管理接口v3.
---

# HuaweiCloud CloudTable API Guide

10 APIs. Tags: CloudTable集群管理接口, CloudTable集群管理接口v3

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateCloudTableCluster` | POST | `/v3/{project_id}/clusters` | 创建CloudTable集群 |
| `CreateCluster` | POST | `/v2/{project_id}/clusters` | 创建CloudTable集群 |
| `DeleteCluster` | DELETE | `/v2/{project_id}/clusters/{cluster_id}` | 删除CloudTable指定集群 |
| `EnableComponent` | POST | `/v2/{project_id}/clusters/{cluster_id}/components/{component_name}` | 开启opentsdb组件 |
| `ExpandClusterComponent` | POST | `/v2/{project_id}/clusters/{cluster_id}/nodes` | 扩容组件 |
| `ListClusters` | GET | `/v2/{project_id}/clusters` | 查询CloudTable集群列表 |
| `RebootCloudTableCluster` | POST | `/v2/{project_id}/clusters/{cluster_id}/restart` | 重启集群的api入口 |
| `ShowClusterDetail` | GET | `/v2/{project_id}/clusters/{cluster_id}` | 查询CloudTable集群详情 |
| `ShowClusterSetting` | GET | `/v2/{project_id}/clusters/{cluster_id}/setting` | 查询集群配置 |
| `UpdateClusterSetting` | PUT | `/v2/{project_id}/clusters/{cluster_id}/setting` | 修改集群配置 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
