---
name: huaweicloud-cci
description: HuaweiCloud CCI API guide. 190 APIs covering API groups, ClusterRole, ConfigMap, CronJob, Deployment.
---

# HuaweiCloud CCI API Guide

190 APIs. Tags: API groups, ClusterRole, ConfigMap, CronJob, Deployment, EIPPool, Endpoint, Event, FeatureGate, HorizontalPodAutoscaler, ImageSnapshot, Ingress, Job, Metrics, Namespace, Network, OpenAPIv2, PersistentVolumeClaim, Pod, ReplicaSet, ResourceQuota, RoleBinding, Secret, Service, StatefulSet, StorageClass, VolcanoJob

## Key APIs

| API | Method | URI | Description |
|---|---|---|---|
| `connectCoreV1GetNamespacedPodExec` | GET | `/api/v1/namespaces/{namespace}/pods/{name}/exec` | 进入容器执行命令 |
| `connectCoreV1PostNamespacedPodExec` | POST | `/api/v1/namespaces/{namespace}/pods/{name}/exec` | 进入容器执行命令 |
| `createAppsV1NamespacedDeployment` | POST | `/apis/apps/v1/namespaces/{namespace}/deployments` | 创建Deployment |
| `createAppsV1NamespacedStatefulSet` | POST | `/apis/apps/v1/namespaces/{namespace}/statefulsets` | 创建StatefulSet |
| `createBatchV1NamespacedJob` | POST | `/apis/batch/v1/namespaces/{namespace}/jobs` | 创建Job |
| `createBatchVolcanoShV1alpha1NamespacedJob` | POST | `/apis/batch.volcano.sh/v1alpha1/namespaces/{namespace}/jobs` | 创建Volcano Job |
| `createCoreV1Namespace` | POST | `/api/v1/namespaces` | 创建Namespace |
| `createCoreV1NamespacedConfigMap` | POST | `/api/v1/namespaces/{namespace}/configmaps` | 创建ConfigMap |
| `createCoreV1NamespacedEndpoints` | POST | `/api/v1/namespaces/{namespace}/endpoints` | 创建Endpoint |
| `createCoreV1NamespacedPersistentVolumeClaim` | POST | `/api/v1/namespaces/{namespace}/persistentvolumeclaims` | 创建PersistentVolumeClaim |
| `createCoreV1NamespacedPod` | POST | `/api/v1/namespaces/{namespace}/pods` | 创建Pod |
| `createCoreV1NamespacedSecret` | POST | `/api/v1/namespaces/{namespace}/secrets` | 创建Secret |
| `createCoreV1NamespacedService` | POST | `/api/v1/namespaces/{namespace}/services` | 创建Service |
| `createCrdYangtseCniV1NamespacedEIPPool` | POST | `/apis/crd.yangtse.cni/v1/namespaces/{namespace}/eippools` | 创建EIPPool |
| `createExtensionsV1beta1NamespacedIngress` | POST | `/apis/extensions/v1beta1/namespaces/{namespace}/ingresses` | 创建Ingress |
| `createImageSnapshot` | POST | `/apis/cci/v2/imagesnapshots` | 创建ImageSnapshot |
| `createNamespace` | POST | `/apis/cci/v2/namespaces` | 创建Namespace |
| `createNamespacedConfigMap` | POST | `/apis/cci/v2/namespaces/{namespace}/configmaps` | 创建ConfigMap |
| `createNamespacedDeployment` | POST | `/apis/cci/v2/namespaces/{namespace}/deployments` | 创建Deployment |
| `createNamespacedHorizontalPodAutoscaler` | POST | `/apis/cci/v2/namespaces/{namespace}/horizontalpodautoscalers` | 创建HorizontalPodAutoscaler |
| `createNamespacedNetwork` | POST | `/apis/yangtse/v2/namespaces/{namespace}/networks` | 创建Network |
| `createNamespacedPod` | POST | `/apis/cci/v2/namespaces/{namespace}/pods` | 创建Pod |
| `createNamespacedSecret` | POST | `/apis/cci/v2/namespaces/{namespace}/secrets` | 创建Secret |
| `createNamespacedService` | POST | `/apis/cci/v2/namespaces/{namespace}/services` | 创建Service |
| `createNetworkingCciIoV1beta1NamespacedNetwork` | POST | `/apis/networking.cci.io/v1beta1/namespaces/{namespace}/networks` | 创建Network |
| `createRbacAuthorizationV1NamespacedRoleBinding` | POST | `/apis/rbac.authorization.k8s.io/v1/namespaces/{namespace}/rolebindings` | 创建RoleBinding |
| `deleteAppsV1CollectionNamespacedDeployment` | DELETE | `/apis/apps/v1/namespaces/{namespace}/deployments` | 删除指定namespace下Deployments |
| `deleteAppsV1CollectionNamespacedStatefulSet` | DELETE | `/apis/apps/v1/namespaces/{namespace}/statefulsets` | 删除指定namespace下的StatefulSets |
| `deleteAppsV1NamespacedDeployment` | DELETE | `/apis/apps/v1/namespaces/{namespace}/deployments/{name}` | 删除Deployment |
| `deleteAppsV1NamespacedStatefulSet` | DELETE | `/apis/apps/v1/namespaces/{namespace}/statefulsets/{name}` | 删除StatefulSet |

... and 160 more. See `references/` for detailed swagger definitions.

## Usage

Use `http_request` to call these APIs. SDK-HMAC-SHA256 signing is automatic — do NOT pass credentials.
Use `read`/`grep`/`ls` to browse `references/` for detailed request/response schemas.
