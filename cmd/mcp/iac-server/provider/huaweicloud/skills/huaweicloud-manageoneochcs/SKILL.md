---
name: huaweicloud-manageoneochcs
description: HuaweiCloud ManageOneOCHCS API guide. 200 APIs covering MODataStrategyEngineService, 告警管理, 安全管理, 容量管理, 性能监控.
---

# HuaweiCloud ManageOneOCHCS API Guide

200 APIs. Tags: MODataStrategyEngineService, 告警管理, 安全管理, 容量管理, 性能监控, 数据分析平台, 租户保障, 租户操作日志, 系统维护, 统一SSO, 统一备份, 统一巡检, 统一日志, 自动化运维, 证书管理, 资源管理, 驱动管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptDsmOperateLog` | PUT | `/rest/logaccessservice/v1/action/dsmoperatelog` | 接收DSM上报的数据 |
| `AddAccessSystem` | POST | `/rest/mounibackupservice/v1/access` | 备份注册 |
| `AddAttributeOld` | POST | `/rest/cmdb/v1/classes/{className}/attrs` | 新增资源类型属性,该接口不建议使用 |
| `AddBackupServer` | PUT | `/rest/mounibackupservice/v1/register/bkserver` | 配置备份服务器 |
| `AddServiceView` | POST | `/mo/api/moauth/v1/service-views` | 注册服务视图 |
| `AuditCIInstance` | PUT | `/rest/cmdb/v1/instances/audit/{className}` | 资源对账上报 |
| `BatchCreateCIInstance` | POST | `/rest/tenant-resource/v1/instances/batch/{class-name}` | 批量创建实例数据 |
| `BatchCreateCIRelationInstance` | POST | `/rest/cmdb/v1/instances/relations/{relationName}` | 批量创建资源关系实例 |
| `BatchDeleteCIInstance` | DELETE | `/rest/cmdb/v1/instances/{className}/{instanceId}` | 批量删除资源实例 |
| `BatchDeleteCIRelationInstance` | DELETE | `/rest/cmdb/v1/instances/relations/{relationName}/{instanceId}` | 删除资源关系实例 |
| `BatchDeleteDevice` | DELETE | `/rest/cmdb/v1/devices/{className}/{instanceId}` | 批量删除物理设备 |
| `BatchPatchCIInstance` | PATCH | `/rest/tenant-resource/v1/instances/batch/{class-name}` | 批量更新实例数据,不存在则新增 |
| `BatchUpdateCIRelationInstance` | PUT | `/rest/cmdb/v1/instances/relations/{relationName}` | 批量修改资源关系实例 |
| `CheckCloudServiceOperate` | GET | `/rest/cmdb/v3/cloud-services/{indexName}/action/check` | 校验CMDB云服务树的完整性 |
| `CheckCts` | GET | `/rest/trace/v1/ctscheck` | 操作日志检查接口 |
| `CountCIInstanceJob` | POST | `/rest/tenant-resource/v1/instances/count/vdc-dimension` | 查询vdc维度数据总数 |
| `CountCIInstances` | GET | `/rest/tenant-resource/v1/instances/{class-name}/statistics` | 查询指定CI类型的实例数量 |
| `CountCSMCIInstances` | GET | `/rest/cmdb/v1/instances/{className}/statistics` | 查询指定资源类型的实例数量 |
| `CountCSMCIRelationInstances` | GET | `/rest/cmdb/v1/instances/relations/{relationName}/statistics` | 查询符合条件的某类型关系的实例数量 |
| `CountInstances` | GET | `/rest/tenant-resource/v1/instances/count/{dimension-name}/{dimension-id}` | 查询资源实例数量信息 |
| `CreateAlarmRule` | POST | `/V1.0/{project_id}/alarms` | 创建一条告警规则 |
| `CreateAutoOpsJob` | POST | `/rest/moautoops/v1/jobs` | 创建作业 |
| `CreateCICategory` | PUT | `/rest/cmdb/v1/categories` | 创建资源分类 |
| `CreateCIInstance` | POST | `/rest/cmdb/v1/instances/{className}` | 创建资源实例 |
| `CreateCIRelation` | POST | `/rest/tenant-resource/v1/instances/relations/{relation-name}` | 创建CI关系数据 |
| `CreateCloudService` | POST | `/rest/cmdb/v2/cloud-services` | 创建云服务部署信息 |
| `CreateCloudServiceV3` | POST | `/rest/cmdb/v3/cloud-services` | 创建云服务部署信息V3版本 |
| `CreateDeployNode` | POST | `/rest/cmdb/v2/deploy-nodes` | 批量新增节点信息 |
| `CreateDevice` | POST | `/rest/cmdb/v1/devices/{className}` | 添加物理设备 |
| `CreateIdP` | PUT | `/mo/api/moauth/v1/idps/{name}` | 创建身份提供商 |

... and 170 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
