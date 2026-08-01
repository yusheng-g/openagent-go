---
name: huaweicloud-cpts
description: HuaweiCloud CPTS API guide. 45 APIs covering PerfTest工程管理, 事务管理, 任务管理, 全局变量管理, 全链路压测管理.
---

# HuaweiCloud CPTS API Guide

45 APIs. Tags: PerfTest工程管理, 事务管理, 任务管理, 全局变量管理, 全链路压测管理, 报告管理, 用例管理, 目录管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchUpdateTaskStatus` | POST | `/v1/{project_id}/test-suites/{test_suit_id}/tasks/batch-update-task-status` | 批量启停任务 |
| `CreateCase` | POST | `/v1/{project_id}/task-cases` | 创建用例(旧版) |
| `CreateDirectory` | POST | `/v1/{project_id}/test-suites/{test_suite_id}/directory` | 创建目录 |
| `CreateNewCase` | POST | `/v2/{project_id}/test-cases` | 创建用例 |
| `CreateNewTask` | POST | `/v3/{project_id}/tasks` | 创建任务 |
| `CreateProject` | POST | `/v1/{project_id}/test-suites` | 创建工程 |
| `CreateTask` | POST | `/v1/{project_id}/tasks` | 创建任务(旧版) |
| `CreateTemp` | POST | `/v1/{project_id}/templates` | 创建事务 |
| `CreateVariable` | POST | `/v1/{project_id}/variables/{test_suite_id}` | 创建变量 |
| `DebugCase` | POST | `/v1/{project_id}/test-suites/{test_suite_id}/tasks/{task_id}/cases/{case_id}/debug` | 调试用例 |
| `DeleteCase` | DELETE | `/v1/{project_id}/task-cases/{case_id}` | 删除用例(旧版) |
| `DeleteDirectory` | DELETE | `/v1/{project_id}/test-suites/{test_suite_id}/directory/{directory_id}` | 删除目录 |
| `DeleteNewCase` | DELETE | `/v2/{project_id}/test-cases/{case_id}` | 删除用例 |
| `DeleteNewTask` | DELETE | `/v3/{project_id}/tasks/{task_id}` | 删除任务 |
| `DeleteProject` | DELETE | `/v1/{project_id}/test-suites/{test_suite_id}` | 删除工程 |
| `DeleteTask` | DELETE | `/v1/{project_id}/tasks/{task_id}` | 删除任务(旧版) |
| `DeleteTemp` | DELETE | `/v1/{project_id}/templates/{template_id}` | 删除事务 |
| `DeleteVariable` | DELETE | `/v1/{project_id}/variables` | 删除全局变量 |
| `ListProjectSets` | GET | `/v1/{project_id}/test-suites` | 查询工程集 |
| `ListProjectTestCase` | GET | `/v1/{project_id}/test-suites/{test_suite_id}/directory` | 查询用例树 |
| `ListTaskCases` | GET | `/v1/{project_id}/test-suites/{test_suit_id}/tasks/{task_id}/test-cases` | 获取任务关联的用例列表 |
| `ListVariables` | GET | `/v1/{project_id}/variables/{variable_type}/test-suites/{test_suite_id}` | 查询全局变量 |
| `ShowAgentConfig` | POST | `/v1/{project_id}/stress/agents` | 全链路压测探针获取配置信息 |
| `ShowCase` | GET | `/v2/{project_id}/test-cases/{case_id}` | 查询用例 |
| `ShowHistoryRunInfo` | GET | `/v1/{project_id}/tasks/history-run-list/{task_id}` | 查询PerfTest任务离线报告列表 |
| `ShowMergeCaseDetail` | GET | `/v2/{project_id}/task-run-infos/{task_run_id}/case-run-infos/{case_run_id}/detail` | 查询用例报告详情 |
| `ShowMergeTaskCase` | GET | `/v2/{project_id}/task-run-infos/{task_run_id}/cases` | 查询任务报告的用例列表 |
| `ShowProcess` | GET | `/v1/{project_id}/test-suites/upload/processes` | 查询导入进度 |
| `ShowProject` | GET | `/v1/{project_id}/test-suites/{test_suite_id}` | 查询工程 |
| `ShowReport` | GET | `/v1/{project_id}/task-run-infos/{task_run_id}/case-run-infos/{case_run_id}/reports` | 查询报告 |

... and 15 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
