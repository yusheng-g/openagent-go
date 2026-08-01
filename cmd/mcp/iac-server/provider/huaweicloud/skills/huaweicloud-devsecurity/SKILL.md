---
name: huaweicloud-devsecurity
description: HuaweiCloud DevSecurity API guide. 18 APIs covering 二进制成分分析.
---

# HuaweiCloud DevSecurity API Guide

18 APIs. Tags: 二进制成分分析

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateReportExcel` | POST | `/v1/{project_id}/sbc/report/excel/create` | 创建报告Excel |
| `CreateReportPdf` | POST | `/v1/{project_id}/sbc/report/pdf/create` | 创建报告PDF |
| `CreateTask` | POST | `/v1/{project_id}/sbc/task/start` | 创建二进制成分分析扫描任务 |
| `CreateTaskMultipartFile` | POST | `/v1/{project_id}/sbc/task/multipart/create` | 创建上传分片文件任务 |
| `DeleteTask` | DELETE | `/v1/{project_id}/sbc/task` | 删除任务 |
| `DownloadReportExcel` | GET | `/v1/{project_id}/sbc/report/excel` | 下载报告Excel |
| `DownloadReportPdf` | GET | `/v1/{project_id}/sbc/report/pdf` | 下载报告PDF |
| `NotifyTaskMultipartFile` | POST | `/v1/{project_id}/sbc/task/multipart/notify` | 结束上传分片文件任务 |
| `ShowInfoLeakSummary` | GET | `/v1/{project_id}/sbc/task/summary/infoleak` | 获取密钥和信息泄露统计数据 |
| `ShowOpenSourceReport` | GET | `/v1/{project_id}/sbc/task/report/opensource` | 获取开源漏洞分析报告 |
| `ShowOpenSourceSummary` | GET | `/v1/{project_id}/sbc/task/summary/opensource` | 获取开源漏洞分析统计数据 |
| `ShowReportExcelStatus` | GET | `/v1/{project_id}/sbc/report/excel/status` | 查看报告Excel状态 |
| `ShowReportPdfStatus` | GET | `/v1/{project_id}/sbc/report/pdf/status` | 查看报告PDF状态 |
| `ShowSecCompileSummary` | GET | `/v1/{project_id}/sbc/task/summary/seccompile` | 获取安全编译选项统计数据 |
| `ShowSecConfigSummary` | GET | `/v1/{project_id}/sbc/task/summary/secconfig` | 获取安全配置统计数据 |
| `ShowTaskStatus` | GET | `/v1/{project_id}/sbc/task/status` | 获取任务状态 |
| `StopTask` | POST | `/v1/{project_id}/sbc/task/stop` | 停止任务 |
| `UploadTaskMultipartFile` | POST | `/v1/{project_id}/sbc/task/multipart` | 上传分片文件 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
