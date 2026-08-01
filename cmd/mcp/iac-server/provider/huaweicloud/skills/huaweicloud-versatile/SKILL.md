---
name: huaweicloud-versatile
description: HuaweiCloud Versatile API guide. 4 APIs covering OpenKnowledgeBase, VersatileRuntime.
---

# HuaweiCloud Versatile API Guide

4 APIs. Tags: OpenKnowledgeBase, VersatileRuntime

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `RunAgent` | POST | `/v1/{project_id}/agents/{agent_id}/conversations/{conversation_id}` | 运行一个智能体 |
| `RunWorkflow` | POST | `/v1/{project_id}/workflows/{workflow_id}/conversations/{conversation_id}` | 运行一个工作流 |
| `SearchKnowledgeBase` | POST | `/v2/{project_id}/knowledge-bases/retrieve` | 知识库检索 |
| `UploadAgentFile` | POST | `/v1/{project_id}/agent-runtime/upload-file` | 上传文件 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
