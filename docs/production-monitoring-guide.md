# Production Monitoring Guide

Practical guide for querying and monitoring production workloads for the Generative API Router service deployed as `go-generative-api-router`.

## 🔧 Environment Setup

### AWS Environment Variables
**CRITICAL: Load .env First, Then Configure AWS:**

```bash
# STEP 1: Load environment variables from .env file (MANDATORY)
export $(cat .env | grep -v '^#' | xargs) && echo "✅ Environment loaded from .env" | cat

# STEP 2: Set up AWS cluster/service names based on SERVICE_NAME from .env
export AWS_CLUSTER_DEV=dev-$SERVICE_NAME AWS_SERVICE_DEV=dev-$SERVICE_NAME AWS_CLUSTER_PROD=prod-$SERVICE_NAME AWS_SERVICE_PROD=prod-$SERVICE_NAME && echo "✅ AWS environment configured" | cat

# STEP 3: Verify configuration
echo "Service Name: $SERVICE_NAME" && echo "AWS Account: $AWS_ACCOUNT_ID" && echo "AWS Region: $AWS_REGION" && echo "Prod Cluster: $AWS_CLUSTER_PROD" && echo "Prod Service: $AWS_SERVICE_PROD" | cat
```

### Time Range Helpers

Quick timestamp generators for common time ranges:

```bash
# Past 3 hours
START_TS=$(( $(date -u -d '3 hours ago' +%s) * 1000 ))

# Past 1 hour  
START_TS=$(( $(date -u -d '1 hour ago' +%s) * 1000 ))

# Past 24 hours
START_TS=$(( $(date -u -d '24 hours ago' +%s) * 1000 ))

# Past 30 minutes
START_TS=$(( $(date -u -d '30 minutes ago' +%s) * 1000 ))

# Custom range (replace with actual dates)
START_TS=$(( $(date -u -d '2025-06-06 10:00:00' +%s) * 1000 ))
END_TS=$(( $(date -u -d '2025-06-06 12:00:00' +%s) * 1000 ))

# Current time for end range
END_TS=$(( $(date -u +%s) * 1000 ))
```

## 🔍 Essential Query Patterns

### Error Detection and Monitoring

**Find All Errors:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "?ERROR ?error ?Error ?failed ?Failed ?panic ?timeout" | jq -r '.events[].message' | cat
```

**Error Count by Type:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "ERROR" | jq -r '.events[].message | fromjson | .error.type // "unknown"' | sort | uniq -c | sort -nr | cat
```

**Recent Critical Errors:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "ERROR" | jq -r '.events[-5:][].message | fromjson | "\(.timestamp) - \(.error.type // "unknown") - \(.error.message // .message)"' | cat
```

### Request Analytics

**Total Request Count:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events | length' | cat
```

**Endpoint Usage Breakdown:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events[].message | fromjson | .request.path' | sort | uniq -c | sort -nr | cat
```

**HTTP Status Code Distribution:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events[].message | fromjson | .response.status_code' | sort | uniq -c | sort -nr | cat
```

**Requests Per Minute (Approximate):**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events[].message | fromjson | .timestamp[:16]' | sort | uniq -c | cat
```

### Performance Monitoring

**Response Time Analysis:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events[].message | fromjson | .attributes.duration_ms' | sort -n | tail -10 | cat
```

**Slow Requests (>5 seconds):**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events[].message | fromjson | select(.attributes.duration_ms > 5000) | "\(.timestamp) - \(.attributes.duration_ms)ms - \(.request.path)"' | cat
```

**Average Response Time:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events[].message | fromjson | .attributes.duration_ms' | awk '{sum+=$1; count++} END {if(count>0) print "Average:", sum/count "ms"; else print "No data"}' | cat
```

## 🎯 API-Specific Monitoring

### Chat Completions Analytics

**Chat Completion Requests:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"/v1/chat/completions\"" | jq -r '.events | length' && echo "chat completion requests" | cat
```

**Vendor Distribution:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Proxy request initiated\"" | jq -r '.events[].message | fromjson | .attributes.selected_vendor' | sort | uniq -c | sort -nr | cat
```

**Model Usage Patterns:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Proxy request initiated\"" | jq -r '.events[].message | fromjson | "\(.attributes.original_model) -> \(.attributes.selected_vendor):\(.attributes.selected_model)"' | sort | uniq -c | sort -nr | cat
```

**Streaming vs Non-Streaming:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"/v1/chat/completions\"" | jq -r '.events[].message | fromjson | if .request.body | contains("\"stream\":true") then "streaming" else "non-streaming" end' | sort | uniq -c | cat
```

### User Agent Analysis

**Client Distribution:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events[].message | fromjson | .request.headers["User-Agent"][0] // "unknown"' | sort | uniq -c | sort -nr | cat
```

**SDK Usage Patterns:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events[].message | fromjson | .request.headers["User-Agent"][0] // "unknown"' | grep -E "(openai|python|node|curl|postman)" | sort | uniq -c | sort -nr | cat
```

## 🚨 Service Health Checks

### ECS Service Status

**Current Service State:**
```bash
aws --region $AWS_REGION ecs describe-services --cluster $AWS_CLUSTER_PROD --services $AWS_SERVICE_PROD --query 'services[0].{Status:status,Running:runningCount,Desired:desiredCount,Platform:platformVersion}' | cat
```

