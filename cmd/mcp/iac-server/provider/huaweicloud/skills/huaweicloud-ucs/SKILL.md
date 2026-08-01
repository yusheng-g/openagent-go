---
name: huaweicloud-ucs
description: HuaweiCloud UCS API guide. 75 APIs covering UCS集群, 容器舰队, 插件管理, 权限管理, 流量管理.
---

# HuaweiCloud UCS API Guide

75 APIs. Tags: UCS集群, 容器舰队, 插件管理, 权限管理, 流量管理, 策略管理, 配置管理, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateAddonInstance` | POST | `/v1/addons` | 安装插件实例 |
| `CreateClusterConf` | POST | `/v1/clusters/{clusterid}/clusterconf` | 获取集群安装时所需的配置信息 |
| `CreateClusterGroupPolicyInstance` | POST | `/v1/clustergroups/{clustergroupid}/policyinstance` | 创建舰队策略实例 |
| `CreateClusterKubeconfig` | POST | `/v1/clusters/{clusterid}/kubeconfig` | 获取集群kubeconfig |
| `CreateClusterPolicyInstance` | POST | `/v1/clusters/{clusterid}/policyinstance` | 创建集群策略实例 |
| `CreateConfigSet` | POST | `/v1/configsets` | 创建配置集合 |
| `CreateFederationCert` | POST | `/v1/clustergroups/{clustergroupid}/cert` | 创建联邦网络连接并下载联邦kubeconfig |
| `CreateFederationConnection` | POST | `/v1/clustergroups/{clustergroupid}/connection` | 创建联邦网络连接 |
| `CreateProxyUnitedWorkload` | POST | `/v1/clustergroups/{clustergroupid}/unitedworkload` | 创建联邦工作负载 |
| `CreateRecordSet` | POST | `/v1/traffic/recordsets` | 创建域名解析记录集 |
| `CreateRepo` | POST | `/v1/configsets/repos` | 创建仓库源 |
| `CreateRule` | POST | `/v1/permissions/rules` | 创建权限策略 |
| `DeleteAddonInstance` | DELETE | `/v1/addons/{id}` | 卸载插件实例 |
| `DeleteCluster` | DELETE | `/v1/clusters/{clusterid}` | 删除集群 |
| `DeleteClusterGroup` | DELETE | `/v1/clustergroups/{clustergroupid}` | 删除容器舰队 |
| `DeleteConfigSet` | DELETE | `/v1/configsets/{configsetid}` | 删除配置集合 |
| `DeletePolicyInstance` | DELETE | `/v1/policyinstances/{policyinstanceid}` | 删除指定策略实例 |
| `DeleteProxyUnitedWorkload` | DELETE | `/v1/clustergroups/{clustergroupid}/unitedworkload` | 删除联邦工作负载 |
| `DeleteRepo` | DELETE | `/v1/configsets/repos/{repoid}` | 删除仓库源 |
| `DeleteRule` | DELETE | `/v1/permissions/rules/{ruleid}` | 删除权限策略 |
| `DisableClusterGroupPolicy` | DELETE | `/v1/clustergroups/{clustergroupid}/policy` | 舰队关闭策略中心 |
| `DisableClusterPolicy` | DELETE | `/v1/clusters/{clusterid}/policy` | 集群关闭策略中心 |
| `DisableFederation` | DELETE | `/v1/clustergroups/{clustergroupid}/federations` | 关闭容器集群联邦 |
| `DisableGitOps` | DELETE | `/v1/clusters/{clusterid}/gitops` | 卸载集群gitops插件 |
| `DownloadFederationKubeconfig` | POST | `/v1/clustergroups/{clustergroupid}/kubeconfig` | 下载联邦kubeconfig |
| `EnableClusterGroupPolicy` | POST | `/v1/clustergroups/{clustergroupid}/policy` | 舰队启用策略中心 |
| `EnableClusterPolicy` | POST | `/v1/clusters/{clusterid}/policy` | 集群启用策略中心 |
| `EnableFederation` | POST | `/v1/clustergroups/{clustergroupid}/federations` | 启用容器舰队联邦 |
| `EnableGitOps` | POST | `/v1/clusters/{clusterid}/gitops` | 启用Gitops插件 |
| `JoinGroup` | POST | `/v1/clusters/{clusterid}/join` | 集群加入容器舰队 |

... and 45 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
