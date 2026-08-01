---
name: huaweicloud-ief
description: HuaweiCloud IEF API guide. 111 APIs covering 加密数据管理, 密钥管理, 应用模板管理, 批量作业管理, 批量节点管理.
---

# HuaweiCloud IEF API Guide

111 APIs. Tags: 加密数据管理, 密钥管理, 应用模板管理, 批量作业管理, 批量节点管理, 服务管理, 标签管理, 端点管理, 系统订阅管理, 终端设备模板管理, 终端设备管理, 规则管理, 边缘节点管理, 边缘节点组管理, 部署管理, 配置项管理, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchAddDeleteTags` | POST | `/v2/{project_id}/{resource_type}/{resource_id}/tags/action` | 批量添加删除资源标签 |
| `CreateApp` | POST | `/v2/{project_id}/edgemgr/apps` | 创建应用模板 |
| `CreateAppVersions` | POST | `/v2/{project_id}/edgemgr/apps/{app_id}/versions` | 创建应用模板版本 |
| `CreateBatchJob` | POST | `/v2/{project_id}/productmgr/jobs` | 创建批量处理任务 |
| `CreateConfigMap` | POST | `/v2/{project_id}/edgemgr/configmaps` | 创建配置项 |
| `CreateDeployments` | POST | `/v3/{project_id}/edgemgr/deployments` | 创建部署 |
| `CreateDevice` | POST | `/v2/{project_id}/edgemgr/devices` | 注册终端设备 |
| `CreateDeviceTemplate` | POST | `/v2/{project_id}/edgemgr/device-templates` | 创建终端设备模板 |
| `CreateEdgeGroup` | POST | `/v2/{project_id}/edgemgr/groups` | 边缘节点组管理 |
| `CreateEdgeGroupCert` | POST | `/v2/{project_id}/edgemgr/groups/{group_id}/certs` | 创建边缘节点组证书 |
| `CreateEdgeNode` | POST | `/v2/{project_id}/edgemgr/nodes` | 注册边缘节点 |
| `CreateEdgeNodeCerts` | POST | `/v2/{project_id}/edgemgr/nodes/{node_id}/certs` | 创建节点证书 |
| `CreateEncryptdatas` | POST | `/v2/{project_id}/edm/encryptdatas` | 新增加密数据 |
| `CreateEndpoint` | POST | `/v2/{project_id}/routemgr/endpoints` | 创建端点 |
| `CreateNodeEncryptdatas` | POST | `/v2/{project_id}/edm/nodes/{node_id}/encryptdatas` | 加密数据绑定边缘节点 |
| `CreateProduct` | POST | `/v2/{project_id}/productmgr/products` | 创建批量节点注册作业 |
| `CreateRule` | POST | `/v2/{project_id}/routemgr/rules` | 创建规则 |
| `CreateSecret` | POST | `/v2/{project_id}/edgemgr/secrets` | 创建密钥 |
| `CreateService` | POST | `/v2/{project_id}/edgemgr/services` | 创建服务 |
| `CreateSystemEvent` | POST | `/v2/{project_id}/routemgr/exchanger/systemevents` | 创建系统订阅 |
| `CreateTag` | POST | `/v2/{project_id}/{resource_type}/{resource_id}/tags` | 添加资源标签 |
| `DeleteApp` | DELETE | `/v2/{project_id}/edgemgr/apps/{app_id}` | 删除应用模板 |
| `DeleteAppVersion` | DELETE | `/v2/{project_id}/edgemgr/apps/{app_id}/versions/{version_id}` | 删除应用版本 |
| `DeleteBatchJob` | DELETE | `/v2/{project_id}/productmgr/jobs/{job_id}` | 删除批量处理作业 |
| `DeleteConfigMap` | DELETE | `/v2/{project_id}/edgemgr/configmaps/{configmap_id}` | 删除配置项 |
| `DeleteDeployment` | DELETE | `/v3/{project_id}/edgemgr/deployments/{deployment_id}` | 删除部署 |
| `DeleteDevice` | DELETE | `/v2/{project_id}/edgemgr/devices/{device_id}` | 删除终端设备 |
| `DeleteDeviceTemplate` | DELETE | `/v2/{project_id}/edgemgr/device-templates/{device_template_id}` | 删除终端设备模板 |
| `DeleteEdgeGroup` | DELETE | `/v2/{project_id}/edgemgr/groups/{group_id}` | 删除边缘节点组 |
| `DeleteEdgeGroupCert` | DELETE | `/v2/{project_id}/edgemgr/groups/{group_id}/certs/{group_cert_id}` | 删除边缘节点组证书 |

... and 81 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
