---
name: huaweicloud-sms
description: HuaweiCloud SMS API guide. 52 APIs covering Agent运行, 任务管理, 历史API, 命令管理, 密钥管理.
---

# HuaweiCloud SMS API Guide

52 APIs. Tags: Agent运行, 任务管理, 历史API, 命令管理, 密钥管理, 查询API版本信息, 模板管理, 源端管理, 网络检测管理, 迁移项目管理, 配置设置管理, 隐私协议管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CheckNetAcl` | None | `None` | 检查网卡安全组端口是否符合要求 |
| `CollectLog` | POST | `/v3/tasks/{task_id}/log` | 上传迁移任务的日志 |
| `CreateMigproject` | POST | `/v3/migprojects` | 新建迁移项目 |
| `CreatePrivacyAgreements` | POST | `/v3/privacy-agreements` | 同意隐私协议 |
| `CreateTask` | POST | `/v3/tasks` | 创建迁移任务 |
| `CreateTemplate` | POST | `/v3/vm/templates` | 新增模板信息 |
| `DeleteMigproject` | DELETE | `/v3/migprojects/{mig_project_id}` | 删除迁移项目 |
| `DeleteServer` | DELETE | `/v3/sources/{source_id}` | 删除指定ID的源端服务器信息 |
| `DeleteServers` | POST | `/v3/sources/delete` | 批量删除源端服务器信息 |
| `DeleteTask` | DELETE | `/v3/tasks/{task_id}` | 删除指定ID的迁移任务 |
| `DeleteTasks` | POST | `/v3/tasks/delete` | 批量删除迁移任务 |
| `DeleteTemplate` | DELETE | `/v3/vm/templates/{id}` | 删除指定ID的模板 |
| `DeleteTemplates` | POST | `/v3/vm/templates/delete` | 批量删除指定ID的模板 |
| `ExportConsistencyResults` | POST | `/v3/tasks/consistency-results/export` | 批量获取一致性校验结果 |
| `ListApiVersion` | GET | `/` | 查询主机迁移服务的API版本信息 |
| `ListErrorServers` | GET | `/v3/errors` | 查询待迁移源端的所有错误 |
| `ListMigprojects` | GET | `/v3/migprojects` | 获取项目列表 |
| `ListServers` | GET | `/v3/sources` | 查询源端服务器列表 |
| `ListTasks` | GET | `/v3/tasks` | 查询迁移任务列表 |
| `ListTemplates` | GET | `/v3/vm/templates` | 查询模板列表 |
| `RegisterServer` | POST | `/v3/sources` | 上报源端服务器基本信息 |
| `ShowApiVersion` | GET | `/{version}` | 查询主机迁移服务指定API版本信息 |
| `ShowCertKey` | GET | `/v3/tasks/{task_id}/certkey` | 获取SSL证书和私钥 |
| `ShowCommand` | GET | `/v3/sources/{server_id}/command` | 获取服务端命令 |
| `ShowConfig` | GET | `/v3/config` | 获取Agent配置信息 |
| `ShowConfigSetting` | GET | `/v3/tasks/{task_id}/configuration-setting` | 查询配置资源 |
| `ShowConsistencyResult` | GET | `/v3/tasks/{task_id}/consistency-result` | 获取一致性校验结果 |
| `ShowMigproject` | GET | `/v3/migprojects/{mig_project_id}` | 查询指定ID迁移项目详情 |
| `ShowOverview` | GET | `/v3/sources/overview` | 获取服务器总览 |
| `ShowPassphrase` | GET | `/v3/tasks/{task_id}/passphrase` | 查询指定任务ID的安全传输通道的证书passphrase |

... and 22 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
