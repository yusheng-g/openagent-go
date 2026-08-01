---
name: huaweicloud-idmeclassicapi
description: HuaweiCloud iDMEClassicAPI API guide. 91 APIs covering 业务编码生成器, 关系实体服务, 基础数据服务, 多维视图和多维分支, 失效管理.
---

# HuaweiCloud iDMEClassicAPI API Guide

91 APIs. Tags: 业务编码生成器, 关系实体服务, 基础数据服务, 多维视图和多维分支, 失效管理, 数据分类管理, 标签管理, 树形结构, 版本服务, 生命周期管理, 系统版本, 结构化文档管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddTag` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/addTag` | 绑定标签 |
| `AddToCategory` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/addToCategory` | 添加数据分类 |
| `BatchAddChildNode` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchAddChildNode` | 批量添加实例的子节点 |
| `BatchCheckin` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchCheckin` | 批量检入M-V模型数据实例 |
| `BatchCheckout` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchCheckout` | 批量检出M-V模型数据实例 |
| `BatchCheckoutAndUpdate` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchCheckoutAndUpdate` | 批量检出并更新M-V模型 |
| `BatchCheckoutUndo` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchUndoCheckout` | 批量撤销检出M-V模型数据实例 |
| `BatchCheckoutUndoByAdmin` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchUndoCheckoutByAdmin` | 管理员批量撤销检出M-V模型数据实例 |
| `BatchCreateShareDocs` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/structured-doc/share-doc/batch` | 批量创建分享结构化文档 |
| `BatchCreateUsingPost` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchCreate` | 批量创建实例 |
| `BatchCreateView` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchCreateView` | 批量创建多维视图 |
| `BatchDeleteBranch` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchDeleteBranch` | 批量删除最新大版本下的所有小版本 |
| `BatchDeleteLatestVersion` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batch-delete-latest-version` | 批量删除版本对象下最新分支的最新版本实例数据 |
| `BatchDeleteLogicalBranch` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchLogicalDeleteBranch` | 批量软删除最新大版本下的所有小版本 |
| `BatchDeleteLogicalLatestVersion` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batch-logical-delete-latest-version` | 批量软删除版本对象下最新分支的最新版本实例数据 |
| `BatchDeleteLogicalUsingPost` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchLogicalDelete` | 批量软删除实例 |
| `BatchDeleteShareDocs` | DELETE | `/rdm_{identifier}_app/publicservices/api/{modelName}/structured-doc/share-doc/batch` | 批量删除结构化文档分享权限 |
| `BatchDeleteStructuredDocument` | DELETE | `/rdm_{identifier}_app/publicservices/api/{modelName}/structured-doc/documents/batch` | 批量删除结构化文档 |
| `BatchDeleteUsingPost` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchDelete` | 批量删除实例 |
| `BatchExecuteRevise` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchRevise` | 批量修订M-V模型数据实例 |
| `BatchRemoveChildNode` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchRemoveChildNode` | 批量移除实例的子节点 |
| `BatchShowGetUsingPost` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchGet` | 批量查询实例 |
| `BatchUpdateAndCheckin` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchUpdateAndCheckin` | 批量更新并检入M-V模型数据实例 |
| `BatchUpdateAndRevise` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchReviseAndUpdate` | 批量修订并更新M-V模型数据实例 |
| `BatchUpdateByAdmin` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchUpdateByAdmin` | 管理员批量更新M-V模型数据实例 |
| `BatchUpdateDocument` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/structured-doc/documents/batch/update` | 批量更新结构化文档 |
| `BatchUpdateUsingPost` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchUpdate` | 批量更新实例 |
| `BatchUpdateVersion` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/batchUpdateVersion` | 批量升级M-V模型实例的版本号 |
| `Checkin` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/checkin` | 检入M-V模型数据实例 |
| `Checkout` | POST | `/rdm_{identifier}_app/publicservices/api/{modelName}/checkout` | 检出M-V模型数据实例 |

... and 61 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
