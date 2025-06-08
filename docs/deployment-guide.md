# Deployment Guide

This guide covers the AWS deployment infrastructure and procedures for the Generative API Router service deployed as `go-generative-api-router`.

## 🏗️ Infrastructure Overview

### AWS Account & Region
- **AWS Account**: `${AWS_ACCOUNT_ID}`
- **Region**: `ap-southeast-3` (Asia Pacific - Jakarta)
- **Service Name**: `go-generative-api-router`

### Architecture
The service is deployed using a modern AWS serverless architecture:
- **CodeBuild** → **ECR** → **ECS Fargate**
- Separate environments for development and production
- Automated CI/CD pipeline triggered by Git commits/tags

## 🌍 Environments

### Development Environment
- **CodeBuild Project**: `dev-${SERVICE_NAME}`
- **ECR Repository**: `dev-${SERVICE_NAME}`
- **ECS Cluster**: `dev-${SERVICE_NAME}`
- **ECS Service**: `dev-${SERVICE_NAME}`
- **Trigger**: Commits to main branch
- **Source**: Latest commit (e.g., `0fa92ae`)

### Production Environment
- **CodeBuild Project**: `prod-${SERVICE_NAME}`
- **ECR Repository**: `prod-${SERVICE_NAME}`
- **ECS Cluster**: `prod-${SERVICE_NAME}`
- **ECS Service**: `prod-${SERVICE_NAME}`
- **Trigger**: Git tags (releases)
- **Source**: Release tags (e.g., `v2.0.1`)

## 🔄 Deployment Pipeline

### Pipeline Flow
```
Git Commit/Tag → CodeBuild → Docker Build → ECR Push → ECS Deploy
```

### Typical Deployment Timeline
| Stage | Development | Production |
|-------|-------------|------------|
| **CodeBuild** | ~1 minute | ~1 minute |
| **ECR Push** | ~30 seconds | ~30 seconds |
| **ECS Deploy** | ~18 seconds | ~6 seconds |
| **Total** | ~2 minutes | ~2 minutes |

### Container Specifications
- **Base Image**: Go Alpine-based container
- **Size**: ~16.4 MB
- **Architecture**: ARM64 (Graviton2)
- **Tag Strategy**: `latest` for both environments

### Environment Variables
The deployment pipeline automatically sets the following environment variables:

| Variable | Description | Development | Production |
|----------|-------------|-------------|------------|
| `VERSION` | Service version identifier | Git commit hash | Git tag version |
| `ENVIRONMENT` | Deployment environment | `development` | `production` |
| `LOG_LEVEL` | Logging level | `debug` | `info` |

**VERSION Variable:**
- **Development**: Set to the git commit hash (e.g., `0fa92ae`)
- **Production**: Set to the git tag version (e.g., `v2.0.1`)
- **Default**: `"unknown"` if not set
- **Usage**: Displayed in `/health` endpoint under `details.version`

## 📊 Monitoring & Health Checks

### Service Status Commands

#### Check CodeBuild Projects
```bash
# List all projects
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 codebuild list-projects

# Check recent builds for dev
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 codebuild list-builds-for-project \
  --project-name dev-${SERVICE_NAME} --sort-order DESCENDING

# Check recent builds for prod
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 codebuild list-builds-for-project \
  --project-name prod-${SERVICE_NAME} --sort-order DESCENDING
```

#### Check ECR Repositories
```bash
# List repositories
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecr describe-repositories \
  --query 'repositories[*].{Name:repositoryName,URI:repositoryUri,CreatedAt:createdAt}' \
  --output table

# Check recent images in dev
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecr describe-images \
  --repository-name dev-${SERVICE_NAME} \
  --query 'imageDetails[*].{Tags:imageTags,Digest:imageDigest,PushedAt:imagePushedAt}' \
  --output table

# Check recent images in prod
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecr describe-images \
  --repository-name prod-${SERVICE_NAME} \
  --query 'imageDetails[*].{Tags:imageTags,Digest:imageDigest,PushedAt:imagePushedAt}' \
  --output table
```

#### Check ECS Services
```bash
# List clusters
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs list-clusters

# Check dev service status
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs describe-services \
  --cluster dev-${SERVICE_NAME} --services dev-${SERVICE_NAME} \
  --query 'services[0].{ServiceName:serviceName,Status:status,RunningCount:runningCount,PendingCount:pendingCount,DesiredCount:desiredCount,TaskDefinition:taskDefinition}'

# Check prod service status
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs describe-services \
  --cluster prod-${SERVICE_NAME} --services prod-${SERVICE_NAME} \
  --query 'services[0].{ServiceName:serviceName,Status:status,RunningCount:runningCount,PendingCount:pendingCount,DesiredCount:desiredCount,TaskDefinition:taskDefinition}'
```

### Health Check Endpoints
```bash
# Development environment health check
curl -f https://dev-genapi.example.com/health

# Production environment health check  
curl -f https://genapi.example.com/health

# Check with timeout
curl --max-time 5 https://genapi.example.com/health
```

## 🚀 Deployment Procedures

### Development Deployment
Development deployments are **automatic** when commits are pushed to the main branch:

1. **Commit Changes**:
   ```bash
   git add .
   git commit -m "feat: your changes"
   git push origin main
   ```

