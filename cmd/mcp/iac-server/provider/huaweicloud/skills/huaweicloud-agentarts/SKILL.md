---
name: huaweicloud-agentarts
description: HuaweiCloud AgentArts API guide. 172 APIs covering 人工标注管理, 代码解释器管理, 会话管理, 入站网络, 指标管理.
---

# HuaweiCloud AgentArts API Guide

172 APIs. Tags: 人工标注管理, 代码解释器管理, 会话管理, 入站网络, 指标管理, 日志管理, 智能体管理, 模型管理, 网关后端管理, 网关标签管理, 网关资源管理, 订阅管理, 记忆库标签管理, 记忆库管理, 记忆异步任务管理, 记忆检索, 记忆策略管理, 记忆网络访问, 记忆访问密钥管理, 评估任务管理, 评估器模板管理, 评估器管理, 评测集管理, 调用链管理, 运行时, 运行时标签管理, 运行时版本, 运行时访问方式, 运行时访问方式标签管理, 配额管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchAddOpsEvaluationTaskCustomLabels` | POST | `/v1/ops/evaluation-tasks/{task_id}/custom-labels` | 添加评估任务自定义标签 |
| `BatchAddOpsEvaluationTaskCustomLabelValues` | POST | `/v1/ops/evaluation-tasks/{task_id}/custom-labels-values` | 批量添加评估任务的自定义标签值 |
| `BatchCreateCoreGatewayTags` | POST | `/v1/gateways/{gateway_id}/tags/create` | 批量添加网关标签 |
| `BatchCreateCoreRuntimeEndpointTags` | POST | `/v1/runtime-endpoints/{endpoint_id}/tags/create` | 批量为RuntimeEndpoint打资源标签 |
| `BatchCreateCoreRuntimeTags` | POST | `/v1/runtimes/{runtime_id}/tags/create` | 批量为Runtime打资源标签 |
| `BatchCreateCoreSpaceTags` | POST | `/v1/memory-space/{space_id}/tags/create` | 批量添加记忆库标签 |
| `BatchCreateOpsDatasetItems` | POST | `/v1/ops/datasets/{dataset_id}/items` | 批量添加评测集条目 |
| `BatchDeleteCoreGatewayTags` | POST | `/v1/gateways/{gateway_id}/tags/delete` | 批量删除网关标签 |
| `BatchDeleteCoreRuntimeEndpointTags` | POST | `/v1/runtime-endpoints/{endpoint_id}/tags/delete` | 批量删除RuntimeEndpoint资源标签 |
| `BatchDeleteCoreRuntimeTags` | POST | `/v1/runtimes/{runtime_id}/tags/delete` | 批量删除Runtime资源标签 |
| `BatchDeleteCoreSpaceTags` | POST | `/v1/memory-space/{space_id}/tags/delete` | 批量删除记忆库标签 |
| `BatchDeleteOpsDatasetItems` | DELETE | `/v1/ops/datasets/{dataset_id}/items` | 批量删除评测集条目 |
| `BatchDeleteOpsDatasets` | DELETE | `/v1/ops/datasets` | 批量删除评测集 |
| `BatchDeleteOpsEvaluationTasks` | DELETE | `/v1/ops/evaluation-tasks` | 批量删除任务 |
| `BatchDeleteOpsEvaluator` | DELETE | `/v1/ops/evaluators` | 批量删除评估器 |
| `BatchDeleteOpsSynthesisTasks` | DELETE | `/v1/ops/datasets-synthesis` | 批量删除评测集合成任务 |
| `BatchUpdateOpsEvaluationTaskCustomLabelValues` | PUT | `/v1/ops/evaluation-tasks/{task_id}/custom-labels-values` | 更新评估任务的自定义标签值 |
| `CheckOpsEvaluationTaskName` | GET | `/v1/ops/evaluation-tasks/name/check` | 检查新建评估任务名称是否存在 |
| `CreateCoreCodeInterpreter` | POST | `/v1/core/code-interpreters` | 创建代码解释器 |
| `CreateCoreGateway` | POST | `/v1/core/gateways` | 创建网关 |
| `CreateCoreGatewayTarget` | POST | `/v1/core/gateways/{gateway_id}/targets` | 创建目标服务 |
| `CreateCoreIngress` | POST | `/v1/core/ingresses` | 创建网关 |
| `CreateCoreIngressNetwork` | POST | `/v1/core/ingresses/{ingress_id}/vpc-networks` | 创建VPC入站网络 |
| `CreateCoreRuntime` | POST | `/v1/core/runtimes` | 创建runtime |
| `CreateCoreRuntimeEndpoint` | POST | `/v1/core/runtimes/{runtime_id}/endpoints` | 创建runtime访问方式 |
| `CreateCoreSpace` | POST | `/v1/core/spaces` | 创建Space |
| `CreateCoreSpaceApiKey` | POST | `/v1/core/space-keys` | 创建 API Key |
| `CreateCoreSpaceCustomizedStrategy` | POST | `/v1/core/spaces/{space_id}/strategies/customized` | 创建自定义记忆策略 |
| `CreateOpsAgentObservation` | POST | `/v1/ops/observation/agents` | 同步agent信息 |
| `CreateOpsDataset` | POST | `/v1/ops/datasets` | 创建评测集 |

... and 142 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
