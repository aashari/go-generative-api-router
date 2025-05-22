#!/bin/bash
set -e

# Configuration
AWS_PROFILE="836322468413"
AWS_REGION="ap-southeast-3"
CLUSTER_NAME="go-generative-api-router-cluster"
SERVICE_NAME="go-api-router-service"
ECR_REPO_NAME="go-generative-api-router"
IMAGE_TAG="$(date +%Y%m%d%H%M%S)"
TASK_FAMILY="go-generative-api-router-task"
CONTAINER_NAME="api-router-container"
VPC_ID="vpc-0413aa0896fb82473"
SUBNETS="subnet-0651d2adc4e3b3f6d,subnet-00e592bcaa6b4e228,subnet-0832915514b09b737"
SECURITY_GROUP="sg-00a03957cf890e935"
LOAD_BALANCER_NAME="go-api-router-alb"
TARGET_GROUP_NAME="go-api-router-tg"
PORT=8082
CPU="512"
MEMORY="1024"
DESIRED_COUNT=3

echo "🚀 Starting deployment process..."

# Get AWS account ID
ACCOUNT_ID=$(aws --profile $AWS_PROFILE sts get-caller-identity --query 'Account' --output text)
ECR_REPO="$ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$ECR_REPO_NAME"

# Check if ECR repository exists, create if not
if ! aws --profile $AWS_PROFILE --region $AWS_REGION ecr describe-repositories --repository-names $ECR_REPO_NAME &> /dev/null; then
  echo "📦 Creating ECR repository $ECR_REPO_NAME..."
  aws --profile $AWS_PROFILE --region $AWS_REGION ecr create-repository --repository-name $ECR_REPO_NAME
else
  echo "📦 ECR repository $ECR_REPO_NAME already exists."
fi

# Log in to ECR
echo "🔑 Logging in to ECR..."
aws --profile $AWS_PROFILE --region $AWS_REGION ecr get-login-password | docker login --username AWS --password-stdin "$ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"

# Build and tag Docker image for ARM64
echo "🏗️ Building ARM64 Docker image..."
docker build --no-cache --platform linux/arm64 -t "$ECR_REPO:$IMAGE_TAG" -t "$ECR_REPO:latest-arm64" .

# Push images to ECR
echo "⬆️ Pushing images to ECR..."
docker push "$ECR_REPO:$IMAGE_TAG"
docker push "$ECR_REPO:latest-arm64"

# Check if cluster exists, create if not
if ! aws --profile $AWS_PROFILE --region $AWS_REGION ecs describe-clusters --clusters $CLUSTER_NAME --query 'clusters[0].status' --output text &> /dev/null; then
  echo "🌐 Creating ECS cluster $CLUSTER_NAME..."
  aws --profile $AWS_PROFILE --region $AWS_REGION ecs create-cluster --cluster-name $CLUSTER_NAME
else
  echo "🌐 ECS cluster $CLUSTER_NAME already exists."
fi

# Create task definition JSON
echo "📄 Creating task definition..."
cat > task-definition.json << EOL
{
  "family": "${TASK_FAMILY}",
  "executionRoleArn": "arn:aws:iam::$ACCOUNT_ID:role/ecsTaskExecutionRole",
  "networkMode": "awsvpc",
  "containerDefinitions": [
    {
      "name": "$CONTAINER_NAME",
      "image": "$ECR_REPO:$IMAGE_TAG",
      "essential": true,
      "portMappings": [
        {
          "containerPort": $PORT,
          "hostPort": $PORT,
          "protocol": "tcp"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/${TASK_FAMILY}",
          "awslogs-region": "$AWS_REGION",
          "awslogs-stream-prefix": "ecs",
          "awslogs-create-group": "true",
          "mode": "non-blocking"
        }
      }
    }
  ],
  "requiresCompatibilities": [
    "FARGATE"
  ],
  "cpu": "$CPU",
  "memory": "$MEMORY",
  "runtimePlatform": {
    "cpuArchitecture": "ARM64",
    "operatingSystemFamily": "LINUX"
  }
}
EOL

