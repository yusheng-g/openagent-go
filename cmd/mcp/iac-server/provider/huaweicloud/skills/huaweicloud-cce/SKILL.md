---
name: huaweicloud-cce
description: HuaweiCloud CCE API guide. 187 APIs covering API版本信息, Job管理(Autopilot), 套餐包管理, 存储管理, 插件管理. Detailed swagger definitions in references/<APIName>.json.
---

# HuaweiCloud CCE API Guide

187 APIs. Tags: API版本信息, Job管理(Autopilot), 套餐包管理, 存储管理, 插件管理, 插件管理(Autopilot), 权限管理, 标签管理, 标签管理(Autopilot), 模板管理, 模板管理(Autopilot), 节点池管理, 节点管理, 配置管理, 配额管理, 配额管理(Autopilot), 镜像缓存管理, 集群升级, 集群升级(Autopilot), 集群管理, 集群管理(Autopilot)

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddNode` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodes/add` | 纳管节点 |
| `AddNodesToNodePool` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodepools/{nodepool_id}/nodes/add` | 自定义节点池纳管节点 |
| `AssumeAgencyForPodIdentity` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/assume-agency-for-pod-identity` | 获取pod-identity关联相关委托凭据 |
| `AwakeCluster` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/operation/awake` | 集群唤醒 |
| `BatchChangeNodeToPeriod` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodes/toperiod` | 按需节点转包年/包月 |
| `BatchCreateAddonPrecheck` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/addons/precheck` | 批量创建插件检查任务 |
| `BatchCreateAutopilotClusterTags` | POST | `/autopilot/v3/projects/{project_id}/clusters/{cluster_id}/tags/create` | 批量添加指定集群的资源标签 |
| `BatchCreateClusterTags` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/tags/create` | 批量添加指定集群的资源标签 |
| `BatchDeleteAutopilotClusterTags` | POST | `/autopilot/v3/projects/{project_id}/clusters/{cluster_id}/tags/delete` | 批量删除指定集群的资源标签 |
| `BatchDeleteClusterTags` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/tags/delete` | 批量删除指定集群的资源标签 |
| `BatchSyncNodes` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodes/sync` | 批量同步节点 |
| `ContinueUpgradeClusterTask` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/operation/upgrade/continue` | 继续执行集群升级任务 |
| `CreateAccessPolicy` | POST | `/api/v3/access-policies` | 创建访问策略 |
| `CreateAddonInstance` | POST | `/api/v3/addons` | 创建AddonInstance |
| `CreateAutopilotAddonInstance` | POST | `/autopilot/v3/addons` | 创建AddonInstance |
| `CreateAutopilotCluster` | POST | `/autopilot/v3/projects/{project_id}/clusters` | 创建集群 |
| `CreateAutopilotClusterMasterSnapshot` | POST | `/autopilot/v3.1/projects/{project_id}/clusters/{cluster_id}/operation/snapshot` | 集群备份 |
| `CreateAutopilotKubernetesClusterCert` | POST | `/autopilot/v3/projects/{project_id}/clusters/{cluster_id}/clustercert` | 获取集群证书 |
| `CreateAutopilotMaintenanceWindow` | POST | `/autopilot/v3/projects/{project_id}/clusters/{cluster_id}/maintenancewindows` | 创建集群维护窗口 |
| `CreateAutopilotPostCheck` | POST | `/autopilot/v3/projects/{project_id}/clusters/{cluster_id}/operation/postcheck` | 集群升级后确认 |
| `CreateAutopilotPreCheck` | POST | `/autopilot/v3/projects/{project_id}/clusters/{cluster_id}/operation/precheck` | 集群升级前检查 |
| `CreateAutopilotRelease` | POST | `/autopilot/cam/v3/clusters/{cluster_id}/releases` | 创建模板实例 |
| `CreateAutopilotUpgradeWorkFlow` | POST | `/autopilot/v3/projects/{project_id}/clusters/{cluster_id}/operation/upgradeworkflows` | 开启集群升级流程引导任务 |
| `CreateCloudPersistentVolumeClaims` | POST | `/api/v1/namespaces/{namespace}/cloudpersistentvolumeclaims` | 创建PVC(待废弃) |
| `CreateCluster` | POST | `/api/v3/projects/{project_id}/clusters` | 创建集群 |
| `CreateClusterMasterSnapshot` | POST | `/api/v3.1/projects/{project_id}/clusters/{cluster_id}/operation/snapshot` | 集群备份 |
| `CreateImageCache` | POST | `/v5/imagecaches` | 创建镜像缓存 |
| `CreateKubernetesClusterCert` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/clustercert` | 获取集群证书 |
| `CreateNode` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodes` | 创建节点 |
| `CreateNodePool` | POST | `/api/v3/projects/{project_id}/clusters/{cluster_id}/nodepools` | 创建节点池 |

... and 157 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
