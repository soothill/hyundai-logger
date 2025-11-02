variable "app_name" {
  description = "Application name"
  type        = string
  default     = "hyundai-logger"
}

variable "environment" {
  description = "Environment name (e.g., production, staging, development)"
  type        = string
  default     = "production"
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "vpc_id" {
  description = "VPC ID where resources will be created"
  type        = string
}

variable "private_subnets" {
  description = "List of private subnet IDs for ECS tasks and RDS"
  type        = list(string)
}

variable "container_image" {
  description = "Docker container image"
  type        = string
}

variable "container_cpu" {
  description = "CPU units for the container (256, 512, 1024, etc.)"
  type        = number
  default     = 256
}

variable "container_memory" {
  description = "Memory for the container in MB (512, 1024, 2048, etc.)"
  type        = number
  default     = 512
}

variable "desired_count" {
  description = "Desired number of ECS tasks"
  type        = number
  default     = 1
}

variable "hyundai_username" {
  description = "Hyundai Bluelink username"
  type        = string
  sensitive   = true
}

variable "hyundai_password" {
  description = "Hyundai Bluelink password"
  type        = string
  sensitive   = true
}

variable "hyundai_pin" {
  description = "Hyundai Bluelink PIN"
  type        = string
  sensitive   = true
}

variable "hyundai_region" {
  description = "Hyundai Bluelink region (na, eu, kr, etc.)"
  type        = string
  default     = "na"
}

variable "hyundai_brand" {
  description = "Vehicle brand (hyundai, kia, genesis)"
  type        = string
  default     = "hyundai"
}

variable "poll_interval_minutes" {
  description = "Polling interval in minutes"
  type        = number
  default     = 5
}

variable "influxdb_enabled" {
  description = "Whether to create an InfluxDB RDS instance"
  type        = bool
  default     = true
}

variable "influxdb_url" {
  description = "External InfluxDB URL (used when influxdb_enabled is false)"
  type        = string
  default     = ""
}

variable "influxdb_token" {
  description = "InfluxDB authentication token"
  type        = string
  sensitive   = true
}

variable "influxdb_organization" {
  description = "InfluxDB organization"
  type        = string
  default     = "hyundai"
}

variable "influxdb_bucket" {
  description = "InfluxDB bucket"
  type        = string
  default     = "vehicle-data"
}

variable "influxdb_instance_class" {
  description = "RDS instance class for InfluxDB"
  type        = string
  default     = "db.t3.micro"
}

variable "influxdb_storage_gb" {
  description = "Storage size in GB for InfluxDB"
  type        = number
  default     = 20
}

variable "influxdb_password" {
  description = "Password for InfluxDB admin user"
  type        = string
  sensitive   = true
  default     = ""
}

variable "log_retention_days" {
  description = "CloudWatch log retention in days"
  type        = number
  default     = 30
}

variable "backup_retention_days" {
  description = "RDS backup retention in days"
  type        = number
  default     = 7
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
