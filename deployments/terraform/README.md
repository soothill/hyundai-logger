# Hyundai Logger Terraform Modules

Terraform modules for deploying Hyundai Logger infrastructure.

## Modules

- `aws` - Deploy to AWS ECS/Fargate with RDS for InfluxDB
- `gcp` - Deploy to Google Cloud Run with Cloud SQL
- `azure` - Deploy to Azure Container Instances with Azure Database

## Prerequisites

- Terraform 1.0+
- Cloud provider credentials configured
- Docker image published to container registry

## Usage

### AWS Deployment

```hcl
module "hyundai_logger" {
  source = "./deployments/terraform/aws"

  app_name          = "hyundai-logger"
  environment       = "production"
  aws_region        = "us-east-1"

  # Container configuration
  container_image   = "your-registry/hyundai-logger:latest"
  container_cpu     = 256
  container_memory  = 512

  # Hyundai credentials (use secrets manager in production)
  hyundai_username  = var.hyundai_username
  hyundai_password  = var.hyundai_password
  hyundai_pin       = var.hyundai_pin
  hyundai_region    = "na"

  # Networking
  vpc_id            = module.vpc.vpc_id
  private_subnets   = module.vpc.private_subnets

  # InfluxDB
  influxdb_enabled  = true
  influxdb_instance_class = "db.t3.micro"
}
```

### GCP Deployment

```hcl
module "hyundai_logger" {
  source = "./deployments/terraform/gcp"

  project_id        = "my-project"
  region            = "us-central1"
  app_name          = "hyundai-logger"

  container_image   = "gcr.io/my-project/hyundai-logger:latest"

  # Configuration
  hyundai_username  = var.hyundai_username
  hyundai_password  = var.hyundai_password
  hyundai_pin       = var.hyundai_pin
}
```

### Azure Deployment

```hcl
module "hyundai_logger" {
  source = "./deployments/terraform/azure"

  resource_group_name = "hyundai-logger-rg"
  location            = "East US"
  app_name            = "hyundai-logger"

  container_image     = "your-registry.azurecr.io/hyundai-logger:latest"

  # Configuration
  hyundai_username    = var.hyundai_username
  hyundai_password    = var.hyundai_password
  hyundai_pin         = var.hyundai_pin
}
```

## Security Best Practices

1. **Never commit credentials** - Use secret management services:
   - AWS: Secrets Manager or Parameter Store
   - GCP: Secret Manager
   - Azure: Key Vault

2. **Use variables** - Store sensitive data in `terraform.tfvars` (add to .gitignore)

3. **Enable encryption** - All modules enable encryption at rest by default

4. **Network security** - Deploy in private subnets with proper security groups

## State Management

Store Terraform state remotely:

```hcl
terraform {
  backend "s3" {
    bucket = "my-terraform-state"
    key    = "hyundai-logger/terraform.tfstate"
    region = "us-east-1"
  }
}
```

## Cost Estimation

Run `terraform plan` to see estimated costs before applying:

```bash
terraform plan -out=tfplan
terraform show -json tfplan | infracost breakdown --path=-
```
