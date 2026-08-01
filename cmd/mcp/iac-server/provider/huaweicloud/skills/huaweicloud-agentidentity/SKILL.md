---
name: huaweicloud-agentidentity
description: HuaweiCloud AgentIdentity API guide. 31 APIs covering ApiKeyCredentialProvider, CredentialProvider, OAuth2Flow, Oauth2CredentialProvider, StsCredentialProvider.
---

# HuaweiCloud AgentIdentity API Guide

31 APIs. Tags: ApiKeyCredentialProvider, CredentialProvider, OAuth2Flow, Oauth2CredentialProvider, StsCredentialProvider, TokenVault, WorkloadAccessToken, WorkloadIdentity

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CompleteResourceTokenAuth` | POST | `/v1/resource-token-auth/complete` | Confirm user authentication session for OAuth2.0 tokens |
| `CreateApiKeyCredentialProvider` | POST | `/v1/api-key-credential-providers` | 创建API密钥凭证提供者 |
| `CreateOauth2CredentialProvider` | POST | `/v1/oauth2-credential-providers` | 创建OAuth2凭证提供者 |
| `CreateStsCredentialProvider` | POST | `/v1/sts-credential-providers` | 创建STS凭证提供者 |
| `CreateWorkloadAccessToken` | POST | `/v1/workload-access-token` | Create workload access token (not acting on behalf of a user) |
| `CreateWorkloadAccessTokenForJwt` | POST | `/v1/workload-access-token-for-jwt` | Create workload access token using JWT (acting on behalf of a user) |
| `CreateWorkloadAccessTokenForUserId` | POST | `/v1/workload-access-token-for-user-id` | Create workload access token using user ID (acting on behalf of a user) |
| `CreateWorkloadIdentity` | POST | `/v1/workload-identities` | 创建工作负载身份 |
| `DeleteApiKeyCredentialProvider` | DELETE | `/v1/api-key-credential-providers/{credential_provider_name}` | 删除API密钥凭证提供者 |
| `DeleteOauth2CredentialProvider` | DELETE | `/v1/oauth2-credential-providers/{credential_provider_name}` | 删除OAuth2凭证提供者 |
| `DeleteStsCredentialProvider` | DELETE | `/v1/sts-credential-providers/{credential_provider_name}` | 删除STS凭证提供者 |
| `DeleteWorkloadIdentity` | DELETE | `/v1/workload-identities/{workload_identity_name}` | 删除工作负载身份 |
| `GetApiKeyCredentialProvider` | GET | `/v1/api-key-credential-providers/{credential_provider_name}` | 查询API密钥凭证提供者详情 |
| `GetOauth2CredentialProvider` | GET | `/v1/oauth2-credential-providers/{credential_provider_name}` | 查询OAuth2凭证提供者详情 |
| `GetResourceApiKey` | POST | `/v1/api-key` | Retrieve API key from resource credential provider |
| `GetResourceOauth2Token` | POST | `/v1/oauth2/token` | Retrieve OAuth2.0 token from resource credential provider |
| `GetResourceStsToken` | POST | `/v1/sts/token` | Retrieve STS credentials from STS credential provider |
| `GetStsCredentialProvider` | GET | `/v1/sts-credential-providers/{credential_provider_name}` | 查询STS凭证提供者详情 |
| `GetTokenVault` | GET | `/v1/token-vaults/{token_vault_id}` | 查询令牌保管库详情 |
| `GetWorkloadIdentity` | GET | `/v1/workload-identities/{workload_identity_name}` | 查询工作负载身份详情 |
| `GetWorkloadIdentityAuthorizerConfiguration` | GET | `/v1/workload-identities/{workload_identity_name}/authorizer-configuration` | 查询工作负载身份的授权配置 |
| `ListApiKeyCredentialProviders` | GET | `/v1/api-key-credential-providers` | 查询API密钥凭证提供者列表 |
| `ListOauth2CredentialProviders` | GET | `/v1/oauth2-credential-providers` | 查询OAuth2凭证提供者列表 |
| `ListStsCredentialProviders` | GET | `/v1/sts-credential-providers` | 查询STS凭证提供者列表 |
| `ListWorkloadIdentities` | GET | `/v1/workload-identities` | 查询工作负载身份列表 |
| `Oauth2Authorize` | GET | `/v1/oauth2/authorize` | OAuth2.0 Pushed Authorization Request (PAR) standard authorize API |
| `Oauth2Callback` | GET | `/v1/oauth2/callback/{credential_provider_id}` | OAuth2.0 Standard Authorization Callback API |
| `UpdateApiKeyCredentialProvider` | PUT | `/v1/api-key-credential-providers/{credential_provider_name}` | 更新API密钥凭证提供者 |
| `UpdateOauth2CredentialProvider` | PUT | `/v1/oauth2-credential-providers/{credential_provider_name}` | 更新OAuth2凭证提供者 |
| `UpdateStsCredentialProvider` | PUT | `/v1/sts-credential-providers/{credential_provider_name}` | 更新STS凭证提供者 |

... and 1 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
