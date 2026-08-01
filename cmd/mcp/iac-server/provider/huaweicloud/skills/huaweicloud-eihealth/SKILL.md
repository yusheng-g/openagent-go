---
name: huaweicloud-eihealth
description: HuaweiCloud eiHealth API guide. 223 APIs covering ADMET, CPI作业管理, CSS集群管理, IChatController, IDrugCommonController.
---

# HuaweiCloud eiHealth API Guide

223 APIs. Tags: ADMET, CPI作业管理, CSS集群管理, IChatController, IDrugCommonController, IObsController, IOverviewController, OBS桶管理, notebook开发环境, 业务委托管理, 作业管理, 供应商管理, 分子优化作业管理, 分子对接作业管理, 分子属性预测作业管理, 分子搜索作业管理, 分子生成作业管理, 初始化平台, 应用管理, 性能加速资源管理, 收藏管理, 数据作业管理, 数据管理, 标签管理, 模型供应商管理, 流程管理, 流程统计管理, 用户管理, 盘古药物分子大模型计费管理, 科研助手对话管理, 科研助手模型管理, 空间管理, 空间统计接口, 系统配置内部接口, 系统配额及资源使用情况获取, 聚类分析作业管理, 自由能微扰作业管理, 节点标签管理, 药物作业管理, 药物数据库管理, 药物模型管理, 药物通用接口, 计算集群管理, 资产管理, 资产计费管理, 镜像管理, 靶点优化作业管理, 靶点口袋分子设计作业管理, 靶点口袋发现作业管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddDrugDatabaseFile` | PUT | `/v1/{project_id}/drug/databases/{database_id}/data` | 数据库追加文件 |
| `BatchCancelJob` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/jobs/batch-terminate` | 批量取消作业 |
| `BatchDeleteData` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/datas/batch-delete` | 批量删除项目数据 |
| `BatchDeleteJob` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/jobs/batch-delete` | 批量删除作业 |
| `BatchDeleteLabel` | POST | `/v1/{project_id}/system/labels/batch-delete` | 批量删除标签 |
| `BatchImportApp` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/apps/batch-import` | 导入应用 |
| `BatchRetryJob` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/jobs/batch-retry` | 批量重试作业 |
| `CancelDataJob` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/data-jobs/{data_job_id}/cancel` | 取消数据作业 |
| `CancelDrugJob` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/drug-jobs/{job_id}/cancel` | 取消药物作业 |
| `CancelJob` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/jobs/{job_id}/terminate` | 取消或强制停止作业调度 |
| `ChangePassword` | POST | `/v1/{project_id}/users/{user_id}/password` | 修改密码 |
| `CheckDrugLigandDifference` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/drug-common/ligand/diff3d` | 计算配体间的3D结构差异 |
| `CheckTokenVerification` | GET | `/v1/{project_id}/users/token-verification` | 校验token |
| `CopyData` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/datas/clone` | 复制项目数据 |
| `CreateAdmetJob` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/drug-jobs/admet` | 创建分子属性预测作业 |
| `CreateApp` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/apps` | 创建应用 |
| `CreateAssetResource` | POST | `/v1/{project_id}/assets/asset-resources` | 创建计费资产资源 |
| `CreateAssistantModel` | POST | `/v1/{project_id}/model-vendors/{vendor_id}/models` | 创建助手模型 |
| `CreateClusteringJob` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/drug-jobs/clustering` | 创建聚类分析作业 |
| `CreateClusterJob` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/drug-jobs/{job_id}/cluster` | 创建分子聚类作业 |
| `CreateCode` | POST | `/v1/{project_id}/users/{user_id}/verification-code` | 发送验证码 |
| `CreateComputingCluster` | POST | `/v1/{project_id}/system/computing-clusters` | 绑定计算集群 |
| `CreateCpiJob` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/drug-jobs/cpi` | 创建CPI作业 |
| `CreateCssCluster` | POST | `/v1/{project_id}/drug/css-clusters` | 绑定CSS集群 |
| `CreateData` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/datas` | 创建文件夹 |
| `CreateDockingJob` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/drug-jobs/docking` | 创建分子对接作业 |
| `CreateDrugDatabase` | POST | `/v1/{project_id}/drug/databases` | 创建数据库 |
| `CreateDrugLigandInteraction2dSvg` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/drug-common/ligand/interaction2d` | 生成相互作用2D图 |
| `CreateDrugLigandPreviewTask` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/drug-common/ligand/preview` | 创建配体文件预览任务 |
| `CreateDrugLigandSdf` | POST | `/v1/{project_id}/eihealth-projects/{eihealth_project_id}/drug-common/ligand/sdf` | 生成分子SDF三维结构 |

... and 193 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
