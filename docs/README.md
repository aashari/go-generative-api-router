# Documentation

Welcome to the Generative API Router documentation! This comprehensive guide covers everything you need to know about using, developing, and deploying the **multi-vendor OpenAI-compatible API router**.

## 🏗️ **Service Overview**

The Generative API Router is a **production-ready transparent proxy** that provides a unified OpenAI-compatible API interface while routing requests to multiple LLM vendors (OpenAI, Gemini) behind the scenes. 

### **Key Characteristics**
- **Multi-Vendor Design**: 19 credentials (18 Gemini + 1 OpenAI) with 4 models
- **Transparent Proxy**: Preserves your original model names in responses
- **Even Distribution**: Fair selection across 114 vendor-credential-model combinations
- **Enterprise-Grade**: Security, reliability, monitoring, and performance optimizations

## 📖 Quick Navigation

### 👤 For Users
- **[User Guide](user-guide.md)** - Complete guide for integrating with and using the service
- **[API Reference](api-reference.md)** - Complete API documentation with examples
- **[Examples](../examples/)** - Ready-to-use code examples in multiple languages

### 👨‍💻 For Developers  
- **[Development Guide](development-guide.md)** - Setup, workflow, and architecture overview
- **[Contributing Guide](contributing-guide.md)** - How to contribute to the project
- **[Testing Guide](testing-guide.md)** - Testing strategies and procedures
- **[Logging Guide](logging-guide.md)** - Comprehensive logging documentation

### 🚀 For DevOps & Deployment
- **[Deployment Guide](deployment-guide.md)** - AWS infrastructure and deployment procedures
- **[Production Monitoring Guide](production-monitoring-guide.md)** - Practical log querying and monitoring

### 🤖 For Cursor AI Development
- **[Development Guide](../.cursor/rules/development_guide.mdc)** - Complete workflow, architecture, Git practices
- **[Running & Testing Guide](../.cursor/rules/running_and_testing.mdc)** - Setup, testing, debugging

## 🚀 Getting Started Paths

### **I want to use the service**
1. Read the [Main README](../README.md) for project overview
2. Follow the [User Guide](user-guide.md) for setup and integration
3. Reference the [API Documentation](api-reference.md) for detailed endpoints
4. Try the [Examples](../examples/) for your programming language

### **I want to contribute**
1. Read the [Development Guide](development-guide.md) for setup
2. Follow the [Contributing Guide](contributing-guide.md) for workflow
3. Check the [Testing Guide](testing-guide.md) for testing procedures
4. Understand the [Logging Guide](logging-guide.md) for debugging

### **I want to deploy or manage infrastructure**
1. Review the [Deployment Guide](deployment-guide.md) for AWS setup
2. Understand monitoring and troubleshooting procedures
3. Check the [Development Guide](development-guide.md) for local testing

### **I'm using Cursor AI for development**
1. Review the [Cursor Development Guide](../.cursor/rules/development_guide.mdc) for complete context
2. Use the [Running & Testing Guide](../.cursor/rules/running_and_testing.mdc) for testing procedures
3. Understand the multi-vendor architecture and recent improvements

## 📋 Documentation Overview

| Document | Purpose | Audience |
|----------|---------|----------|
| **[User Guide](user-guide.md)** | Service integration and usage | End users, developers integrating the API |
| **[API Reference](api-reference.md)** | Complete API specification | Developers, API consumers |
| **[Development Guide](development-guide.md)** | Development setup and workflow | Contributors, maintainers |
| **[Contributing Guide](contributing-guide.md)** | Contribution process and standards | Contributors |
| **[Testing Guide](testing-guide.md)** | Testing strategies and procedures | Developers, maintainers |
| **[Logging Guide](logging-guide.md)** | Logging system documentation | Developers, operations |
| **[Deployment Guide](deployment-guide.md)** | Infrastructure and deployment | DevOps, maintainers |
| **[Production Monitoring Guide](production-monitoring-guide.md)** | Log querying and monitoring | DevOps, operations |

## 🔧 Service Overview

The Generative API Router is a transparent proxy service that:

- **Routes OpenAI-compatible requests** to multiple LLM vendors (OpenAI, Gemini)
- **Preserves your model names** in responses while intelligently selecting vendors
- **Maintains full API compatibility** with OpenAI's chat completions API
- **Supports advanced features** including streaming, tool calling, and vendor selection
- **Provides comprehensive logging** for monitoring and debugging
- **Includes enterprise features** like circuit breakers, retry logic, and health monitoring

