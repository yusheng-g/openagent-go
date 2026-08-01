---
name: huaweicloud-cdm
description: HuaweiCloud CDM API guide. 27 APIs covering 作业管理, 连接管理, 集群管理.
---

# HuaweiCloud CDM API Guide

27 APIs. Tags: 作业管理, 连接管理, 集群管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateAndStartRandomClusterJob` | POST | `/v1.1/{project_id}/clusters/job` | 随机集群创建作业并执行 |
| `CreateCluster` | POST | `/v1.1/{project_id}/clusters` | 创建集群 |
| `CreateJob` | POST | `/v1.1/{project_id}/clusters/{cluster_id}/cdm/job` | 指定集群创建作业 |
| `CreateLink` | POST | `/v1.1/{project_id}/clusters/{cluster_id}/cdm/link` | 创建连接 |
| `DeleteCluster` | DELETE | `/v1.1/{project_id}/clusters/{cluster_id}` | 删除集群 |
| `DeleteJob` | DELETE | `/v1.1/{project_id}/clusters/{cluster_id}/cdm/job/{job_name}` | 删除作业 |
| `DeleteLink` | DELETE | `/v1.1/{project_id}/clusters/{cluster_id}/cdm/link/{link_name}` | 删除连接 |
| `ListClusters` | GET | `/v1.1/{project_id}/clusters` | 查询集群列表 |
| `ModifyCluster` | POST | `/v1.1/{project_id}/cluster/modify/{cluster_id}` | 修改集群 |
| `RestartCluster` | POST | `/v1.1/{project_id}/clusters/{cluster_id}/action` | 重启集群 |
| `ShowAvailabilityZones` | GET | `/v1.1/{project_id}/regions/{region_id}/availability_zones` | 查询所有可用区 |
| `ShowClusterDetail` | GET | `/v1.1/{project_id}/clusters/{cluster_id}` | 查询集群详情 |
| `ShowClusterEnterpriseProjects` | GET | `/v1.1/{project_id}/clusters/{cluster_id}/enterprise-projects` | 查询集群的企业项目ID |
| `ShowDatastores` | GET | `/v1.1/{project_id}/datastores` | 查询支持的版本 |
| `ShowEnterpriseProjects` | GET | `/v1.1/{project_id}/enterprise-projects` | 查询所有集群的企业项目ID |
| `ShowFlavorDetail` | GET | `/v1.1/{project_id}/flavors/{flavor_id}` | 查询规格详情 |
| `ShowFlavors` | GET | `/v1.1/{project_id}/datastores/{datastore_id}/flavors` | 查询版本规格 |
| `ShowInstanceDetail` | GET | `/v1.1/{project_id}/instances/{instance_id}` | 查询集群实例信息 |
| `ShowJobs` | GET | `/v1.1/{project_id}/clusters/{cluster_id}/cdm/job/{job_name}` | 查询作业 |
| `ShowJobStatus` | GET | `/v1.1/{project_id}/clusters/{cluster_id}/cdm/job/{job_name}/status` | 查询作业状态 |
| `ShowLink` | GET | `/v1.1/{project_id}/clusters/{cluster_id}/cdm/link/{link_name}` | 查询连接 |
| `ShowSubmissions` | GET | `/v1.1/{project_id}/clusters/{cluster_id}/cdm/submissions` | 查询作业执行历史 |
| `StartCluster` | POST | `/v1.1/{project_id}/clusters/{cluster_id}/action` | 启动集群 |
| `StartJob` | PUT | `/v1.1/{project_id}/clusters/{cluster_id}/cdm/job/{job_name}/start` | 启动作业 |
| `StopJob` | PUT | `/v1.1/{project_id}/clusters/{cluster_id}/cdm/job/{job_name}/stop` | 停止作业 |
| `UpdateJob` | PUT | `/v1.1/{project_id}/clusters/{cluster_id}/cdm/job/{job_name}` | 修改作业 |
| `UpdateLink` | PUT | `/v1.1/{project_id}/clusters/{cluster_id}/cdm/link/{link_name}` | 修改连接 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
