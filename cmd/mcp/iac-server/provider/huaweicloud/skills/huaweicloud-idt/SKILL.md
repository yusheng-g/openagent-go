---
name: huaweicloud-idt
description: HuaweiCloud IDT API guide. 500 APIs covering Linkx同步任务, XDM基线对象, XDM权限属性, 关联模型图记录, 分类管理.
---

# HuaweiCloud IDT API Guide

500 APIs. Tags: Linkx同步任务, XDM基线对象, XDM权限属性, 关联模型图记录, 分类管理, 分类节点的分组, 单位类型, 参与者关系的操作历史记录, 合法值, 合法值类型, 团队, 团队与团队角色关系, 团队角色, 团队里的团队角色成员关系, 基线对象与被基线对象的关系, 属性值与属性库关系表, 属性库管理, 搜索服务API, 搜索服务定义, 搜索服务定义主人, 操作关系, 数据实体授权, 数据实例授权, 数据实例鉴权, 文件夹, 文件夹内容, 文件管理, 服务编排, 权限, 权限分配, 权限操作, 权限管理功能与团队关系, 权限管理功能与策略集关系, 标签, 标签与对象关系, 标签分组, 特征点, 生命周期业务操作, 生命周期操作类型, 生命周期模板, 生命周期模板与实体模型的关系, 生命周期状态, 生命周期阶段, 用户管理, 用户组成员, 用户群组, 租户, 策略, 策略集, 类型定义, 类型定义模型继承关系, 索引字段宽表, 维度属性映射模型, 规则, 角色, 角色成员, 计量单位

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddAttributeattribute` | POST | `/rdm_{appName}_app/services/rdm/basic/api/TypeDefinition/attribute/add` | 添加属性 |
| `AddTagExaDefinition` | POST | `/rdm_{appName}_app/services/rdm/common/api/EXADefinition/addTag` | addTag |
| `AddTagXdmSearchServDefineMaster` | POST | `/rdm_{appName}_app/services/rdm/common/api/XDMSearchServDefineMaster/addTag` | addTag |
| `AttachTagToExaAttrtag` | POST | `/rdm_{appName}_app/services/rdm/basic/api/tag/attachTagToExaAttr` | 添加扩展属性标签 |
| `CheckHasAccessAccessService` | POST | `/rdm_{appName}_app/services/rdm/basic/api/AccessService/hasAccess` | 实例权限鉴定 |
| `CheckinXdmSearchServDefine` | POST | `/rdm_{appName}_app/services/rdm/common/api/XDMSearchServDefine/checkin` | checkin |
| `CheckoutAndUpdateXdmSearchServDefine` | POST | `/rdm_{appName}_app/services/rdm/common/api/XDMSearchServDefine/checkoutAndUpdate` | checkoutAndUpdate |
| `CheckoutLifecycleTemplate` | POST | `/rdm_{appName}_app/services/rdm/common/api/LifecycleTemplate/checkout` | checkout |
| `CheckoutXdmSearchServDefine` | POST | `/rdm_{appName}_app/services/rdm/common/api/XDMSearchServDefine/checkout` | checkout |
| `CheckUniqueExaDefinition` | POST | `/rdm_{appName}_app/services/rdm/basic/api/EXADefinition/checkUnique` | 检验唯一性 |
| `CheckUniqueFolder` | POST | `/rdm_{appName}_app/services/rdm/basic/api/Folder/checkUnique` | 检验唯一性 |
| `CheckUniqueLifecycleTemplate` | POST | `/rdm_{appName}_app/services/rdm/basic/api/LifecycleTemplate/checkUnique` | 检验唯一性 |
| `CheckUniquetag` | POST | `/rdm_{appName}_app/services/rdm/basic/api/tag/checkunique` | 检验唯一性 |
| `CheckUniquetaggroup` | POST | `/rdm_{appName}_app/services/rdm/basic/api/taggroup/checkunique` | 检验唯一性 |
| `CheckUniqueunique` | POST | `/rdm_{appName}_app/services/rdm/basic/api/customservice/unique/check` | 检验参数是否唯一 |
| `CompareBusinessVersionLifecycleTemplate` | POST | `/rdm_{appName}_app/services/rdm/common/api/LifecycleTemplate/compareBusinessVersion` | compareBusinessVersion |
| `CompareBusinessVersionXdmSearchServDefine` | POST | `/rdm_{appName}_app/services/rdm/common/api/XDMSearchServDefine/compareBusinessVersion` | compareBusinessVersion |
| `CompareVersionBaseLine` | POST | `/rdm_{appName}_app/services/rdm/common/api/BaseLine/compareVersion` | compareVersion |
| `CompareVersionBaseLineLink` | POST | `/rdm_{appName}_app/services/rdm/common/api/BaseLineLink/compareVersion` | compareVersion |
| `CompareVersionClassificationNode` | POST | `/rdm_{appName}_app/services/rdm/common/api/ClassificationNode/compareVersion` | compareVersion |
| `CompareVersionClassificationNodeGroup` | POST | `/rdm_{appName}_app/services/rdm/common/api/ClassificationNodeGroup/compareVersion` | compareVersion |
| `CompareVersionExaDefinition` | POST | `/rdm_{appName}_app/services/rdm/common/api/EXADefinition/compareVersion` | compareVersion |
| `CompareVersionExaDefinitionLink` | POST | `/rdm_{appName}_app/services/rdm/common/api/EXADefinitionLink/compareVersion` | compareVersion |
| `CompareVersionFolder` | POST | `/rdm_{appName}_app/services/rdm/common/api/Folder/compareVersion` | compareVersion |
| `CompareVersionLifecycleBusinessOperation` | POST | `/rdm_{appName}_app/services/rdm/common/api/LifecycleBusinessOperation/compareVersion` | compareVersion |
| `CompareVersionLifecycleOperation` | POST | `/rdm_{appName}_app/services/rdm/common/api/LifecycleOperation/compareVersion` | compareVersion |
| `CompareVersionLifecyclePhase` | POST | `/rdm_{appName}_app/services/rdm/common/api/LifecyclePhase/compareVersion` | compareVersion |
| `CompareVersionLifecycleState` | POST | `/rdm_{appName}_app/services/rdm/common/api/LifecycleState/compareVersion` | compareVersion |
| `CompareVersionLifecycleTemplate` | POST | `/rdm_{appName}_app/services/rdm/common/api/LifecycleTemplate/compareVersion` | compareVersion |
| `CompareVersionLinkxSyncTask` | POST | `/rdm_{appName}_app/services/rdm/common/api/LinkxSyncTask/compareVersion` | compareVersion |

... and 470 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
