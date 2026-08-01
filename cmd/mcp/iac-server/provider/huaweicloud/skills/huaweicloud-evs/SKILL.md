---
name: huaweicloud-evs
description: HuaweiCloud EVS API guide. 38 APIs covering API版本信息查询, 云硬盘, 云硬盘快照, 云硬盘标签, 云硬盘过户. Detailed swagger definitions in references/<APIName>.json.
---

# HuaweiCloud EVS API Guide

38 APIs. Tags: API版本信息查询, 云硬盘, 云硬盘快照, 云硬盘标签, 云硬盘过户, 其他, 回收站管理

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `BatchCreateVolumeTags` | POST | `/v2/{project_id}/cloudvolumes/{volume_id}/tags/action` | 为指定云硬盘批量添加标签 |
| `BatchDeleteVolumeTags` | POST | `/v2/{project_id}/cloudvolumes/{volume_id}/tags/action` | 为指定云硬盘批量删除标签 |
| `BatchResizeVolumes` | POST | `/v5/{project_id}/volumes/batch-extend` | 批量扩容云硬盘 |
| `CinderAcceptVolumeTransfer` | POST | `/v2/{project_id}/os-volume-transfer/{transfer_id}/accept` | 接受云硬盘过户 |
| `CinderCreateVolumeTransfer` | POST | `/v2/{project_id}/os-volume-transfer` | 创建云硬盘过户 |
| `CinderDeleteVolumeTransfer` | DELETE | `/v2/{project_id}/os-volume-transfer/{transfer_id}` | 删除云硬盘过户 |
| `CinderListAvailabilityZones` | GET | `/v2/{project_id}/os-availability-zone` | 查询所有的可用分区信息 |
| `CinderListQuotas` | GET | `/v2/{project_id}/os-quota-sets/{target_project_id}` | 查询租户的详细配额 |
| `CinderListVolumeTransfers` | GET | `/v2/{project_id}/os-volume-transfer` | 查询云硬盘过户记录列表概要 |
| `CinderListVolumeTypes` | GET | `/v2/{project_id}/types` | 查询云硬盘类型列表 |
| `CinderShowVolumeTransfer` | GET | `/v2/{project_id}/os-volume-transfer/{transfer_id}` | 查询单个云硬盘过户记录详情 |
| `CreateSnapshot` | POST | `/v2/{project_id}/cloudsnapshots` | 创建云硬盘快照 |
| `CreateVolume` | POST | `/v2.1/{project_id}/cloudvolumes` | 创建云硬盘 |
| `DeleteSnapshot` | DELETE | `/v2/{project_id}/cloudsnapshots/{snapshot_id}` | 删除云硬盘快照 |
| `DeleteVolume` | DELETE | `/v2/{project_id}/cloudvolumes/{volume_id}` | 删除云硬盘 |
| `DeleteVolumeInRecycle` | DELETE | `/v3/{project_id}/recycle-bin-volumes/{volume_id}` | 删除回收站中单个云硬盘 |
| `ListSnapshots` | GET | `/v2/{project_id}/cloudsnapshots/detail` | 查询云硬盘快照详情列表 |
| `ListVersions` | GET | `/` | 查询接口版本信息列表 |
| `ListVolumes` | GET | `/v2/{project_id}/cloudvolumes/detail` | 查询所有云硬盘详情 |
| `ListVolumesByTags` | POST | `/v2/{project_id}/cloudvolumes/resource_instances/action` | 通过标签查询云硬盘资源实例详情 |
| `ListVolumesInRecycle` | GET | `/v3/{project_id}/recycle-bin-volumes/detail` | 查询回收站中所有云硬盘详情 |
| `ListVolumeTags` | GET | `/v2/{project_id}/cloudvolumes/tags` | 获取云硬盘资源的所有标签 |
| `ModifyVolumeQoS` | PUT | `/v5/{project_id}/cloudvolumes/{volume_id}/qos` | 修改云硬盘QoS |
| `ResizeVolume` | POST | `/v2.1/{project_id}/cloudvolumes/{volume_id}/action` | 扩容云硬盘 |
| `RetypeVolume` | POST | `/v2/{project_id}/volumes/{volume_id}/retype` | 磁盘类型变更 |
| `RevertVolumeInRecycle` | POST | `/v3/{project_id}/recycle-bin-volumes/{volume_id}/revert` | 还原回收站中单个云硬盘 |
| `RollbackSnapshot` | POST | `/v2/{project_id}/cloudsnapshots/{snapshot_id}/rollback` | 回滚快照到云硬盘 |
| `ShowJob` | GET | `/v1/{project_id}/jobs/{job_id}` | 查询job的状态 |
| `ShowRecyclePolicy` | GET | `/v3/{project_id}/recycle-bin-volumes/policy` | 查询回收站策略 |
| `ShowSnapshot` | GET | `/v2/{project_id}/cloudsnapshots/{snapshot_id}` | 查询单个云硬盘快照详情 |

... and 8 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
