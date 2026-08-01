---
name: huaweicloud-functiongraph
description: HuaweiCloud FunctionGraph API guide. 92 APIs covering 函数依赖包, 函数导入导出, 函数应用中心, 函数异步配置, 函数指标.
---

# HuaweiCloud FunctionGraph API Guide

92 APIs. Tags: 函数依赖包, 函数导入导出, 函数应用中心, 函数异步配置, 函数指标, 函数日志, 函数模板, 函数流, 函数测试事件, 函数版本别名, 函数生命周期管理, 函数触发器, 函数调用, 函数调用链, 函数配额, 函数预留实例

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AsyncInvokeFunction` | POST | `/v2/{project_id}/fgs/functions/{function_urn}/invocations-async` | 异步执行函数 |
| `BatchDeleteFunctionTriggers` | DELETE | `/v2/{project_id}/fgs/triggers/{function_urn}` | 删除指定函数的所有触发器 |
| `BatchDeleteWorkflows` | DELETE | `/v2/{project_id}/fgs/workflows` | 删除函数流 |
| `CancelAsyncInvocation` | POST | `/v2/{project_id}/fgs/functions/{function_urn}/cancel` | 停止函数异步调用请求 |
| `CreateCallbackWorkflow` | POST | `/v2/{project_id}/fgs/workflows/{workflow_id}/callback` | 回调工作流 |
| `CreateDependencyVersion` | POST | `/v2/{project_id}/fgs/dependencies/version` | 创建依赖包版本 |
| `CreateEvent` | POST | `/v2/{project_id}/fgs/functions/{function_urn}/events` | 创建测试事件 |
| `CreateFunction` | POST | `/v2/{project_id}/fgs/functions` | 创建函数 |
| `CreateFunctionApp` | POST | `/v2/{project_id}/fgs/applications` | 创建应用程序 |
| `CreateFunctionTrigger` | POST | `/v2/{project_id}/fgs/triggers/{function_urn}` | 创建触发器 |
| `CreateFunctionVersion` | POST | `/v2/{project_id}/fgs/functions/{function_urn}/versions` | 发布函数版本 |
| `CreateTags` | POST | `/v2/{project_id}/{resource_type}/{resource_id}/tags/create` | 创建资源标签 |
| `CreateVersionAlias` | POST | `/v2/{project_id}/fgs/functions/{function_urn}/aliases` | 创建函数版本别名 |
| `CreateVpcEndpoint` | POST | `/v2/{project_id}/fgs/vpc-endpoint` | 创建下沉入口 |
| `CreateWorkflow` | POST | `/v2/{project_id}/fgs/workflows` | 创建函数流 |
| `DeleteDependencyVersion` | DELETE | `/v2/{project_id}/fgs/dependencies/{depend_id}/version/{version}` | 删除依赖包版本 |
| `DeleteEvent` | DELETE | `/v2/{project_id}/fgs/functions/{function_urn}/events/{event_id}` | 删除指定测试事件 |
| `DeleteFunction` | DELETE | `/v2/{project_id}/fgs/functions/{function_urn}` | 删除函数/版本 |
| `DeleteFunctionApp` | DELETE | `/v2/{project_id}/fgs/applications/{id}` | 删除应用程序 |
| `DeleteFunctionAsyncInvokeConfig` | DELETE | `/v2/{project_id}/fgs/functions/{function_urn}/async-invoke-config` | 删除函数异步配置信息 |
| `DeleteFunctionTrigger` | DELETE | `/v2/{project_id}/fgs/triggers/{function_urn}/{trigger_type_code}/{trigger_id}` | 删除触发器 |
| `DeleteTags` | DELETE | `/v2/{project_id}/{resource_type}/{resource_id}/tags/delete` | 删除资源标签 |
| `DeleteVersionAlias` | DELETE | `/v2/{project_id}/fgs/functions/{function_urn}/aliases/{alias_name}` | 删除函数版本别名 |
| `DeleteVpcEndpoint` | DELETE | `/v2/{project_id}/fgs/vpc-endpoint/{vpc_id}/{subnet_id}` | 删除下沉入口 |
| `EnableAsyncStatusLog` | POST | `/v2/{project_id}/fgs/functions/enable-async-status-logs` | 允许异步状态通知 |
| `EnableLtsLogs` | POST | `/v2/{project_id}/fgs/functions/enable-lts-logs` | 开通lts日志上报功能 |
| `ExportFunction` | GET | `/v2/{project_id}/fgs/functions/{function_urn}/export` | 导出函数 |
| `ImportFunction` | POST | `/v2/{project_id}/fgs/functions/import` | 导入函数 |
| `InvokeFunction` | POST | `/v2/{project_id}/fgs/functions/{function_urn}/invocations` | 同步执行函数 |
| `ListActiveAsyncInvocations` | GET | `/v2/{project_id}/fgs/functions/{function_urn}/active-async-invocations` | 获取函数活跃异步调用请求列表 |

... and 62 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
