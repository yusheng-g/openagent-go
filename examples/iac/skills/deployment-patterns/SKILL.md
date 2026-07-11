---
name: deployment-patterns
description: Cloud architecture patterns, HuaweiCloud resource catalog, and pricing reference
---

# Deployment Patterns & HuaweiCloud Reference

## Provider Configuration

```hcl
terraform {
  required_providers {
    huaweicloud = {
      source  = "huaweicloud/huaweicloud"
      version = "~> 1.50"
    }
  }
}

provider "huaweicloud" {
  region     = "cn-north-4"
  access_key = var.access_key
  secret_key = var.secret_key
}
```

Default region: `cn-north-4`. Auth via env vars: `HW_ACCESS_KEY`, `HW_SECRET_KEY`, `HW_REGION`.

## Available Services (HuaweiCloud)

| Service | Resource Prefix | Purpose |
|---------|----------------|---------|
| ECS | compute_instance | Virtual machines |
| RDS | rds_instance | Managed PostgreSQL/MySQL |
| OBS | obs_bucket | Object storage (S3-like) |
| CDN | cdn_domain | Content delivery network |
| VPC | vpc | Virtual private cloud |
| ELB | elb_loadbalancer | Load balancing |
| DCS | dcs_instance | Redis cache |
| CCE | cce_cluster | Kubernetes |

## Resource Pricing (RMB/month, approximate)

| Resource | Spec | Price |
|----------|------|-------|
| ECS | 1C1G | ~65/mo |
| ECS | 2C4G | ~170/mo |
| ECS | 4C8G | ~340/mo |
| ECS | 8C16G | ~680/mo |
| RDS PostgreSQL | 4GB, 100GB SSD | ~320/mo |
| RDS PostgreSQL HA | 4GB, 100GB SSD | ~680/mo |
| RDS MySQL | 4GB, 100GB SSD | ~280/mo |
| OBS | per GB | ~0.10/GB/mo |
| CDN | per GB | ~0.20/GB |
| ELB | shared | ~80/mo |
| Redis (DCS) | 2GB | ~150/mo |

All prices are estimates. Verify with official HuaweiCloud calculator for production use.

## Deployment Patterns

### Pattern A: Simple Web (Blog, CMS, API)

```
User → CDN → ECS (2C4G) → RDS PostgreSQL (4GB)
                          → OBS (assets)
                          → Redis (cache)
```

Resources: VPC, ECS ×1, RDS, OBS, CDN, Redis. ~620/mo.

### Pattern B: Static Site (SPA, docs)

```
User → CDN → OBS (website hosting)
```

Resources: OBS, CDN. ~20/mo.

### Pattern C: High Availability (e-commerce, SaaS)

```
User → CDN → ELB → ECS ×2 → RDS HA → Redis HA
                          → OBS (backups)
```

Resources: VPC, ECS ×2, RDS HA, OBS, CDN, ELB, Redis HA. ~1400/mo.

### Pattern D: Container (microservices, CI/CD)

```
CCE (3×4C8G) → RDS HA → Redis → OBS
```

Resources: CCE cluster, RDS HA, Redis, OBS, ELB. ~2500+/mo.

## Naming Convention

`{project}-{env}-{service}` — e.g. `myapp-dev-ecs`, `myapp-prod-rds`

## Region List

| Region | Code |
|--------|------|
| CN North-Beijing4 | cn-north-4 |
| CN East-Shanghai1 | cn-east-3 |
| CN South-Guangzhou | cn-south-1 |
