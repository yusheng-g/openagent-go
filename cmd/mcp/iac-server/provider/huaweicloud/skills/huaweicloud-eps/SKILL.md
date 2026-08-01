---
name: huaweicloud-eps
description: HuaweiCloud EPS API guide. 17 APIs covering 企业项目管理操作, 查询企业项目支持的服务, 查询版本操作.
---

# HuaweiCloud EPS API Guide

17 APIs. Tags: 企业项目管理操作, 查询企业项目支持的服务, 查询版本操作

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateEnterpriseProject` | POST | `/v1.0/enterprise-projects` | 创建企业项目 |
| `DeleteEnterpriseProject` | DELETE | `/v1.0/enterprise-projects/{enterprise_project_id}` | 删除企业项目 |
| `DisableEnterpriseProject` | POST | `/v1.0/enterprise-projects/{enterprise_project_id}/action` | 停用企业项目 |
| `EnableEnterpriseProject` | POST | `/v1.0/enterprise-projects/{enterprise_project_id}/action` | 启用企业项目 |
| `ListApiVersions` | GET | `/` | 查询API版本列表 |
| `ListEnterpriseProject` | GET | `/v1.0/enterprise-projects` | 查询企业项目列表 |
| `ListMigrationRecord` | GET | `/v1.0/enterprise-projects/migrate-record/list` | 查询资源迁移记录 |
| `ListProviders` | GET | `/v1.0/enterprise-projects/providers` | 查询企业项目支持的服务 |
| `ListResourceMapping` | GET | `/v1.0/enterprise-projects/resources-mapping` | 查询资源类型映射 |
| `MigrateResource` | POST | `/v1.0/enterprise-projects/{enterprise_project_id}/resources-migrate` | 迁移资源 |
| `ShowApiVersion` | GET | `/{api_version}` | 查询API版本号详情 |
| `ShowAssociatedResources` | GET | `/v1.0/associated-resources/{resource_id}` | 查询关联资源 |
| `ShowEnterpriseProject` | GET | `/v1.0/enterprise-projects/{enterprise_project_id}` | 查询企业项目详情 |
| `ShowEnterpriseProjectQuota` | GET | `/v1.0/enterprise-projects/quotas` | 查询企业项目配额 |
| `ShowEpConfigs` | GET | `/v1/enterprise-projects/configs` | 查询服务配置 |
| `ShowResourceBindEnterpriseProject` | POST | `/v1.0/enterprise-projects/{enterprise_project_id}/resources/filter` | 查询企业项目绑定的资源列表 |
| `UpdateEnterpriseProject` | PUT | `/v1.0/enterprise-projects/{enterprise_project_id}` | 修改企业项目 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