2. **Monitor Deployment**:
   ```bash
   # Check build status
   aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 codebuild list-builds-for-project \
     --project-name dev-${SERVICE_NAME} --sort-order DESCENDING | head -5
   
   # Check service status
   aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs describe-services \
     --cluster dev-${SERVICE_NAME} --services dev-${SERVICE_NAME}
   ```

### Production Deployment
Production deployments are **automatic** when release tags are created:

1. **Create Release** (following [release process](../.cursor/rules/development_guide.mdc#release-process)):
   ```bash
   # Create and push tag
   git tag -a v2.0.2 -m "Release v2.0.2: Description"
   git push origin v2.0.2
   ```

2. **Monitor Deployment**:
   ```bash
   # Check build status
   aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 codebuild list-builds-for-project \
     --project-name prod-${SERVICE_NAME} --sort-order DESCENDING | head -5
   
   # Check service status
   aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs describe-services \
     --cluster prod-${SERVICE_NAME} --services prod-${SERVICE_NAME}
   ```

## 🔍 Troubleshooting

### Common Issues

#### 1. Build Failures
```bash
# Get detailed build information
BUILD_ID="prod-${SERVICE_NAME}:latest-build-id"
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 codebuild batch-get-builds \
  --ids $BUILD_ID --query 'builds[0].{Status:buildStatus,Phase:currentPhase,Logs:logs.groupName}'

# View build logs
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 logs get-log-events \
  --log-group-name /aws/codebuild/prod-${SERVICE_NAME} \
  --log-stream-name latest-stream
```

#### 2. Service Not Starting
```bash
# Check task definition
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs describe-task-definition \
  --task-definition prod-${SERVICE_NAME}:latest

# Check service events
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs describe-services \
  --cluster prod-${SERVICE_NAME} --services prod-${SERVICE_NAME} \
  --query 'services[0].events[0:5]'

# Check CloudWatch logs
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 logs describe-log-groups \
  --log-group-name-prefix /ecs/prod-${SERVICE_NAME}
```

#### 3. Health Check Failures
```bash
# Check service health
curl -v https://genapi.example.com/health

# Check ECS task health
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs list-tasks \
  --cluster prod-${SERVICE_NAME} --service-name prod-${SERVICE_NAME}

# Get task details
TASK_ARN="arn:aws:ecs:ap-southeast-3:${AWS_ACCOUNT_ID}:task/prod-${SERVICE_NAME}/task-id"
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs describe-tasks \
  --cluster prod-${SERVICE_NAME} --tasks $TASK_ARN
```

### Emergency Procedures

#### Rollback Production
```bash
# List recent task definitions
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs list-task-definitions \
  --family-prefix prod-${SERVICE_NAME} --sort DESC

# Update service to previous task definition
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs update-service \
  --cluster prod-${SERVICE_NAME} \
  --service prod-${SERVICE_NAME} \
  --task-definition prod-${SERVICE_NAME}:previous-version
```

#### Scale Service
```bash
# Scale up for high load
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs update-service \
  --cluster prod-${SERVICE_NAME} \
  --service prod-${SERVICE_NAME} \
  --desired-count 3

# Scale down for maintenance
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs update-service \
  --cluster prod-${SERVICE_NAME} \
  --service prod-${SERVICE_NAME} \
  --desired-count 0
```

## 📈 Performance Monitoring

### CloudWatch Metrics
Key metrics to monitor:
- **ECS Service**: CPU utilization, memory utilization, task count
- **Application Load Balancer**: Request count, response time, error rate
- **CodeBuild**: Build duration, success rate

### Monitoring Commands
```bash
# Get ECS service metrics
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 cloudwatch get-metric-statistics \
  --namespace AWS/ECS \
  --metric-name CPUUtilization \
  --dimensions Name=ServiceName,Value=prod-${SERVICE_NAME} Name=ClusterName,Value=prod-${SERVICE_NAME} \
  --start-time 2025-01-01T00:00:00Z \
  --end-time 2025-01-01T23:59:59Z \
  --period 3600 \
  --statistics Average

# Check recent deployments
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs describe-services \
  --cluster prod-${SERVICE_NAME} --services prod-${SERVICE_NAME} \
  --query 'services[0].deployments[*].{Status:status,CreatedAt:createdAt,TaskDefinition:taskDefinition}'
```

## 🔐 Security & Access

### Required Permissions
To manage deployments, you need:
- **CodeBuild**: Read/write access to projects
- **ECR**: Read/write access to repositories
- **ECS**: Read/write access to clusters and services
- **CloudWatch**: Read access to logs and metrics

### Access Management
```bash
# Verify AWS credentials
aws --profile ${AWS_ACCOUNT_ID} sts get-caller-identity

# Check permissions
aws --profile ${AWS_ACCOUNT_ID} iam get-user
```

## 📚 Additional Resources

- **[Development Guide](development-guide.md)** - Local development setup
- **[Release Process](../.cursor/rules/development_guide.mdc#release-process)** - Creating releases
- **[AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)** - Official AWS documentation
- **[AWS CodeBuild Documentation](https://docs.aws.amazon.com/codebuild/)** - CI/CD pipeline documentation

---

**Need Help?** Contact the infrastructure team or check the [troubleshooting section](../.cursor/rules/running_and_testing.mdc#troubleshooting) for common issues.