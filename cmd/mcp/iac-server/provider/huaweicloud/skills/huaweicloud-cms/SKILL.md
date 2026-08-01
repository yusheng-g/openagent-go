---
name: huaweicloud-cms
description: HuaweiCloud CMS API guide. 7 APIs covering 智能购买组管理, 规格推荐管理.
---

# HuaweiCloud CMS API Guide

7 APIs. Tags: 智能购买组管理, 规格推荐管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `CreateAutoLaunchGroup` | POST | `/v2/{domain_id}/auto-launch-groups` | 创建智能购买组 |
| `DeleteAutoLaunchGroup` | DELETE | `/v2/{domain_id}/auto-launch-groups/{auto_launch_group_id}` | 删除智能购买组 |
| `ListAutoLaunchGroups` | GET | `/v2/{domain_id}/auto-launch-groups` | 查询智能购买组列表 |
| `ListInstances` | GET | `/v2/{domain_id}/auto-launch-groups/{auto_launch_group_id}/instances` | 查询智能购买组实例列表 |
| `ListSupplyRecommendation` | POST | `/v1/{domain_id}/recommendations/ecs-supply` | 地域推荐 |
| `ShowAutoLaunchGroup` | GET | `/v2/{domain_id}/auto-launch-groups/{auto_launch_group_id}` | 查询智能购买组详情 |
| `UpdateAutoLaunchGroup` | PUT | `/v2/{domain_id}/auto-launch-groups/{auto_launch_group_id}` | 修改智能购买组 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