### Key Features

✅ **Transparent Proxy** - Original model names preserved in responses  
✅ **Multi-Vendor Support** - OpenAI, Gemini, and more  
✅ **Streaming Support** - Real-time response streaming  
✅ **Tool Calling** - OpenAI-compatible function calling  
✅ **Vendor Selection** - Force specific vendors via query parameters  
✅ **Enterprise Logging** - Structured JSON logging with request correlation  
✅ **Production Ready** - Deployed on AWS with monitoring  
✅ **Security** - Encrypted credentials, sensitive data masking  
✅ **Reliability** - Circuit breakers, retry logic, health monitoring  

### Recent Enterprise Improvements (2024)

The service has undergone comprehensive enterprise-grade improvements:

- 🔒 **Security Enhancements**: AES-GCM encryption for credentials, sensitive data masking in logs
- 🔄 **Reliability Features**: Exponential backoff retry logic, circuit breaker pattern implementation
- 📊 **Monitoring & Health**: Comprehensive health checks with vendor connectivity monitoring
- ⚡ **Performance**: Production-optimized logging, conditional detail levels
- 🧹 **Code Quality**: DRY principles, centralized utilities, eliminated code duplication

## 🎯 Quick Start

### For End Users
```bash
curl -X POST https://genapi.example.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "my-preferred-model",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### For Developers
```bash
git clone https://github.com/aashari/go-generative-api-router.git
cd go-generative-api-router
make setup
make run
```

### For DevOps
```bash
# Check production status
curl https://genapi.example.com/health

# Monitor AWS deployment
aws --profile ${AWS_ACCOUNT_ID} --region ap-southeast-3 ecs describe-services \
  --cluster prod-${SERVICE_NAME} --services prod-${SERVICE_NAME}
```

### Multi-Vendor Testing
```bash
# Test OpenAI vendor
curl -X POST "http://localhost:8082/v1/chat/completions?vendor=openai" \
  -H "Content-Type: application/json" \
  -d '{"model": "test-openai", "messages": [{"role": "user", "content": "Hello"}]}'

# Test Gemini vendor
curl -X POST "http://localhost:8082/v1/chat/completions?vendor=gemini" \
  -H "Content-Type: application/json" \
  -d '{"model": "test-gemini", "messages": [{"role": "user", "content": "Hello"}]}'

# Monitor vendor distribution
grep "Even distribution selected combination" logs/server.log | tail -5
```

## 🔗 External Resources

- **[GitHub Repository](https://github.com/aashari/go-generative-api-router)** - Source code and issues
- **[Production Service](https://genapi.example.com)** - Live production deployment
- **[Development Service](https://dev-genapi.example.com)** - Development environment
- **[License](../LICENSE)** - MIT License

## 📝 Documentation Standards

All documentation follows these principles:
- **Clear and concise** - Easy to understand and follow
- **Up-to-date** - Reflects current implementation including recent improvements
- **Comprehensive** - Covers all necessary information
- **Well-organized** - Logical structure and navigation
- **Example-driven** - Includes practical examples
- **Multi-vendor aware** - Acknowledges the true multi-vendor architecture

## 💡 Getting Help

- **Documentation Issues**: Open an issue on [GitHub](https://github.com/aashari/go-generative-api-router/issues)
- **Feature Requests**: Use GitHub issues with the "enhancement" label
- **Bug Reports**: Follow the bug report template on GitHub
- **General Questions**: Use GitHub Discussions
- **Multi-Vendor Questions**: Check the [Cursor Development Guide](../.cursor/rules/development_guide.mdc) for architecture details

## 🏗️ Architecture Quick Reference

### Multi-Vendor Design
- **19 credentials** (18 Gemini + 1 OpenAI)
- **4 models** (2 Gemini + 2 OpenAI)
- **114 total combinations** with even distribution
- **OpenAI-compatible endpoints** for all vendors

### Core Components
- **Proxy Handler**: Routes requests to selected vendors
- **Vendor Selector**: Even distribution across combinations
- **Request Validator**: OpenAI-compatible validation
- **Response Processor**: Model name transparency
- **Health Monitor**: Vendor connectivity monitoring
- **Circuit Breaker**: Reliability pattern implementation
- **Retry Logic**: Exponential backoff for failures

---

**Need something specific?** Use the navigation above to jump to the right guide, or check the [main README](../README.md) for a project overview.