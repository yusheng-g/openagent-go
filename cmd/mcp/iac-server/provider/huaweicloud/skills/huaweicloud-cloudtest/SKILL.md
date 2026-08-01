---
name: huaweicloud-cloudtest
description: HuaweiCloud CloudTest API guide. 173 APIs covering attachment-controller, test-item-controller, 拨测仪表盘信息管理, 拨测任务配置管理, 拨测告警信息管理.
---

# HuaweiCloud CloudTest API Guide

173 APIs. Tags: attachment-controller, test-item-controller, 拨测仪表盘信息管理, 拨测任务配置管理, 拨测告警信息管理, 拨测套餐状态查询, 接口测试套管理, 接口测试套餐用量管理, 接口测试管理, 查询单次测试套执行的详细结果, 查询指定表的内容, 根据任务uri查询测试任务执行历史, 测试套管理, 测试报告管理, 测试报表管理, 测试服务关联关系, 测试用例管理, 测试计划管理, 测试设计查询, 环境参数分组管理, 用例关联关系管理, 自定义测试服务接入管理, 自定义测试服务测试套件管理, 自定义测试服务用例管理, 附件管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddCaseResultFour` | POST | `/v4/{project_id}/versions/{version_uri}/testcases/{case_uri}/results` | 设置用例结果 |
| `AddFeature` | POST | `/v4/features` | 添加目录信息 |
| `AddTestCaseComment` | POST | `/GT3KServer/v4/{project_id}/testcases/{testcase_id}/comments` | 新增用例评论 |
| `AddTestCaseResultLog` | POST | `/v4/{project_id}/versions/{version_uri}/testcases/{case_uri}/results/init` | 初始化用例执行记录 |
| `BatchAddCaseResultInTask` | POST | `/v4/{project_id}/versions/{version_uri}/task/testcases/results` | 在任务下批量设置用例结果 |
| `BatchAddRelationsByOneCase` | POST | `/testrelation/v4/workitems/{workitem_id}/relations` | 添加需求/缺陷和多个用例关联关系 |
| `BatchAddResourcesForIterator` | POST | `/GT3KServer/v4/iterators/{iterator_id}/testcases/batch-add` | 向迭代中添加资源 |
| `BatchDeleteFacotrByIds` | DELETE | `/v1/{project_id}/factor` | 批量删除因子 |
| `BatchDeleteTestCase` | POST | `/v1/projects/{project_id}/testcases/batch-delete` | 批量删除自定义测试服务类型用例 |
| `BatchDeleteTestCases` | DELETE | `/GT3KServer/v4/testcases/batch-delete` | 批量删除用例 |
| `BatchDeleteTestReport` | DELETE | `/testreport/v4/{project_id}/test-reports/batch-delete` | 根据测试报告uri列表,删除测试报告 |
| `BatchRemoveTestCasesFromIterator` | DELETE | `/GT3KServer/v4/iterators/{iterator_id}/testcases/batch-delete` | 从迭代中批量移除用例 |
| `BatchShowTestCase` | POST | `/v3/{project_id}/testcases` | 批量查询用例V3 |
| `BatchUpdateTestCasesInDiffVersion` | PUT | `/v4/batch/update/testcases` | 在不同分支或者迭代下批量修改用例 |
| `BatchUpdateVersionTestCases` | PUT | `/GT3KServer/v4/{project_id}/testcases/batch-update` | 批量更新用例属性 |
| `CheckPermission` | GET | `/v1/{project_id}/permission/{id}` | 检查项目权限 |
| `CreateApiTestSuiteByRepoFile` | POST | `/v1/projects/{project_id}/repository/testsuites` | 通过导入仓库中的文件生成接口测试套 |
| `CreateAssetTree` | POST | `/v1/{project_id}/asset-tree/{asset_id}/{parent_id}` | 新增资产树节点 |
| `CreateBackupMindmap` | POST | `/v2/{project_id}/mindmap-backups/backup` | 备份脑图V2 |
| `CreateIterator` | POST | `/GT3KServer/v4/iterators` | 新增迭代 |
| `CreatePlan` | POST | `/v1/projects/{project_id}/plans` | 项目下创建计划 |
| `CreateProjectBranch` | POST | `/GT3KServer/v4/branches` | 新增分支 |
| `CreateRelationsByOneCase` | POST | `/testrelation/v4/testcases/{case_id}/relations` | 添加一个用例和多个需求/缺陷关联关系 |
| `CreateReport` | POST | `/GT3KServer/v4/{project_id}/versions/{version_id}/custom-reports` | 保存单个自定义报表 |
| `CreateResourceUri` | POST | `/GT3KServer/v4/{project_id}/resource-uri` | 生成资源URI |
| `CreateService` | POST | `/v1/services` | 新测试类型服务注册 |
| `CreateTaskDefaultResult` | POST | `/v4/{project_id}/tasks/{task_uri}/results/init` | 初始化测试任务执行记录 |
| `CreateTemplate` | POST | `/v2/{project_id}/templates` | 保存模板V2 |
| `CreateTestCase` | POST | `/v1/projects/{project_id}/testcases` | 创建自定义测试服务类型用例 |
| `CreateTestCaseInPlan` | POST | `/v1/projects/{project_id}/plans/{plan_id}/testcases/batch-add` | 计划中批量添加测试用例 |

... and 143 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
