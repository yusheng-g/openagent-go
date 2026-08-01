---
name: huaweicloud-ims
description: HuaweiCloud IMS API guide. 51 APIs covering 查询API版本信息(OpenStack原生), 镜像, 镜像(OpenStack原生), 镜像任务, 镜像共享.
---

# HuaweiCloud IMS API Guide

51 APIs. Tags: 查询API版本信息(OpenStack原生), 镜像, 镜像(OpenStack原生), 镜像任务, 镜像共享, 镜像共享(OpenStack原生), 镜像复制, 镜像标签, 镜像标签(OpenStack原生), 镜像视图(OpenStack原生), 镜像配额

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddImageTag` | POST | `/v2/{project_id}/images/{image_id}/tags` | 添加镜像标签 |
| `BatchAddMembers` | POST | `/v1/cloudimages/members` | 批量添加镜像成员 |
| `BatchAddOrDeleteTags` | POST | `/v2/{project_id}/images/{image_id}/tags/action` | 批量添加删除镜像标签 |
| `BatchDeleteMembers` | DELETE | `/v1/cloudimages/members` | 批量删除镜像成员 |
| `BatchDeleteTags` | DELETE | `/v1/{project_id}/cloudimages/{image_id}/tags/delete` | 批量删除镜像标签 |
| `BatchUpdateMembers` | PUT | `/v1/cloudimages/members` | 批量更新镜像成员状态 |
| `CopyImageCrossRegion` | POST | `/v1/cloudimages/{image_id}/cross_region_copy` | 跨Region复制镜像 |
| `CopyImageInRegion` | POST | `/v1/cloudimages/{image_id}/copy` | Region内复制镜像 |
| `CopyImageInRegionInSafeMode` | POST | `/v2.1/cloudimages/{image_id}/copy` | Region内复制镜像(新) |
| `CreateDataImage` | POST | `/v1/cloudimages/dataimages/action` | 使用外部镜像文件制作数据镜像 |
| `CreateDataImageInSafeMode` | POST | `/v2.1/cloudimages/dataimages/action` | 使用外部镜像文件制作数据镜像(新) |
| `CreateImage` | POST | `/v2/cloudimages/action` | 制作镜像 |
| `CreateImageInSafeMode` | POST | `/v2.1/cloudimages/action` | 制作镜像(新) |
| `CreateOrUpdateTags` | PUT | `/v1/cloudimages/tags` | 增加或修改标签 |
| `CreateWholeImage` | POST | `/v1/cloudimages/wholeimages/action` | 制作整机镜像 |
| `DeleteImageTag` | DELETE | `/v2/{project_id}/images/{image_id}/tags/{key}` | 删除镜像标签 |
| `ExportImage` | POST | `/v1/cloudimages/{image_id}/file` | 导出镜像 |
| `ExportImageInSafeMode` | POST | `/v2.1/cloudimages/{image_id}/file` | 导出镜像(新) |
| `GlanceAddImageMember` | POST | `/v2/images/{image_id}/members` | 添加镜像成员(OpenStack原生) |
| `GlanceCreateImageMetadata` | POST | `/v2/images` | 创建镜像元数据(OpenStack原生) |
| `GlanceCreateTag` | PUT | `/v2/images/{image_id}/tags/{tag}` | 增加标签(OpenStack原生) |
| `GlanceDeleteImage` | DELETE | `/v2/images/{image_id}` | 删除镜像(OpenStack原生) |
| `GlanceDeleteImageMember` | DELETE | `/v2/images/{image_id}/members/{member_id}` | 删除指定的镜像成员(OpenStack原生) |
| `GlanceDeleteTag` | DELETE | `/v2/images/{image_id}/tags/{tag}` | 删除标签(OpenStack原生) |
| `GlanceListImageMembers` | GET | `/v2/images/{image_id}/members` | 获取镜像成员列表(OpenStack原生) |
| `GlanceListImageMemberSchemas` | GET | `/v2/schemas/members` | 查询镜像成员列表视图(OpenStack原生) |
| `GlanceListImages` | GET | `/v2/images` | 查询镜像列表(OpenStack原生) |
| `GlanceListImageSchemas` | GET | `/v2/schemas/images` | 查询镜像列表视图(OpenStack原生) |
| `GlanceShowImage` | GET | `/v2/images/{image_id}` | 查询镜像详情(OpenStack原生) |
| `GlanceShowImageMember` | GET | `/v2/images/{image_id}/members/{member_id}` | 获取镜像成员详情(OpenStack原生) |

... and 21 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
