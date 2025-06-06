# Deployment Guide

This guide covers the AWS deployment infrastructure and procedures for the Generative API Router service deployed as `xyz-aduh-genapi`.

## 🏗️ Infrastructure Overview

### AWS Account & Region
- **AWS Account**: `836322468413`
- **Region**: `ap-southeast-3` (Asia Pacific - Jakarta)
- **Service Name**: `xyz-aduh-genapi`

### Architecture
The service is deployed using a modern AWS serverless architecture:
- **CodeBuild** → **ECR** → **ECS Fargate**
- Separate environments for development and production
- Automated CI/CD pipeline triggered by Git commits/tags

## 🌍 Environments

### Development Environment
- **CodeBuild Project**: `dev-xyz-aduh-genapi`
- **ECR Repository**: `dev-xyz-aduh-genapi`
- **ECS Cluster**: `dev-xyz-aduh-genapi`
- **ECS Service**: `dev-xyz-aduh-genapi`
- **Trigger**: Commits to main branch
- **Source**: Latest commit (e.g., `0fa92ae`)

### Production Environment
- **CodeBuild Project**: `prod-xyz-aduh-genapi`
- **ECR Repository**: `prod-xyz-aduh-genapi`
- **ECS Cluster**: `prod-xyz-aduh-genapi`
- **ECS Service**: `prod-xyz-aduh-genapi`
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

## 📊 Monitoring & Health Checks

### Service Status Commands

#### Check CodeBuild Projects
```bash
# List all projects
aws --profile 836322468413 --region ap-southeast-3 codebuild list-projects

# Check recent builds for dev
aws --profile 836322468413 --region ap-southeast-3 codebuild list-builds-for-project \
  --project-name dev-xyz-aduh-genapi --sort-order DESCENDING

# Check recent builds for prod
aws --profile 836322468413 --region ap-southeast-3 codebuild list-builds-for-project \
  --project-name prod-xyz-aduh-genapi --sort-order DESCENDING
```

#### Check ECR Repositories
```bash
# List repositories
aws --profile 836322468413 --region ap-southeast-3 ecr describe-repositories \
  --query 'repositories[*].{Name:repositoryName,URI:repositoryUri,CreatedAt:createdAt}' \
  --output table

# Check recent images in dev
aws --profile 836322468413 --region ap-southeast-3 ecr describe-images \
  --repository-name dev-xyz-aduh-genapi \
  --query 'imageDetails[*].{Tags:imageTags,Digest:imageDigest,PushedAt:imagePushedAt}' \
  --output table

# Check recent images in prod
aws --profile 836322468413 --region ap-southeast-3 ecr describe-images \
  --repository-name prod-xyz-aduh-genapi \
  --query 'imageDetails[*].{Tags:imageTags,Digest:imageDigest,PushedAt:imagePushedAt}' \
  --output table
```

#### Check ECS Services
```bash
# List clusters
aws --profile 836322468413 --region ap-southeast-3 ecs list-clusters

# Check dev service status
aws --profile 836322468413 --region ap-southeast-3 ecs describe-services \
  --cluster dev-xyz-aduh-genapi --services dev-xyz-aduh-genapi \
  --query 'services[0].{ServiceName:serviceName,Status:status,RunningCount:runningCount,PendingCount:pendingCount,DesiredCount:desiredCount,TaskDefinition:taskDefinition}'

# Check prod service status
aws --profile 836322468413 --region ap-southeast-3 ecs describe-services \
  --cluster prod-xyz-aduh-genapi --services prod-xyz-aduh-genapi \
  --query 'services[0].{ServiceName:serviceName,Status:status,RunningCount:runningCount,PendingCount:pendingCount,DesiredCount:desiredCount,TaskDefinition:taskDefinition}'
```

### Health Check Endpoints
```bash
# Development environment health check
curl -f https://dev-genapi.aduh.xyz/health

# Production environment health check  
curl -f https://genapi.aduh.xyz/health

# Check with timeout
curl --max-time 5 https://genapi.aduh.xyz/health
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
   aws --profile 836322468413 --region ap-southeast-3 codebuild list-builds-for-project \
     --project-name dev-xyz-aduh-genapi --sort-order DESCENDING | head -5
   
   # Check service status
   aws --profile 836322468413 --region ap-southeast-3 ecs describe-services \
     --cluster dev-xyz-aduh-genapi --services dev-xyz-aduh-genapi
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
   aws --profile 836322468413 --region ap-southeast-3 codebuild list-builds-for-project \
     --project-name prod-xyz-aduh-genapi --sort-order DESCENDING | head -5
   
   # Check service status
   aws --profile 836322468413 --region ap-southeast-3 ecs describe-services \
     --cluster prod-xyz-aduh-genapi --services prod-xyz-aduh-genapi
   ```

