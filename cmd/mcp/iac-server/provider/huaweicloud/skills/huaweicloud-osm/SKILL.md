---
name: huaweicloud-osm
description: HuaweiCloud OSM API guide. 82 APIs covering 一键诊断, 关联工单管理, 协议管理, 反馈, 工单权限管理.
---

# HuaweiCloud OSM API Guide

82 APIs. Tags: 一键诊断, 关联工单管理, 协议管理, 反馈, 工单权限管理, 工单查询相关接口, 工单留言管理, 工单管理, 工单配额管理, 授权管理, 提单基础配置查询, 标签管理, 配置管理, 附件功能, 附件管理, 验证码管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CheckHosts` | POST | `/v2/servicerequest/authorizations/authorization-details/{authorization_detail_id}/verify-host` | 验证授权主机 |
| `CheckNeedVerify` | GET | `/v2/servicerequest/verifycodes/need-verify` | 是否需要验证 |
| `CheckVerifyCodes` | POST | `/v2/servicerequest/verifycodes` | 验证联系方式 |
| `ConfirmAuthorizations` | PUT | `/v2/servicerequest/authorizations/{authorization_id}` | 确认授权 |
| `CreateCaseExtendsParam` | POST | `/v2/servicerequest/cases/{case_id}/extends-param` | 提交工单扩展参数 |
| `CreateCaseLabels` | POST | `/v2/servicerequest/cases/{case_id}/labels` | 添加工单关联标签接口 |
| `CreateCases` | POST | `/v2/servicerequest/cases` | 创建工单 |
| `CreateDiagnoseFeedback` | POST | `/v2.0/servicerequest/diagnose/feedback` | 用户反馈是否有帮助 |
| `CreateDiagnoseJob` | POST | `/v2.0/servicerequest/diagnose/job/start` | 开始一键诊断 |
| `CreateFeedback` | POST | `/v2/servicerequest/feedbacks` | 创建举报反馈 |
| `CreateLabels` | POST | `/v2/servicerequest/labels` | 创建标签 |
| `CreateMessages` | POST | `/v2/servicerequest/cases/{case_id}/message` | 提交留言 |
| `CreatePrivileges` | POST | `/v2/servicerequest/privileges` | 创建授权 |
| `CreateRelations` | POST | `/v2/servicerequest/cases/{case_id}/relations` | 创建关联 |
| `CreateScores` | POST | `/v2/servicerequest/cases/{case_id}/score` | 提交评分 |
| `DeleteAccessories` | DELETE | `/v2/servicerequest/accessorys/{accessory_id}` | 删除附件 |
| `DeleteCaseLabels` | DELETE | `/v2/servicerequest/cases/{case_id}/labels` | 删除工单关联标签接口 |
| `DeleteLabels` | DELETE | `/v2/servicerequest/labels/{label_id}` | 删除标签 |
| `DeleteRelation` | DELETE | `/v2/servicerequest/cases/{case_id}/relations` | 删除关联 |
| `DownloadAccessories` | GET | `/v2/servicerequest/accessorys/{accessory_id}` | 下载附件 |
| `DownloadCases` | GET | `/v2/servicerequest/cases/export` | 工单导出 |
| `DownloadImages` | GET | `/v2/servicerequest/accessorys/{accessory_id}/image` | 图片展示 |
| `ListAccessoryAccessUrls` | GET | `/v2/servicerequest/accessorys/access-urls` | 租户批量获取下载链接 |
| `ListAgencies` | GET | `/v2/servicerequest/agencies` | 查询委托 |
| `ListAreaCodes` | GET | `/v2/servicerequest/config/area-codes` | 查询国家码 |
| `ListAuthorizations` | GET | `/v2/servicerequest/authorizations` | 查看授权列表 |
| `ListCaseCategories` | GET | `/v2/servicerequest/config/categories` | 查询工单类目列表 |
| `ListCaseCcEmails` | GET | `/v2/servicerequest/carbon-copy-emails` | 查询工单抄送邮箱 |
| `ListCaseCounts` | GET | `/v2/servicerequest/cases/count` | 统计各状态工单数量 |
| `ListCaseLabels` | GET | `/v2/servicerequest/cases/{case_id}/labels` | 查询工单关联标签接口 |

... and 52 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
