---
name: terraform-modules
description: Terraform module templates available in the workspace — read, fill, write
---

# Terraform Module Catalog

Templates are in the `templates/` directory. Use `read_file` to read each template, replace placeholders with actual values, then `write_file` to save the result to `terraform/`.

## Available Templates

| File | Module | Creates |
|------|--------|---------|
| templates/ecs.tf.tmpl | ECS | VM instance with optional public IP, user_data bootstrap |
| templates/rds.tf.tmpl | RDS | PostgreSQL/MySQL managed database, backup config |
| templates/obs.tf.tmpl | OBS | Object storage bucket with optional versioning and website |
| templates/cdn.tf.tmpl | CDN | CDN domain with HTTPS and origin config |
| templates/provider.tf.tmpl | Provider | HuaweiCloud provider + backend + shared variables |

## Template Variables

### ecs.tf.tmpl
- `Name` — resource name, e.g. `web_server`
- `ProjectName`, `Environment` — tagging
- `Flavor` — ECS spec, e.g. `s6.large.2` (2C4G)
- `DiskSize` — GB (40 typical)
- `PublicIP` — true/false
- `Bandwidth` — Mbps (5 for web)
- `UserData` — bootstrap shell script

### rds.tf.tmpl
- `Name` — e.g. `postgres_db`
- `ProjectName`, `Environment` — tagging
- `Flavor` — `rds.pg.c2.medium` (4GB) or `rds.pg.c2.large` (8GB)
- `Engine` — PostgreSQL or MySQL
- `Version` — 14, 15, 8.0
- `Port` — 5432 (PG) or 3306 (MySQL)
- `VolumeType`, `VolumeSize` — SSD, 100 (GB)
- `HAMode` — async (single) or semisync (HA)
- `BackupDays` — 7

### obs.tf.tmpl
- `Name`, `ProjectName`, `Environment`
- `BucketName` — globally unique name
- `ACL` — private, public-read
- `Versioning` — true/false
- `Website` — true/false

### cdn.tf.tmpl
- `Name`, `ProjectName`, `Environment`
- `Domain` — CDN domain name
- `ServiceType` — web, download
- `Origin`, `OriginType`, `OriginProtocol`
- `HTTPSStatus`, `CertName`

### provider.tf.tmpl
- Shared: `region`, `access_key`, `secret_key`, `project_name`, `environment`

## Workflow

1. Read `templates/provider.tf.tmpl` with `read_file`
2. Replace placeholders with values
3. Write to `terraform/provider.tf` with `write_file`
4. Repeat for each service template (ecs, rds, obs, cdn)
5. Run `terraform_init` to validate
6. Hand off to reviewer for `terraform_plan`
