---
name: huaweicloud-coc
description: HuaweiCloud COC API guide. 144 APIs covering AccountManagement, Alarm, AssessTask, CocChangeV2, CocIncidentV2.
---

# HuaweiCloud COC API Guide

144 APIs. Tags: AccountManagement, Alarm, AssessTask, CocChangeV2, CocIncidentV2, CustomEventMessageIntegration, Diagnosis, DocumentManagement, EventMessageIntegration, ExecutionManagement, ExternalCOCChange, ExternalCocIncident, ExternalCocIssues, ExternalCocTicket, ListAuthorizableTicketsExternal, ResourceTagManagement, ResourceTags, ScheduledTask, ScriptExecutionManagement, ScriptManagement, ScriptPublicManagement, WarRoom, application, application-model, applicationview, compliant, component, delegation_platform, enterpriseProjectCollect, group, groupResourceRelation, groupRmsResourceRelation, job, multipleCloud, resource, resourceView, vendorAccount

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AcceptScript` | POST | `/v1/job/scripts/{script_uuid}/action` | 审批待审批的脚本 |
| `BatchCreateApplicationView` | POST | `/v1/application-view/batch-create` | 批量创建应用,分组,组件 |
| `CancelDiagnosisTask` | POST | `/v1/diagnosis/tasks/{task_id}/cancel` | 取消诊断任务 |
| `CheckScriptRisk` | POST | `/v1/job/analyze-job` | 评估脚本风险等级 |
| `ClearAlarm` | POST | `/v1/alarm-mgmt/alarms/cancel` | 批量清除告警 |
| `CountGroupRmsResourceRelations` | GET | `/v1/group-resource-relations/count` | 查询分组关联的资源总数 |
| `CountMultiCloudResources` | GET | `/v1/multicloud-resources/count` | 查询用户在云厂商的资源总数 |
| `CountMultiResources` | GET | `/v1/resources/multi-count` | 查询用户各种资源总数 |
| `CountOtherResource` | GET | `/v1/other-resources/count` | 查询线下IDC资源数量 |
| `CountResources` | GET | `/v1/resources/count` | 查询用户资源总数 |
| `CountResourcesOfResourceView` | GET | `/v1/resource/views/resources/count` | 查询CMDB跨账号资源视图聚合的资源总数 |
| `CreateApplication` | POST | `/v1/applications` | 创建应用 |
| `CreateApplicationComponents` | POST | `/v1/components` | 创建组件 |
| `CreateApplicationGroup` | POST | `/v1/groups` | 创建分组 |
| `CreateAssessTask` | POST | `/v1/assess-tasks` | 创建应用评估任务 |
| `CreateAttachment` | POST | `/v1/{ticket_type}/attachments` | 上传附件 |
| `CreateCocIncident` | POST | `/v1/external/incident/create` | CreateExternalIncident 创建事件单 |
| `CreateCocIssues` | POST | `/v1/external/issues/create` | CreateExternalIssues 创建问题单 |
| `CreateDiagnosisTask` | POST | `/v1/diagnosis/tasks` | 提交诊断任务 |
| `CreateDocument` | POST | `/v1/documents` | 创建自定义作业 |
| `CreateExternalCocAttachment` | POST | `/v1/external/incident/attachments` | 上传附件 |
| `CreateGroupRmsResourceRelation` | POST | `/v1/group-resource-relations` | 创建分组资源关联 |
| `CreatePasswordChangePlan` | POST | `/v1/account-mgmt/accounts/password-change-plan` | 创建改密计划 |
| `CreateReportCustomEvent` | POST | `/v1/event/huawei/custom/{integration_key}` | 支持用户自主接入告警数据 |
| `CreateReportPrometheusEvent` | POST | `/v1/event/prometheus/{integration_key}` | Prometheus事件接入 |
| `CreateResourceTags` | POST | `/v1/resources/{resource_id}/tags` | 添加资源标签 |
| `CreateResourceViews` | POST | `/v1/resource/views` | 创建CMDB跨账号资源视图 |
| `CreateScheduledTask` | POST | `/v1/schedule/task` | 新建定时运维 |
| `CreateScript` | POST | `/v1/job/scripts` | 创建脚本 |
| `CreateTicket` | POST | `/v1/{ticket_type}/tickets` | 新建工单 |

... and 114 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
