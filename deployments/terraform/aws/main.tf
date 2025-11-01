terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# ECS Cluster
resource "aws_ecs_cluster" "main" {
  name = "${var.app_name}-${var.environment}"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }

  tags = merge(
    var.tags,
    {
      Name        = "${var.app_name}-${var.environment}"
      Environment = var.environment
    }
  )
}

# CloudWatch Log Group
resource "aws_cloudwatch_log_group" "app" {
  name              = "/ecs/${var.app_name}-${var.environment}"
  retention_in_days = var.log_retention_days

  tags = var.tags
}

# ECS Task Definition
resource "aws_ecs_task_definition" "app" {
  family                   = "${var.app_name}-${var.environment}"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.container_cpu
  memory                   = var.container_memory
  execution_role_arn       = aws_iam_role.ecs_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name  = var.app_name
      image = var.container_image

      essential = true

      environment = [
        {
          name  = "HYUNDAI_REGION"
          value = var.hyundai_region
        },
        {
          name  = "HYUNDAI_BRAND"
          value = var.hyundai_brand
        },
        {
          name  = "INFLUXDB_URL"
          value = var.influxdb_enabled ? "http://${aws_db_instance.influxdb[0].address}:8086" : var.influxdb_url
        },
        {
          name  = "INFLUXDB_ORG"
          value = var.influxdb_organization
        },
        {
          name  = "INFLUXDB_BUCKET"
          value = var.influxdb_bucket
        },
        {
          name  = "POLL_INTERVAL"
          value = tostring(var.poll_interval_minutes)
        }
      ]

      secrets = [
        {
          name      = "HYUNDAI_USERNAME"
          valueFrom = aws_secretsmanager_secret.hyundai_credentials.arn
        },
        {
          name      = "HYUNDAI_PASSWORD"
          valueFrom = aws_secretsmanager_secret.hyundai_credentials.arn
        },
        {
          name      = "HYUNDAI_PIN"
          valueFrom = aws_secretsmanager_secret.hyundai_credentials.arn
        },
        {
          name      = "INFLUXDB_TOKEN"
          valueFrom = aws_secretsmanager_secret.influxdb_token.arn
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.app.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }

      healthCheck = {
        command     = ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"]
        interval    = 30
        timeout     = 5
        retries     = 3
        startPeriod = 60
      }
    }
  ])

  tags = var.tags
}

# ECS Service
resource "aws_ecs_service" "app" {
  name            = "${var.app_name}-${var.environment}"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.app.arn
  desired_count   = var.desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.private_subnets
    security_groups  = [aws_security_group.app.id]
    assign_public_ip = false
  }

  depends_on = [
    aws_iam_role_policy_attachment.ecs_execution
  ]

  tags = var.tags
}

# Security Group for Application
resource "aws_security_group" "app" {
  name        = "${var.app_name}-${var.environment}-app"
  description = "Security group for ${var.app_name} application"
  vpc_id      = var.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(
    var.tags,
    {
      Name = "${var.app_name}-${var.environment}-app"
    }
  )
}

# Secrets Manager for Hyundai Credentials
resource "aws_secretsmanager_secret" "hyundai_credentials" {
  name        = "${var.app_name}-${var.environment}-hyundai-credentials"
  description = "Hyundai Bluelink credentials"

  tags = var.tags
}

resource "aws_secretsmanager_secret_version" "hyundai_credentials" {
  secret_id = aws_secretsmanager_secret.hyundai_credentials.id
  secret_string = jsonencode({
    username = var.hyundai_username
    password = var.hyundai_password
    pin      = var.hyundai_pin
  })
}

# Secrets Manager for InfluxDB Token
resource "aws_secretsmanager_secret" "influxdb_token" {
  name        = "${var.app_name}-${var.environment}-influxdb-token"
  description = "InfluxDB authentication token"

  tags = var.tags
}

resource "aws_secretsmanager_secret_version" "influxdb_token" {
  secret_id     = aws_secretsmanager_secret.influxdb_token.id
  secret_string = var.influxdb_token
}

# IAM Role for ECS Task Execution
resource "aws_iam_role" "ecs_execution" {
  name = "${var.app_name}-${var.environment}-ecs-execution"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })

  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "ecs_execution" {
  role       = aws_iam_role.ecs_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# IAM Policy for Secrets Access
resource "aws_iam_role_policy" "secrets_access" {
  name = "${var.app_name}-${var.environment}-secrets-access"
  role = aws_iam_role.ecs_execution.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue"
        ]
        Resource = [
          aws_secretsmanager_secret.hyundai_credentials.arn,
          aws_secretsmanager_secret.influxdb_token.arn
        ]
      }
    ]
  })
}

# IAM Role for ECS Task
resource "aws_iam_role" "ecs_task" {
  name = "${var.app_name}-${var.environment}-ecs-task"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })

  tags = var.tags
}

# RDS Instance for InfluxDB (optional)
resource "aws_db_instance" "influxdb" {
  count = var.influxdb_enabled ? 1 : 0

  identifier        = "${var.app_name}-${var.environment}-influxdb"
  engine            = "postgres"  # Using PostgreSQL as placeholder for InfluxDB
  engine_version    = "14"
  instance_class    = var.influxdb_instance_class
  allocated_storage = var.influxdb_storage_gb

  db_name  = "influxdb"
  username = "influxadmin"
  password = var.influxdb_password

  vpc_security_group_ids = [aws_security_group.influxdb[0].id]
  db_subnet_group_name   = aws_db_subnet_group.influxdb[0].name

  backup_retention_period = var.backup_retention_days
  skip_final_snapshot     = var.environment != "production"
  final_snapshot_identifier = var.environment == "production" ? "${var.app_name}-${var.environment}-final-snapshot" : null

  storage_encrypted = true

  tags = var.tags
}

# DB Subnet Group
resource "aws_db_subnet_group" "influxdb" {
  count = var.influxdb_enabled ? 1 : 0

  name       = "${var.app_name}-${var.environment}-influxdb"
  subnet_ids = var.private_subnets

  tags = var.tags
}

# Security Group for InfluxDB
resource "aws_security_group" "influxdb" {
  count = var.influxdb_enabled ? 1 : 0

  name        = "${var.app_name}-${var.environment}-influxdb"
  description = "Security group for InfluxDB"
  vpc_id      = var.vpc_id

  ingress {
    from_port       = 8086
    to_port         = 8086
    protocol        = "tcp"
    security_groups = [aws_security_group.app.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(
    var.tags,
    {
      Name = "${var.app_name}-${var.environment}-influxdb"
    }
  )
}
