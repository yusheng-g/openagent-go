---
name: huaweicloud-drs
description: HuaweiCloud DRS API guide. 169 APIs covering 企业项目管理, 公共接口管理, 参数管理, 备份迁移, 实例操作.
---

# HuaweiCloud DRS API Guide

169 APIs. Tags: 企业项目管理, 公共接口管理, 参数管理, 备份迁移, 实例操作, 实例数据库对象配置, 实例管理, 实例详情, 实时同步管理, 实时灾备管理, 实时迁移管理, 对比管理, 录制回放, 批量异步实例管理, 数据加工, 权限管理, 标签管理, 资源管理, 连接管理, 配置管理, 配额

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchChangeData` | POST | `/v3/{project_id}/jobs/batch-transformation` | 批量数据加工 |
| `BatchCheckJobs` | POST | `/v3/{project_id}/jobs/batch-precheck` | 批量预检查 |
| `BatchCheckResults` | POST | `/v3/{project_id}/jobs/batch-precheck-result` | 批量查询预检查结果 |
| `BatchCreateJobs` | POST | `/v3/{project_id}/jobs/batch-creation` | 批量创建任务 |
| `BatchCreateJobsAsync` | POST | `/v5/{project_id}/jobs/batch-async-create` | 批量异步创建任务 |
| `BatchCreateTags` | POST | `/v5/{project_id}/{resource_type}/{resource_id}/tags/create` | 批量添加资源标签 |
| `BatchDeleteJobs` | DELETE | `/v3/{project_id}/jobs/batch-jobs` | 批量结束任务或删除任务 |
| `BatchDeleteJobsById` | DELETE | `/v5/{project_id}/jobs` | 批量删除任务 |
| `BatchDeleteTags` | POST | `/v5/{project_id}/{resource_type}/{resource_id}/tags/delete` | 批量删除资源标签 |
| `BatchExecuteJobActions` | POST | `/v5/{project_id}/jobs/action` | 批量操作指定ID任务 |
| `BatchListJobDetails` | POST | `/v3/{project_id}/jobs/batch-detail` | 批量查询任务详情 |
| `BatchListJobStatus` | POST | `/v3/{project_id}/jobs/batch-status` | 批量查询任务状态 |
| `BatchListProgresses` | POST | `/v3/{project_id}/jobs/batch-progress` | 批量查询任务进度 |
| `BatchListRposAndRtos` | POST | `/v3/{project_id}/jobs/batch-rpo-and-rto` | 批量查询RPO和RTO |
| `BatchListStructDetail` | POST | `/v3/{project_id}/jobs/{type}/batch-struct-detail` | 批量查询灾备初始化对象详情 |
| `BatchListStructProcess` | POST | `/v3/{project_id}/jobs/batch-struct-process` | 批量查询灾备初始化进度 |
| `BatchResetPassword` | PUT | `/v3/{project_id}/jobs/batch-modify-pwd` | 批量修改源库/目标库密码 |
| `BatchRestoreTask` | POST | `/v3/{project_id}/jobs/batch-retry-task` | 批量续传/重试 |
| `BatchSetDefiner` | POST | `/v3/{project_id}/jobs/batch-replace-definer` | 批量设置definer |
| `BatchSetObjects` | PUT | `/v3/{project_id}/jobs/batch-select-objects` | 批量数据库对象选择 |
| `BatchSetPolicy` | POST | `/v3/{project_id}/jobs/batch-sync-policy` | 批量设置同步策略 |
| `BatchSetSmn` | POST | `/v3/{project_id}/jobs/batch-set-smn` | 批量配置异常通知 |
| `BatchSetSpeed` | PUT | `/v3/{project_id}/jobs/batch-limit-speed` | 批量设置任务限速 |
| `BatchShowParams` | POST | `/v3/{project_id}/jobs/batch-get-params` | 批量获取数据库参数 |
| `BatchStartJobs` | POST | `/v3/{project_id}/jobs/batch-starting` | 批量启动任务 |
| `BatchStopJobs` | POST | `/v3/{project_id}/jobs/batch-pause-task` | 批量暂停任务 |
| `BatchStopJobsAction` | POST | `/v5/{project_id}/jobs/batch-stop` | 批量结束任务 |
| `BatchSwitchover` | POST | `/v3/{project_id}/jobs/batch-switchover` | 批量主备倒换 |
| `BatchTagAction` | POST | `/v5/{project_id}/jobs/{resource_type}/{job_id}/tags/action` | 批量添加或删除资源标签 |
| `BatchUpdateJob` | PUT | `/v3/{project_id}/jobs/batch-modification` | 批量修改任务 |

... and 139 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