# Register task definition
echo "📋 Registering task definition..."
TASK_DEFINITION_ARN=$(aws --profile $AWS_PROFILE --region $AWS_REGION ecs register-task-definition --cli-input-json file://task-definition.json --query 'taskDefinition.taskDefinitionArn' --output text)
echo "✅ Task definition registered: $TASK_DEFINITION_ARN (Family: ${TASK_FAMILY})"

# Check if CloudWatch log group exists, create if not
LOG_GROUP_NAME="/ecs/${TASK_FAMILY}"
if ! aws --profile $AWS_PROFILE --region $AWS_REGION logs describe-log-groups --log-group-name-prefix "$LOG_GROUP_NAME" --query 'logGroups[0].logGroupName' --output text &> /dev/null; then
  echo "📊 Creating CloudWatch log group $LOG_GROUP_NAME..."
  aws --profile $AWS_PROFILE --region $AWS_REGION logs create-log-group --log-group-name "$LOG_GROUP_NAME"
  
  # Set retention period for the log group (e.g., 14 days)
  echo "📊 Setting log retention period to 14 days..."
  aws --profile $AWS_PROFILE --region $AWS_REGION logs put-retention-policy --log-group-name "$LOG_GROUP_NAME" --retention-in-days 14
else
  echo "📊 CloudWatch log group $LOG_GROUP_NAME already exists."
  
  # Update retention period for existing log group
  echo "📊 Updating log retention period to 14 days..."
  aws --profile $AWS_PROFILE --region $AWS_REGION logs put-retention-policy --log-group-name "$LOG_GROUP_NAME" --retention-in-days 14
fi

# Check if target group exists, create if not
TARGET_GROUP_ARN=$(aws --profile $AWS_PROFILE --region $AWS_REGION elbv2 describe-target-groups --names $TARGET_GROUP_NAME --query 'TargetGroups[0].TargetGroupArn' --output text 2>/dev/null || echo "")
if [ -z "$TARGET_GROUP_ARN" ] || [ "$TARGET_GROUP_ARN" == "None" ]; then
  echo "🎯 Creating target group $TARGET_GROUP_NAME..."
  TARGET_GROUP_ARN=$(aws --profile $AWS_PROFILE --region $AWS_REGION elbv2 create-target-group --name $TARGET_GROUP_NAME --protocol HTTP --port $PORT --vpc-id $VPC_ID --target-type ip --health-check-path /health --query 'TargetGroups[0].TargetGroupArn' --output text)
else
  echo "🎯 Target group $TARGET_GROUP_NAME already exists."
fi

# Check if load balancer exists, create if not
LOAD_BALANCER_ARN=$(aws --profile $AWS_PROFILE --region $AWS_REGION elbv2 describe-load-balancers --names $LOAD_BALANCER_NAME --query 'LoadBalancers[0].LoadBalancerArn' --output text 2>/dev/null || echo "")
if [ -z "$LOAD_BALANCER_ARN" ] || [ "$LOAD_BALANCER_ARN" == "None" ]; then
  echo "⚖️ Creating load balancer $LOAD_BALANCER_NAME..."
  SUBNET_LIST=(${SUBNETS//,/ })
  SUBNET_ARGS=""
  for SUBNET in "${SUBNET_LIST[@]}"; do
    SUBNET_ARGS="$SUBNET_ARGS --subnets $SUBNET"
  done
  LOAD_BALANCER_ARN=$(aws --profile $AWS_PROFILE --region $AWS_REGION elbv2 create-load-balancer --name $LOAD_BALANCER_NAME $SUBNET_ARGS --security-groups $SECURITY_GROUP --scheme internet-facing --query 'LoadBalancers[0].LoadBalancerArn' --output text)
  
  # Wait for load balancer to be active
  echo "⏳ Waiting for load balancer to be active..."
  aws --profile $AWS_PROFILE --region $AWS_REGION elbv2 wait load-balancer-available --load-balancer-arns $LOAD_BALANCER_ARN
  
  # Create listener
  echo "👂 Creating listener on port 80..."
  aws --profile $AWS_PROFILE --region $AWS_REGION elbv2 create-listener --load-balancer-arn $LOAD_BALANCER_ARN --protocol HTTP --port 80 --default-actions Type=forward,TargetGroupArn=$TARGET_GROUP_ARN
else
  echo "⚖️ Load balancer $LOAD_BALANCER_NAME already exists."
  
  # Check if listener exists, create if not
  LISTENER_ARN=$(aws --profile $AWS_PROFILE --region $AWS_REGION elbv2 describe-listeners --load-balancer-arn $LOAD_BALANCER_ARN --query 'Listeners[?Port==`80`].ListenerArn' --output text)
  if [ -z "$LISTENER_ARN" ] || [ "$LISTENER_ARN" == "None" ]; then
    echo "👂 Creating listener on port 80..."
    aws --profile $AWS_PROFILE --region $AWS_REGION elbv2 create-listener --load-balancer-arn $LOAD_BALANCER_ARN --protocol HTTP --port 80 --default-actions Type=forward,TargetGroupArn=$TARGET_GROUP_ARN
  else
    echo "👂 Listener on port 80 already exists."
  fi
fi

# Get the load balancer DNS name
LOAD_BALANCER_DNS=$(aws --profile $AWS_PROFILE --region $AWS_REGION elbv2 describe-load-balancers --names $LOAD_BALANCER_NAME --query 'LoadBalancers[0].DNSName' --output text)

# Check if service exists, update or create accordingly
SERVICE_EXISTS=$(aws --profile $AWS_PROFILE --region $AWS_REGION ecs describe-services --cluster $CLUSTER_NAME --services $SERVICE_NAME --query 'services[0].status' --output text 2>/dev/null || echo "")
if [ -z "$SERVICE_EXISTS" ] || [ "$SERVICE_EXISTS" == "None" ]; then
  echo "🚢 Creating ECS service $SERVICE_NAME..."
  aws --profile $AWS_PROFILE --region $AWS_REGION ecs create-service \
    --cluster $CLUSTER_NAME \
    --service-name $SERVICE_NAME \
    --task-definition $TASK_DEFINITION_ARN \
    --desired-count $DESIRED_COUNT \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[${SUBNETS}],securityGroups=[$SECURITY_GROUP],assignPublicIp=ENABLED}" \
    --load-balancers "targetGroupArn=$TARGET_GROUP_ARN,containerName=$CONTAINER_NAME,containerPort=$PORT"
else
  echo "🔄 Updating ECS service $SERVICE_NAME..."
  aws --profile $AWS_PROFILE --region $AWS_REGION ecs update-service \
    --cluster $CLUSTER_NAME \
    --service $SERVICE_NAME \
    --task-definition $TASK_DEFINITION_ARN \
    --load-balancers "targetGroupArn=$TARGET_GROUP_ARN,containerName=$CONTAINER_NAME,containerPort=$PORT" \
    --desired-count $DESIRED_COUNT \
    --force-new-deployment
fi

echo "✅ Deployment completed successfully!"
echo "🔗 Application URL: http://$LOAD_BALANCER_DNS"
echo "⏳ It may take a few minutes for the new task to start and the old one to drain."
echo "📊 CloudWatch logs will be available at: https://$AWS_REGION.console.aws.amazon.com/cloudwatch/home?region=$AWS_REGION#logsV2:log-groups/log-group/$LOG_GROUP_NAME"

# Clean up temporary files
rm -f task-definition.json 