## 🔍 Troubleshooting

### Common Issues

#### 1. Build Failures
```bash
# Get detailed build information
BUILD_ID="prod-xyz-aduh-genapi:latest-build-id"
aws --profile 836322468413 --region ap-southeast-3 codebuild batch-get-builds \
  --ids $BUILD_ID --query 'builds[0].{Status:buildStatus,Phase:currentPhase,Logs:logs.groupName}'

# View build logs
aws --profile 836322468413 --region ap-southeast-3 logs get-log-events \
  --log-group-name /aws/codebuild/prod-xyz-aduh-genapi \
  --log-stream-name latest-stream
```

#### 2. Service Not Starting
```bash
# Check task definition
aws --profile 836322468413 --region ap-southeast-3 ecs describe-task-definition \
  --task-definition prod-xyz-aduh-genapi:latest

# Check service events
aws --profile 836322468413 --region ap-southeast-3 ecs describe-services \
  --cluster prod-xyz-aduh-genapi --services prod-xyz-aduh-genapi \
  --query 'services[0].events[0:5]'

# Check CloudWatch logs
aws --profile 836322468413 --region ap-southeast-3 logs describe-log-groups \
  --log-group-name-prefix /ecs/prod-xyz-aduh-genapi
```

#### 3. Health Check Failures
```bash
# Check service health
curl -v https://genapi.aduh.xyz/health

# Check ECS task health
aws --profile 836322468413 --region ap-southeast-3 ecs list-tasks \
  --cluster prod-xyz-aduh-genapi --service-name prod-xyz-aduh-genapi

# Get task details
TASK_ARN="arn:aws:ecs:ap-southeast-3:836322468413:task/prod-xyz-aduh-genapi/task-id"
aws --profile 836322468413 --region ap-southeast-3 ecs describe-tasks \
  --cluster prod-xyz-aduh-genapi --tasks $TASK_ARN
```

### Emergency Procedures

#### Rollback Production
```bash
# List recent task definitions
aws --profile 836322468413 --region ap-southeast-3 ecs list-task-definitions \
  --family-prefix prod-xyz-aduh-genapi --sort DESC

# Update service to previous task definition
aws --profile 836322468413 --region ap-southeast-3 ecs update-service \
  --cluster prod-xyz-aduh-genapi \
  --service prod-xyz-aduh-genapi \
  --task-definition prod-xyz-aduh-genapi:previous-version
```

#### Scale Service
```bash
# Scale up for high load
aws --profile 836322468413 --region ap-southeast-3 ecs update-service \
  --cluster prod-xyz-aduh-genapi \
  --service prod-xyz-aduh-genapi \
  --desired-count 3

# Scale down for maintenance
aws --profile 836322468413 --region ap-southeast-3 ecs update-service \
  --cluster prod-xyz-aduh-genapi \
  --service prod-xyz-aduh-genapi \
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
aws --profile 836322468413 --region ap-southeast-3 cloudwatch get-metric-statistics \
  --namespace AWS/ECS \
  --metric-name CPUUtilization \
  --dimensions Name=ServiceName,Value=prod-xyz-aduh-genapi Name=ClusterName,Value=prod-xyz-aduh-genapi \
  --start-time 2025-01-01T00:00:00Z \
  --end-time 2025-01-01T23:59:59Z \
  --period 3600 \
  --statistics Average

# Check recent deployments
aws --profile 836322468413 --region ap-southeast-3 ecs describe-services \
  --cluster prod-xyz-aduh-genapi --services prod-xyz-aduh-genapi \
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
aws --profile 836322468413 sts get-caller-identity

# Check permissions
aws --profile 836322468413 iam get-user
```

## 📚 Additional Resources

- **[Development Guide](development-guide.md)** - Local development setup
- **[Release Process](../.cursor/rules/development_guide.mdc#release-process)** - Creating releases
- **[AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)** - Official AWS documentation
- **[AWS CodeBuild Documentation](https://docs.aws.amazon.com/codebuild/)** - CI/CD pipeline documentation

---

**Need Help?** Contact the infrastructure team or check the [troubleshooting section](../.cursor/rules/running_and_testing.mdc#troubleshooting) for common issues.