**Task Health:**
```bash
aws --region $AWS_REGION ecs list-tasks --cluster $AWS_CLUSTER_PROD --service-name $AWS_SERVICE_PROD | jq -r '.taskArns[]' | head -1 | xargs -I {} aws --region $AWS_REGION ecs describe-tasks --cluster $AWS_CLUSTER_PROD --tasks {} --query 'tasks[0].{Status:lastStatus,Health:healthStatus,CPU:cpu,Memory:memory}' | cat
```

**Recent Deployments:**
```bash
aws --region $AWS_REGION ecs describe-services --cluster $AWS_CLUSTER_PROD --services $AWS_SERVICE_PROD --query 'services[0].deployments[0].{Status:status,CreatedAt:createdAt,UpdatedAt:updatedAt,TaskDefinition:taskDefinition}' | cat
```

**Service Events (Issues):**
```bash
aws --region $AWS_REGION ecs describe-services --cluster $AWS_CLUSTER_PROD --services $AWS_SERVICE_PROD --query 'services[0].events[0:5].{CreatedAt:createdAt,Message:message}' | cat
```

### CloudWatch Metrics

**CPU and Memory Usage:**
```bash
aws --region $AWS_REGION cloudwatch get-metric-statistics --namespace AWS/ECS --metric-name CPUUtilization --dimensions Name=ServiceName,Value=$AWS_SERVICE_PROD Name=ClusterName,Value=$AWS_CLUSTER_PROD --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) --end-time $(date -u +%Y-%m-%dT%H:%M:%S) --period 300 --statistics Average | jq -r '.Datapoints | sort_by(.Timestamp) | .[-1] | "CPU: \(.Average)%"' | cat
```

## 🔧 Advanced Filtering

### Request Tracing

**Follow Specific Request:**
```bash
REQUEST_ID="your-request-id-here"
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"$REQUEST_ID\"" | jq -r '.events[].message | fromjson | "\(.timestamp) - \(.message)"' | cat
```

**High-Traffic IPs:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events[].message | fromjson | .request.headers["X-Forwarded-For"][0] // .request.headers["X-Real-IP"][0] // "unknown"' | sort | uniq -c | sort -nr | head -10 | cat
```

### Error Deep Dive

**Vendor-Specific Errors:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "ERROR" | jq -r '.events[].message | fromjson | select(.attributes.vendor) | "\(.timestamp) - \(.attributes.vendor) - \(.error.message // .message)"' | cat
```

**Authentication Issues:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "401" | jq -r '.events[].message | fromjson | "\(.timestamp) - \(.request.path) - \(.request.headers["User-Agent"][0] // "unknown")"' | cat
```

**Rate Limit Tracking:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "429" | jq -r '.events[].message | fromjson | "\(.timestamp) - \(.request.headers["User-Agent"][0] // "unknown")"' | cat
```

## 📊 Business Intelligence

### Usage Patterns

**Peak Hours Analysis:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events[].message | fromjson | .timestamp[11:13]' | sort | uniq -c | sort -k2 -n | cat
```

**Daily Request Volume:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events[].message | fromjson | .timestamp[0:10]' | sort | uniq -c | cat
```

**Content Size Analysis:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events[].message | fromjson | .response.content_length' | sort -n | tail -10 | cat
```

## 💡 Pro Tips

### Query Optimization
- **Always pipe to `| cat`** for clean output in terminal environments
- **Use `--max-items 100`** for large result sets to avoid timeouts
- **Remove `--max-items`** when you need exact counts
- **Chain with `| head -10` or `| tail -5`** for manageable output
- **Use `jq -r`** for clean text output vs `jq` for formatted JSON

### Testing Filters
- **Test jq filters** on small datasets first before running on large logs
- **Use `echo '{"test": "data"}' | jq 'your_filter'`** to validate jq syntax
- **Start with small time ranges** then expand once the query works

### Performance
- **Specify end times** with `--end-time $END_TS` for better performance
- **Use specific filter patterns** instead of broad searches when possible
- **Combine multiple filters** in CloudWatch filter syntax for efficiency

### Emergency Debugging

**Quick Service Check:**
```bash
curl -f https://genapi.example.com/health && echo "✅ Service responding" || echo "❌ Service down" | cat
```

**Last 10 Critical Events:**
```bash
aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "?ERROR ?CRITICAL ?FATAL" --max-items 10 | jq -r '.events[].message | fromjson | "\(.timestamp) - \(.level) - \(.message)"' | cat
```

**Current Load:**
```bash
START_TS=$(( $(date -u -d '5 minutes ago' +%s) * 1000 )) && aws logs filter-log-events --profile $AWS_PROFILE --region $AWS_REGION --log-group-name "/aws/ecs/service/$AWS_SERVICE_PROD" --start-time $START_TS --filter-pattern "\"Request completed\"" | jq -r '.events | length' && echo "requests in last 5 minutes" | cat
```

---

For more detailed information, see:
- **[Deployment Guide](deployment-guide.md)** - AWS infrastructure details
- **[Logging Guide](logging-guide.md)** - Understanding log structure
- **[Development Guide](development-guide.md)** - Local debugging techniques