---
name: huaweicloud-iotedge
description: HuaweiCloud IoTEdge API guide. 107 APIs covering 北向HTTP请求代理, 北向NA管理, 南向IA配置项管理, 外部实体管理, 应用实例管理.
---

# HuaweiCloud IoTEdge API Guide

107 APIs. Tags: 北向HTTP请求代理, 北向NA管理, 南向IA配置项管理, 外部实体管理, 应用实例管理, 应用版本管理, 应用管理, 数据流转配置管理, 模块影子管理, 模块管理, 点位表模板管理, 节点管理, 设备控制, 设备管理, 调度计划管理, 边缘应用模板版本管理, 边缘应用模板管理, 边缘应用配置模板管理, 边缘数据源点位配置管理, 边缘数据源配置管理, 边缘数采配置模板, 边缘集群管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddAppConfigsTemplates` | POST | `/{project_id}/templates/apps/configs` | 添加应用配置模板 |
| `AddDevice` | POST | `/{project_id}/edge-nodes/{edge_node_id}/devices` | 添加设备 |
| `AddGeneralAppConfigsTemplate` | POST | `/{project_id}/templates/apps/configs/import` | 导入标准应用配置模板 |
| `AddGeneralOtTemplate` | POST | `/{project_id}/templates/ots/data-sources/import` | 导入标准数采模板 |
| `AddOtTemplates` | POST | `/{project_id}/templates/ots/data-sources` | 添加数采模板 |
| `BatchAssociateNaToNodes` | POST | `/{project_id}/nas/{na_id}/nodes` | 授权北向NA信息到边缘节点 |
| `BatchConfirmConfigsNew` | POST | `/{project_id}/edge-nodes/{node_id}/ias/{ia_id}/configs/batch-confirm` | 批量确认南向3rdIA配置项 |
| `BatchImportConfigs` | POST | `/{project_id}/edge-nodes/{node_id}/ias/{ia_id}/configs/batch-import` | 批量导入南向3rdIA配置项 |
| `BatchListAppConfigsTemplates` | GET | `/{project_id}/templates/apps/configs` | 查询应用配置模板列表 |
| `BatchListDcDevices` | GET | `/{project_id}/edge-nodes/{edge_node_id}/ots/data-sources/{ds_id}/devices` | 查数采连接子设备列表 |
| `BatchListDcDs` | GET | `/{project_id}/edge-nodes/{edge_node_id}/ots/data-sources` | 查询数据源配置列表 |
| `BatchListDcPoints` | GET | `/{project_id}/edge-nodes/{edge_node_id}/ots/data-sources/{ds_id}/points` | 查询点位表配置列表 |
| `BatchListEdgeApps` | GET | `/{project_id}/edge-apps` | 查询应用列表 |
| `BatchListEdgeAppVersions` | GET | `/{project_id}/edge-apps/{edge_app_id}/versions` | 查询应用版本列表 |
| `BatchListModules` | GET | `/{project_id}/edge-nodes/{edge_node_id}/modules` | 查询边缘模块列表 |
| `BatchListOtTemplates` | GET | `/{project_id}/templates/ots/data-sources` | 查询数采模板列表 |
| `CreateApp` | POST | `/{project_id}/apps` | 创建应用模板 |
| `CreateAppInstance` | POST | `/{project_id}/clusters/{cluster_id}/app-instances` | 创建应用实例 |
| `CreateAppVersion` | POST | `/{project_id}/apps/{app_id}/versions` | 创建应用版本 |
| `CreateCluster` | POST | `/{project_id}/clusters` | 创建边缘集群 |
| `CreateClusterInstallCmd` | POST | `/{project_id}/clusters/{cluster_id}/install-cmd` | 生成边缘集群安装命令 |
| `CreateDcPoint` | POST | `/{project_id}/edge-nodes/{edge_node_id}/ots/data-sources/{ds_id}/points` | 创建点位表配置 |
| `CreateDs` | POST | `/{project_id}/edge-nodes/{edge_node_id}/ots/data-sources` | 创建数据源配置 |
| `CreateEdgeApp` | POST | `/{project_id}/edge-apps` | 创建应用 |
| `CreateEdgeApplicationVersion` | POST | `/{project_id}/edge-apps/{edge_app_id}/versions` | 创建应用版本 |
| `CreateEdgeNode` | POST | `/{project_id}/edge-nodes` | 创建边缘节点 |
| `CreateExternalEntity` | POST | `/{project_id}/edge-nodes/{edge_node_id}/externals` | 在指定节点上创建外部实体 |
| `CreateInstallCmd` | POST | `/{project_id}/edge-nodes/{edge_node_id}/install` | 生成边缘节点安装命令 |
| `CreateModule` | POST | `/{project_id}/edge-nodes/{edge_node_id}/modules` | 创建边缘模块 |
| `CreateSchedule` | POST | `/{project_id}/edge-nodes/{edge_node_id}/schedules` | 创建调度计划 |

... and 77 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
