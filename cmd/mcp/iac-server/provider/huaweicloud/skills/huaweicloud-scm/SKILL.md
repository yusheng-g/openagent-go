---
name: huaweicloud-scm
description: HuaweiCloud SCM API guide. 27 APIs covering CSR管理, 证书标签管理, 证书生命周期管理, 证书申请管理, 证书订单管理.
---

# HuaweiCloud SCM API Guide

27 APIs. Tags: CSR管理, 证书标签管理, 证书生命周期管理, 证书申请管理, 证书订单管理, 证书部署管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `ApplyCertificate` | POST | `/v3/scm/certificates/{certificate_id}/apply` | 申请证书 |
| `BatchCreateOrDeleteTags` | POST | `/v3/scm/{resource_id}/tags/action` | 批量创建或删除标签 |
| `BatchPushCertificate` | POST | `/v3/scm/certificates/{certificate_id}/batch-push` | 批量推送证书 |
| `CancelCertificateRequest` | POST | `/v3/scm/certificates/{certificate_id}/cancel-cert` | 撤回证书申请 |
| `CreateCertificateTag` | POST | `/v3/scm/{resource_id}/tags` | 创建标签 |
| `CreateCsr` | POST | `/v3/scm/csr` | 创建CSR |
| `DeleteCertificate` | DELETE | `/v3/scm/certificates/{certificate_id}` | 删除证书 |
| `DeleteCsr` | DELETE | `/v3/scm/csr/{id}` | 删除CSR |
| `DeployCertificate` | POST | `/v3/scm/certificates/{certificate_id}/deploy` | 部署证书 |
| `DisableNotification` | POST | `/v3/scm/notification/disable` | 禁用证书提醒 |
| `EnableNotification` | POST | `/v3/scm/notification/enable` | 启用证书提醒 |
| `ExportCertificate` | POST | `/v3/scm/certificates/{certificate_id}/export` | 导出证书 |
| `ImportCertificate` | POST | `/v3/scm/certificates/import` | 导入证书 |
| `ListAllTags` | GET | `/v3/scm/tags` | 查询所有标签列表 |
| `ListCertificates` | GET | `/v3/scm/certificates` | 查询证书列表 |
| `ListCertificatesByTag` | POST | `/v3/scm/{resource_instances}/action` | 根据标签查询证书列表 |
| `ListCsr` | GET | `/v3/scm/csr` | 查询CSR列表 |
| `ListDeployedResources` | POST | `/v3/scm/deployed-resources` | 查询已部署资源 |
| `ListTagsByCertificate` | GET | `/v3/scm/{resource_id}/tags` | 根据证书ID查询标签列表 |
| `PushCertificate` | POST | `/v3/scm/certificates/{certificate_id}/push` | 推送证书 |
| `ShowCertificate` | GET | `/v3/scm/certificates/{certificate_id}` | 获取证书详情 |
| `ShowCsr` | GET | `/v3/scm/csr/{id}` | 查询CSR |
| `ShowCsrPrivateKey` | GET | `/v3/scm/csr/{id}/private-key` | 查询私钥 |
| `SubscribeCertificate` | POST | `/v3/scm/certificates/buy` | 购买SSL证书 |
| `UnsubscribeCertificate` | DELETE | `/v3/scm/certificates/{cert_id}/unsubscribe` | 退订证书 |
| `UpdateCsr` | PUT | `/v3/scm/csr/{id}` | 更新CSR |
| `UploadCsr` | POST | `/v3/scm/csr/upload` | 上传CSR |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
