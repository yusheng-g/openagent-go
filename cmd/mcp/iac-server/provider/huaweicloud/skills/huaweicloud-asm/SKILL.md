---
name: huaweicloud-asm
description: HuaweiCloud ASM API guide. 4 APIs covering 网格接口.
---

# HuaweiCloud ASM API Guide

4 APIs. Tags: 网格接口

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateMesh` | POST | `/v1/{project_id}/meshes` | 创建网格 |
| `DeleteMesh` | DELETE | `/v1/{project_id}/meshes/{mesh_id}` | 删除网格 |
| `ListMeshes` | GET | `/v1/{project_id}/meshes` | 查询网格列表 |
| `ShowMesh` | GET | `/v1/{project_id}/meshes/{mesh_id}` | 查询网格 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
