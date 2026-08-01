---
name: huaweicloud-kms
description: HuaweiCloud KMS API guide. 61 APIs covering 专属密钥库管理, 别名管理, 多区域密钥, 密钥协商, 密钥授权管理.
---

# HuaweiCloud KMS API Guide

61 APIs. Tags: 专属密钥库管理, 别名管理, 多区域密钥, 密钥协商, 密钥授权管理, 密钥查询, 密钥标签管理, 密钥生命周期管理, 密钥轮换管理, 导入密钥管理, 小数据加解密, 数据密钥管理, 查询密钥API版本信息, 消息验证码, 签名验签

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AssociateAlias` | POST | `/v1.0/{project_id}/kms/alias/associate` | None |
| `BatchCreateKmsTags` | POST | `/v1.0/{project_id}/kms/{key_id}/tags/action` | 批量添加删除密钥标签 |
| `CancelGrant` | POST | `/v1.0/{project_id}/kms/revoke-grant` | 撤销授权 |
| `CancelKeyDeletion` | POST | `/v1.0/{project_id}/kms/cancel-key-deletion` | 取消计划删除密钥 |
| `CancelSelfGrant` | POST | `/v1.0/{project_id}/kms/retire-grant` | 退役授权 |
| `CreateAlias` | POST | `/v1.0/{project_id}/kms/aliases` | None |
| `CreateDatakey` | POST | `/v1.0/{project_id}/kms/create-datakey` | 创建数据密钥 |
| `CreateDatakeyWithoutPlaintext` | POST | `/v1.0/{project_id}/kms/create-datakey-without-plaintext` | 创建不含明文数据密钥 |
| `CreateEcDatakeyPair` | POST | `/v1.0/{project_id}/kms/create-ec-datakey-pair` | 创建EC数据密钥对 |
| `CreateGrant` | POST | `/v1.0/{project_id}/kms/create-grant` | 创建授权 |
| `CreateKey` | POST | `/v1.0/{project_id}/kms/create-key` | 创建密钥 |
| `CreateKeyStore` | POST | `/v1.0/{project_id}/keystores` | 创建专属密钥库 |
| `CreateKmsTag` | POST | `/v1.0/{project_id}/kms/{key_id}/tags` | 添加密钥标签 |
| `CreateParametersForImport` | POST | `/v1.0/{project_id}/kms/get-parameters-for-import` | 获取密钥导入参数 |
| `CreatePin` | POST | `/v1.0/{project_id}/kms/create-pin` | 创建PIN码 |
| `CreateRandom` | POST | `/v1.0/{project_id}/kms/gen-random` | 创建随机数 |
| `CreateRsaDatakeyPair` | POST | `/v1.0/{project_id}/kms/create-rsa-datakey-pair` | 创建RSA数据密钥对 |
| `DecryptData` | POST | `/v1.0/{project_id}/kms/decrypt-data` | 解密数据 |
| `DecryptDatakey` | POST | `/v1.0/{project_id}/kms/decrypt-datakey` | 解密数据密钥 |
| `DeleteAlias` | DELETE | `/v1.0/{project_id}/kms/aliases` | None |
| `DeleteImportedKeyMaterial` | POST | `/v1.0/{project_id}/kms/delete-imported-key-material` | 删除密钥材料 |
| `DeleteKey` | POST | `/v1.0/{project_id}/kms/schedule-key-deletion` | 计划删除密钥 |
| `DeleteKeyStore` | DELETE | `/v1.0/{project_id}/keystores/{keystore_id}` | 删除专属密钥库 |
| `DeleteTag` | DELETE | `/v1.0/{project_id}/kms/{key_id}/tags/{key}` | 删除密钥标签 |
| `DeriveSharedSecret` | POST | `/v1.0/{project_id}/kms/derive-shared-secret` | 派生共享密钥 |
| `DisableKey` | POST | `/v1.0/{project_id}/kms/disable-key` | 禁用密钥 |
| `DisableKeyRotation` | POST | `/v1.0/{project_id}/kms/disable-key-rotation` | 关闭密钥轮换 |
| `DisableKeyStore` | POST | `/v1.0/{project_id}/keystores/{keystore_id}/disable` | 禁用专属密钥库 |
| `EnableKey` | POST | `/v1.0/{project_id}/kms/enable-key` | 启用密钥 |
| `EnableKeyRotation` | POST | `/v1.0/{project_id}/kms/enable-key-rotation` | 开启密钥轮换 |

... and 31 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
