---
name: huaweicloud-codeartscheck
description: HuaweiCloud CodeArtsCheck API guide. 29 APIs covering 任务管理, 缺陷管理, 规则管理.
---

# HuaweiCloud CodeArtsCheck API Guide

29 APIs. Tags: 任务管理, 缺陷管理, 规则管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CheckParameters` | GET | `/v2/{project_id}/tasks/{task_id}/ruleset/{ruleset_id}/check-parameters` | 查询任务规则集的检查参数 |
| `CheckRecord` | GET | `/v2/{project_id}/tasks/{task_id}/checkrecord` | 历史扫描结果查询 |
| `CheckRulesetParameters` | GET | `/v3/{project_id}/tasks/{task_id}/ruleset/{ruleset_id}/check-parameters` | 查询任务规则集的检查参数 |
| `CreateRuleset` | POST | `/v2/ruleset` | 创建自定义规则集 |
| `CreateTask` | POST | `/v2/{project_id}/task` | 新建检查任务 |
| `DeleteRuleset` | DELETE | `/v2/{project_id}/ruleset/{ruleset_id}` | 删除自定义规则集 |
| `DeleteTask` | DELETE | `/v2/tasks/{task_id}` | 删除检查任务 |
| `ListRules` | GET | `/v2/rules` | 获取规则列表接口 |
| `ListRulesets` | GET | `/v2/{project_id}/rulesets` | 查询规则集列表 |
| `ListTaskParameter` | POST | `/v2/{project_id}/tasks/{task_id}/config-parameters` | 任务配置检查参数 |
| `ListTaskRuleset` | GET | `/v2/{project_id}/tasks/{task_id}/rulesets` | 查询任务的已选规则集列表 |
| `ListTemplateRules` | GET | `/v2/{project_id}/ruleset/{ruleset_id}/rules` | 查看规则集的规则列表 |
| `RunTask` | POST | `/v2/tasks/{task_id}/run` | 执行检查任务 |
| `SetDefaulTemplate` | POST | `/v2/{project_id}/ruleset/{ruleset_id}/{language}/default` | 设置每个项目对应语言的默认规则集配置 |
| `ShowProgressDetail` | GET | `/v2/tasks/{task_id}/progress` | 查询任务执行状态 |
| `ShowTaskCmetrics` | GET | `/v2/{project_id}/tasks/{task_id}/metrics-summary` | 查询cmertrics缺陷概要 |
| `ShowTaskDefects` | GET | `/v2/tasks/{task_id}/defects-detail` | 查询缺陷详情 |
| `ShowTaskDefectsStatistic` | GET | `/v2/tasks/{task_id}/defects-statistic` | 查询缺陷详情的统计 |
| `ShowTaskDetail` | GET | `/v2/tasks/{task_id}/defects-summary` | 查询缺陷概要 |
| `ShowTaskListByProjectId` | GET | `/v2/{project_id}/tasks` | 查询任务列表 |
| `ShowTasklog` | GET | `/v2/{project_id}/tasks/{task_id}/log-detail` | 查询任务检查失败日志 |
| `ShowTaskPathTree` | GET | `/v2/{project_id}/tasks/{task_id}/listpathtree` | 获取任务的目录树 |
| `ShowTaskSettings` | GET | `/v2/{project_id}/tasks/{task_id}/settings` | 查询任务的高级选项 |
| `ShowTasksRulesets` | GET | `/v3/{project_id}/tasks/{task_id}/rulesets` | 查询任务的已选规则集列表 |
| `StopTaskById` | POST | `/v2/tasks/{task_id}/stop` | 终止检查任务 |
| `UpdateDefectStatus` | PUT | `/v2/tasks/{task_id}/defect-status` | 修改缺陷状态 |
| `UpdateIgnorePath` | POST | `/v2/{project_id}/tasks/{task_id}/config-ignorepath` | 任务配置屏蔽目录 |
| `UpdateTaskRuleset` | PUT | `/v2/tasks/{task_id}/ruleset` | 修改任务规则集 |
| `UpdateTaskSettings` | POST | `/v2/{project_id}/tasks/{task_id}/settings` | 任务配置高级选项 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
