---
name: huaweicloud-ccm
description: HuaweiCloud CCM API guide. 46 APIs covering 查询局点支持特性, 标签管理, 私有CA管理, 私有证书管理, 订单管理.
---

# HuaweiCloud CCM API Guide

46 APIs. Tags: 查询局点支持特性, 标签管理, 私有CA管理, 私有证书管理, 订单管理, 证书吊销处理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateCaTags` | POST | `/v1/private-certificate-authorities/{ca_id}/tags/create` | 批量创建CA标签 |
| `BatchCreateCertTags` | POST | `/v1/private-certificates/{certificate_id}/tags/create` | 批量创建证书标签 |
| `BatchDeleteCaTags` | DELETE | `/v1/private-certificate-authorities/{ca_id}/tags/delete` | 批量删除CA标签 |
| `BatchDeleteCertTags` | DELETE | `/v1/private-certificates/{certificate_id}/tags/delete` | 批量删除证书标签 |
| `CountCaResourceInstances` | POST | `/v1/private-certificate-authorities/resource-instances/count` | 根据标签查询CA数量 |
| `CountCertResourceInstances` | POST | `/v1/private-certificates/resource-instances/count` | 根据标签查询证书数量 |
| `CreateAgency` | POST | `/v1/private-certificate-authorities/agencies` | 创建服务委托 |
| `CreateCaTag` | POST | `/v1/private-certificate-authorities/{ca_id}/tags` | 创建CA标签 |
| `CreateCertificate` | POST | `/v1/private-certificates` | 申请证书 |
| `CreateCertificateAuthority` | POST | `/v1/private-certificate-authorities` | 创建CA |
| `CreateCertificateAuthorityObsAgency` | POST | `/v1/private-certificate-authorities/obs/agencies` | 创建委托 |
| `CreateCertificateAuthorityOrder` | POST | `/v1/private-certificate-authorities/order` | 购买CA |
| `CreateCertificateByCsr` | POST | `/v1/private-certificates/csr` | 通过CSR签发证书 |
| `CreateCertTag` | POST | `/v1/private-certificates/{certificate_id}/tags` | 创建证书标签 |
| `DeleteCertificate` | DELETE | `/v1/private-certificates/{certificate_id}` | 删除证书 |
| `DeleteCertificateAuthority` | DELETE | `/v1/private-certificate-authorities/{ca_id}` | 删除CA |
| `DisableCertificateAuthority` | POST | `/v1/private-certificate-authorities/{ca_id}/disable` | 禁用CA |
| `DisableCertificateAuthorityCrl` | POST | `/v1/private-certificate-authorities/{ca_id}/crl/disable` | 禁用CRL |
| `EnableCertificateAuthority` | POST | `/v1/private-certificate-authorities/{ca_id}/enable` | 启用CA |
| `EnableCertificateAuthorityCrl` | POST | `/v1/private-certificate-authorities/{ca_id}/crl/enable` | 启用CRL |
| `ExportCertificate` | POST | `/v1/private-certificates/{certificate_id}/export` | 导出证书 |
| `ExportCertificateAuthorityCertificate` | POST | `/v1/private-certificate-authorities/{ca_id}/export` | 导出CA证书 |
| `ExportCertificateAuthorityCsr` | GET | `/v1/private-certificate-authorities/{ca_id}/csr` | 导出CA的证书签名请求(CSR) |
| `ImportCertificateAuthorityCertificate` | POST | `/v1/private-certificate-authorities/{ca_id}/import` | 导入CA证书 |
| `IssueCertificateAuthorityCertificate` | POST | `/v1/private-certificate-authorities/{ca_id}/activate` | 激活CA |
| `ListCaResourceInstances` | POST | `/v1/private-certificate-authorities/resource-instances/filter` | 根据标签查询CA列表 |
| `ListCaTags` | GET | `/v1/private-certificate-authorities/{ca_id}/tags` | 根据CA查询标签列表 |
| `ListCertificate` | GET | `/v1/private-certificates` | 查询私有证书列表 |
| `ListCertificateAuthority` | GET | `/v1/private-certificate-authorities` | 查询CA列表 |
| `ListCertificateAuthorityObsBucket` | GET | `/v1/private-certificate-authorities/obs/buckets` | 查询OBS桶列表 |

... and 16 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
