---
name: huaweicloud-koomessage
description: HuaweiCloud KooMessage API guide. 62 APIs covering 一站式创建服务号, 智能信息发送, 智能信息回执, 智能信息基础版发送, 智能信息基础版模板.
---

# HuaweiCloud KooMessage API Guide

62 APIs. Tags: 一站式创建服务号, 智能信息发送, 智能信息回执, 智能信息基础版发送, 智能信息基础版模板, 智能信息服务号主页, 智能信息服务号菜单, 智能信息服务号资料, 智能信息服务号通道号, 智能信息模板, 智能信息解析, 短信发送, 短信应用, 短信模板, 短信签名

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddAimMsgSignature` | POST | `/v1/sms/signatures` | 创建短信签名 |
| `AddCallBack` | POST | `/v1/aim/callbacks` | 注册智能信息回执URL |
| `AddVmsCallBack` | POST | `/v1/aim-basic/callbacks` | 注册智能信息基础版回执URL |
| `CheckMobileCapability` | POST | `/v1/aim/mobile-capabilities/check` | 查询手机号智能信息解析能力 |
| `CreateAimMsgTemplate` | POST | `/v1/sms/templates` | 创建短信模板 |
| `CreateAimPersonalTemplate` | POST | `/v1/aim/templates` | 创建个人模板 |
| `CreateAimSendTask` | POST | `/v1/aim/send-tasks` | 发送智能信息 |
| `CreatePubInfo` | POST | `/v1/aim-sa/unify/pubs` | 一站式服务号创建 |
| `CreateResolveTask` | POST | `/v1/aim/resolve-tasks` | 生成解析任务 |
| `CreateSmsApp` | POST | `/v1/sms/apps` | 创建短信应用 |
| `CreateVmsSendTask` | POST | `/v1/aim-basic/send-tasks` | 发送智能信息基础版任务 |
| `CreateVmsTemplate` | POST | `/v1/aim-basic/templates` | 新建智能信息基础版模板 |
| `DeleteAimMsgSignature` | DELETE | `/v1/sms/signatures/{signature_id}` | 删除短信签名 |
| `DeleteAimMsgTemplate` | DELETE | `/v1/sms/templates/{template_id}` | 删除短信模板 |
| `DeleteAimPersonalTemplate` | DELETE | `/v1/aim/template/{tpl_id}` | 删除模板实例 |
| `DeletePortInfo` | DELETE | `/v1/aim-sa/ports/{port_id}` | 删除通道号 |
| `DeleteTemplateMaterial` | POST | `/v1/aim/template-materials/delete` | 删除模板素材 |
| `FreezePub` | POST | `/v1/aim-sa/pubs/{pub_id}/freeze` | 冻结服务号 |
| `ListAimCallbacks` | GET | `/v1/aim/callbacks` | 查询用户已注册回执接口 |
| `ListAimMsgApp` | GET | `/v1/sms/apps` | 查询短信应用 |
| `ListAimMsgAppDetail` | GET | `/v1/sms/apps/{app_id}` | 获取短信应用详情 |
| `ListAimMsgSignature` | GET | `/v1/sms/signatures` | 查询短信签名 |
| `ListAimMsgSignatureDetail` | GET | `/v1/sms/signatures/{signature_id}` | 获取短信签名详情 |
| `ListAimMsgTemplate` | GET | `/v1/sms/templates` | 查询短信模板 |
| `ListAimResolveDetails` | GET | `/v1/aim/resolve-details` | 查询解析明细 |
| `ListAimSendDetails` | GET | `/v1/aim/send-details` | 查询智能信息发送明细 |
| `ListAimSendReports` | POST | `/v1/aim/send-reports` | 查询智能信息发送报表 |
| `ListAimSendTasks` | GET | `/v1/aim/send-tasks` | 查询智能信息发送任务 |
| `ListAimTemplateMaterials` | GET | `/v1/aim/template-materials` | 查询智能消息模板素材列表 |
| `ListAimTemplateReports` | POST | `/v1/aim/template-reports/query` | 查询模板报表 |

... and 32 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
