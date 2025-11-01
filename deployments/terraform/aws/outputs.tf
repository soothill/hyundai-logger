output "ecs_cluster_name" {
  description = "Name of the ECS cluster"
  value       = aws_ecs_cluster.main.name
}

output "ecs_cluster_arn" {
  description = "ARN of the ECS cluster"
  value       = aws_ecs_cluster.main.arn
}

output "ecs_service_name" {
  description = "Name of the ECS service"
  value       = aws_ecs_service.app.name
}

output "task_definition_arn" {
  description = "ARN of the task definition"
  value       = aws_ecs_task_definition.app.arn
}

output "cloudwatch_log_group" {
  description = "CloudWatch log group name"
  value       = aws_cloudwatch_log_group.app.name
}

output "security_group_id" {
  description = "Security group ID for the application"
  value       = aws_security_group.app.id
}

output "influxdb_endpoint" {
  description = "InfluxDB RDS endpoint (if enabled)"
  value       = var.influxdb_enabled ? aws_db_instance.influxdb[0].endpoint : null
}

output "influxdb_address" {
  description = "InfluxDB RDS address (if enabled)"
  value       = var.influxdb_enabled ? aws_db_instance.influxdb[0].address : null
}

output "secrets_arn" {
  description = "ARNs of secrets in Secrets Manager"
  value = {
    hyundai_credentials = aws_secretsmanager_secret.hyundai_credentials.arn
    influxdb_token      = aws_secretsmanager_secret.influxdb_token.arn
  }
}
