---
name: huaweicloud-kps
description: HuaweiCloud KPS API guide. 18 APIs covering 密钥对任务管理, 密钥对管理.
---

# HuaweiCloud KPS API Guide

18 APIs. Tags: 密钥对任务管理, 密钥对管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AssociateKeypair` | POST | `/v3/{project_id}/keypairs/associate` | 绑定SSH密钥对 |
| `BatchAssociateKeypair` | POST | `/v3/{project_id}/keypairs/batch-associate` | 批量绑定SSH密钥对 |
| `BatchExportPrivateKey` | POST | `/v3/{project_id}/keypairs/private-key/batch-export` | 批量导出密钥对私钥 |
| `BatchImportKeypair` | POST | `/v3/{project_id}/keypairs/batch-import` | 批量导入SSH密钥对 |
| `ClearPrivateKey` | DELETE | `/v3/{project_id}/keypairs/{keypair_name}/private-key` | 清除私钥 |
| `CreateKeypair` | POST | `/v3/{project_id}/keypairs` | 创建和导入SSH密钥对 |
| `DeleteAllFailedTask` | DELETE | `/v3/{project_id}/failed-tasks` | 删除所有失败的任务 |
| `DeleteFailedTask` | DELETE | `/v3/{project_id}/failed-tasks/{task_id}` | 删除失败的任务 |
| `DeleteKeypair` | DELETE | `/v3/{project_id}/keypairs/{keypair_name}` | 删除SSH密钥对 |
| `DisassociateKeypair` | POST | `/v3/{project_id}/keypairs/disassociate` | 解绑SSH密钥对 |
| `ExportPrivateKey` | POST | `/v3/{project_id}/keypairs/private-key/export` | 导出私钥 |
| `ImportPrivateKey` | POST | `/v3/{project_id}/keypairs/private-key/import` | 导入私钥 |
| `ListFailedTask` | GET | `/v3/{project_id}/failed-tasks` | 查询失败的任务信息 |
| `ListKeypairDetail` | GET | `/v3/{project_id}/keypairs/{keypair_name}` | 查询SSH密钥对详细信息 |
| `ListKeypairs` | GET | `/v3/{project_id}/keypairs` | 查询SSH密钥对列表 |
| `ListKeypairTask` | GET | `/v3/{project_id}/tasks/{task_id}` | 查询任务信息 |
| `ListRunningTask` | GET | `/v3/{project_id}/running-tasks` | 查询正在处理的任务信息 |
| `UpdateKeypairDescription` | PUT | `/v3/{project_id}/keypairs/{keypair_name}` | 更新SSH密钥对描述 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
