---
name: huaweicloud-frs
description: HuaweiCloud FRS API guide. 29 APIs covering 人脸库资源管理, 人脸搜索, 人脸检测, 人脸比对, 人脸资源管理.
---

# HuaweiCloud FRS API Guide

29 APIs. Tags: 人脸库资源管理, 人脸搜索, 人脸检测, 人脸比对, 人脸资源管理, 动作活体检测, 静默活体检测

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `AddFacesByBase64` | POST | `/v2/{project_id}/face-sets/{face_set_name}/faces` | 添加人脸 |
| `AddFacesByFile` | POST | `/v2/{project_id}/face-sets/{face_set_name}/faces` | 添加人脸 |
| `AddFacesByUrl` | POST | `/v2/{project_id}/face-sets/{face_set_name}/faces` | 添加人脸 |
| `BatchDeleteFaces` | DELETE | `/v2/{project_id}/face-sets/{face_set_name}/faces/batch` | 批量删除人脸 |
| `CompareFaceByBase64` | POST | `/v2/{project_id}/face-compare` | 人脸比对 |
| `CompareFaceByFile` | POST | `/v2/{project_id}/face-compare` | 人脸比对 |
| `CompareFaceByUrl` | POST | `/v2/{project_id}/face-compare` | 人脸比对 |
| `CreateFaceSet` | POST | `/v2/{project_id}/face-sets` | 创建人脸库 |
| `DeleteFaceByExternalImageId` | DELETE | `/v2/{project_id}/face-sets/{face_set_name}/faces` | 删除人脸 |
| `DeleteFaceByFaceId` | DELETE | `/v2/{project_id}/face-sets/{face_set_name}/faces` | 删除人脸 |
| `DeleteFaceSet` | DELETE | `/v2/{project_id}/face-sets/{face_set_name}` | 删除人脸库 |
| `DetectFaceByBase64` | POST | `/v2/{project_id}/face-detect` | 人脸检测 |
| `DetectFaceByFile` | POST | `/v2/{project_id}/face-detect` | 人脸检测 |
| `DetectFaceByUrl` | POST | `/v2/{project_id}/face-detect` | 人脸检测 |
| `DetectLiveByBase64` | POST | `/v1/{project_id}/live-detect` | 动作活体检测 |
| `DetectLiveByFile` | POST | `/v1/{project_id}/live-detect` | 动作活体检测 |
| `DetectLiveByUrl` | POST | `/v1/{project_id}/live-detect` | 动作活体检测 |
| `DetectLiveFaceByBase64` | POST | `/v1/{project_id}/live-detect-face` | 静默活体检测 |
| `DetectLiveFaceByFile` | POST | `/v1/{project_id}/live-detect-face` | 静默活体检测 |
| `DetectLiveFaceByUrl` | POST | `/v1/{project_id}/live-detect-face` | 静默活体检测 |
| `SearchFaceByBase64` | POST | `/v2/{project_id}/face-sets/{face_set_name}/search` | 人脸搜索 |
| `SearchFaceByFaceId` | POST | `/v2/{project_id}/face-sets/{face_set_name}/search` | 人脸搜索 |
| `SearchFaceByFile` | POST | `/v2/{project_id}/face-sets/{face_set_name}/search` | 人脸搜索 |
| `SearchFaceByUrl` | POST | `/v2/{project_id}/face-sets/{face_set_name}/search` | 人脸搜索 |
| `ShowAllFaceSets` | GET | `/v2/{project_id}/face-sets` | 查询所有人脸库 |
| `ShowFacesByFaceId` | GET | `/v2/{project_id}/face-sets/{face_set_name}/faces` | 查询人脸 |
| `ShowFacesByLimit` | GET | `/v2/{project_id}/face-sets/{face_set_name}/faces` | 查询人脸 |
| `ShowFaceSet` | GET | `/v2/{project_id}/face-sets/{face_set_name}` | 查询人脸库 |
| `UpdateFace` | PUT | `/v2/{project_id}/face-sets/{face_set_name}/faces` | 更新人脸 |

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
