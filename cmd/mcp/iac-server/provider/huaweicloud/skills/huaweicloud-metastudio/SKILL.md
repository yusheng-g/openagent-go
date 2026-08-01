---
name: huaweicloud-metastudio
description: HuaweiCloud MetaStudio API guide. 293 APIs covering 互动规则库管理, 交互助手MCP服务端对接配置管理, 交互助手大语言模型配置管理, 交互助手对话管理接口, 交互助手指令管理.
---

# HuaweiCloud MetaStudio API Guide

293 APIs. Tags: 互动规则库管理, 交互助手MCP服务端对接配置管理, 交互助手大语言模型配置管理, 交互助手对话管理接口, 交互助手指令管理, 交互助手指令集管理, 交互助手插件配置管理, 交互助手文档分段管理, 交互助手文档管理, 交互助手知识库管理, 交互助手角色管理, 交互助手问答对管理, 分身形象制作管理, 分身数字人字幕文件生成管理, 分身数字人视频制作管理, 声音制作任务管理, 导入导出管理, 数字人名片制作管理, 数字人视频制作管理, 数字资产管理, 数据分析管理, 文件管理, 智能交互数字人委托管理, 智能交互数字人安抚话术管理, 智能交互数字人对话任务管理, 智能交互数字人对话管理, 智能交互数字人对话结果上报配置管理, 智能交互数字人应用管理, 智能交互数字人欢迎词管理, 智能交互数字人激活码管理, 智能交互数字人热点问题管理, 智能交互数字人热词记录管理, 智能交互数字人知识库意图管理, 智能交互数字人知识库技能管理, 智能交互数字人知识库问法管理, 智能交互数字人语音录制配置管理, 智能交互数字人鉴权码管理, 智能直播间管理, 照片数字人视频制作管理, 直播任务管理, 直播商品管理, 直播配置管理, 租户管理, 第三方直播平台管理, 视频制作剧本管理, 订购管理, 训练资源额度管理, 语音合成异步任务管理, 语音合成租户级配置管理, 语音合成管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchDeletePacifyWords` | POST | `/v1/{project_id}/digital-human-chat/pacify-words/delete` | 批量删除安抚话术 |
| `BatchExecuteAssetAction` | POST | `/v1/{project_id}/digital-assets/batch-action` | 批量资产操作 |
| `BindUserAssetResource` | POST | `/v1/{project_id}/tenants/bind-resource` | 资源绑定接口 |
| `Cancel2DDigitalHumanVideo` | POST | `/v1/{project_id}/2d-digital-human-videos/{job_id}/cancel` | 取消等待中的分身数字人视频制作任务 |
| `CancelPhotoDigitalHumanVideo` | POST | `/v1/{project_id}/photo-digital-human-videos/{job_id}/cancel` | 取消等待中的照片分身数字人视频制作任务 |
| `CheckRecallKnowledgeLibrary` | POST | `/v1/{project_id}/wise-brain-manager/knowledge-library/recall` | 知识库召回测试 |
| `CheckVoiceAsset` | POST | `/v1/{project_id}/ttsc/check-voice-asset/{voice_asset_id}` | 校验音色模型是否可用(自研和第三方音色) |
| `CommitShortJob` | POST | `/v1/{project_id}/voice-training-manage/user/short-jobs/{job_id}` | 提交短任务 |
| `CommitVoiceTrainingJob` | POST | `/v1/{project_id}/voice-training-manage/user/jobs/{job_id}` | 提交语音训练任务 |
| `ConfirmFileUpload` | POST | `/v1/{project_id}/files/{file_id}/complete` | 确认文件已上传 |
| `ConfirmSmarLiveRoom` | POST | `/v1/{project_id}/smart-live-rooms/{room_id}/confirm` | 直播间确认 |
| `ConfirmTrainingSegment` | POST | `/v1/{project_id}/voice-training-manage/user/training-segment` | 确认在线录音结果 |
| `CopyVideoScripts` | POST | `/v1/{project_id}/digital-human-video-scripts/{script_id}/copy` | 复制视频制作剧本 |
| `CountTenantResources` | GET | `/v1/{project_id}/tenants/resources-count` | 统计时间段内过期的资源数量 |
| `Create2DDigitalHumanVideo` | POST | `/v1/{project_id}/2d-digital-human-videos` | 创建分身数字人视频制作任务 |
| `Create2dModelTrainingJob` | POST | `/v1/{project_id}/digital-human-training-manage/user/jobs` | 创建分身数字人模型训练任务 |
| `CreateActiveCode` | POST | `/v1/{project_id}/digital-human-chat/active-code` | 创建激活码 |
| `CreateAgencyWithRoleType` | POST | `/v1/{project_id}/digital-human-chat/agency/{role_type}` | 创建委托 |
| `CreateAssetByReplicationInfo` | POST | `/v1/{project_id}/digital-assets-by-replication-info` | 复制资产 |
| `CreateAsyncTtsJob` | POST | `/v1/{project_id}/ttsc/async-jobs` | 创建TTS异步任务 |
| `CreateAudioRecordConfig` | POST | `/v1/{project_id}/digital-human-chat/audio-record-config` | 创建语音录制配置 |
| `CreateBatchKnowledgeQuestion` | POST | `/v1/{project_id}/digital-human-chat/knowledge/question-batch` | 批量创建知识库问法 |
| `CreateDialogReportConfig` | POST | `/v1/{project_id}/digital-human-chat/dialog-report-config` | 创建对话结果上报配置 |
| `CreateDialogUrl` | POST | `/v1/{project_id}/digital-human-chat/create-dialog-url` | 创建对话链接 |
| `CreateDigitalAsset` | POST | `/v1/{project_id}/digital-assets` | 创建资产 |
| `CreateDigitalHumanBusinessCard` | POST | `/v1/{project_id}/digital-human-business-cards` | 创建数字人名片制作 |
| `CreateDocument` | POST | `/v1/{project_id}/wise-brain-manager/document` | 上传文档 |
| `CreateFile` | POST | `/v1/{project_id}/files` | 创建文件并获取上传URL |
| `CreateHotQuestion` | POST | `/v1/{project_id}/digital-human-chat/hot-question` | 创建热点问题 |
| `CreateHotWords` | POST | `/v1/{project_id}/digital-human-chat/hot-words` | 创建热词记录 |

... and 263 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
