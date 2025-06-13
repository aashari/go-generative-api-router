# Security Guide

This document outlines security practices and tools to prevent sensitive data from being committed to the repository.

## 🚨 Critical Security Issues

### AWS Account IDs and Resource IDs

**NEVER commit the following to the repository:**
- AWS Account IDs (12-digit numbers like `123456789012`)
- VPC IDs (like `vpc-0413aa0896fb82473`)
- Subnet IDs (like `subnet-0651d2adc4e3b3f6d`)
- Security Group IDs (like `sg-00a03957cf890e935`)
- AWS Access Keys (starting with `AKIA` or `ASIA`)
- AWS Secret Keys

### API Keys and Secrets

**NEVER commit the following:**
- OpenAI API Keys (starting with `sk-`)
- Gemini API Keys (starting with `AIza`)
- Any other API keys or tokens
- Database passwords
- Encryption keys

## 🛡️ Security Tools

### 1. Security Check Script

Run the comprehensive security check:

```bash
# Check for all types of sensitive data
./scripts/security-check.sh

# Or use the Makefile target
make security-check
```

This script checks for:
- AWS Account IDs (12-digit numbers)
- AWS Access Keys and Secret Keys
- AWS Resource IDs (VPC, Subnet, Security Group)
- API Keys (OpenAI, Gemini, etc.)
- Hardcoded secrets in environment variables
- Sensitive files that should be gitignored
- Staged sensitive files in git

### 2. Pre-commit Hooks

Install Git pre-commit hooks to automatically check for sensitive data:

```bash
# Install the pre-commit hook
./scripts/install-git-hooks.sh

# Or use the setup target (includes hooks)
make setup
```

The pre-commit hook will:
- Run security checks before every commit
- Block commits containing sensitive data
- Provide guidance on fixing issues

### 3. CI/CD Integration

Run comprehensive checks including security:

```bash
# Run all CI checks (formatting, linting, security, build)
make ci-check
```

## 🔧 Current Security Status

### Issues Found

Based on the latest security scan, the following issues were identified:

1. **AWS Account ID in .env file** ✅ (Expected - .env is gitignored)
2. **AWS Resource IDs in scripts/deploy.sh** ❌ (Should be moved to environment variables)
3. **API Keys in configs/credentials.json** ✅ (Expected - file is gitignored)
4. **Hardcoded secrets in documentation** ❌ (Should use placeholders)

### Files Properly Protected

The following sensitive files are properly gitignored:
- `.env` - Contains AWS Account ID and other environment variables
- `configs/credentials.json` - Contains API keys for all vendors
- `scripts/deploy.sh` - Contains AWS infrastructure IDs
- `logs/` - May contain sensitive data in logs

## 🔒 Best Practices

### 1. Environment Variables

Always use environment variables for sensitive data:

```bash
# ✅ Good - Use environment variables
AWS_ACCOUNT_ID=${AWS_ACCOUNT_ID}
VPC_ID=${VPC_ID}

# ❌ Bad - Hardcoded values
AWS_ACCOUNT_ID=123456789012
VPC_ID=vpc-0413aa0896fb82473
```

### 2. Configuration Files

Keep sensitive configuration in gitignored files:

```bash
# These files should be in .gitignore:
.env
.env.*
configs/credentials.json
scripts/deploy.sh
```

### 3. Documentation

Use placeholders in documentation:

```bash
# ✅ Good - Use placeholders
AWS_ACCOUNT_ID=your-account-id
GENAPI_API_KEY=your-api-key-here

# ❌ Bad - Real values
AWS_ACCOUNT_ID=123456789012
GENAPI_API_KEY=sk-real-key-here
```

## 🚨 Emergency Response

### If Sensitive Data is Committed

1. **Immediately revoke the exposed credentials**
2. **Remove from git history:**
   ```bash
   # Remove file from git history
   git filter-branch --force --index-filter \
     'git rm --cached --ignore-unmatch path/to/sensitive/file' \
     --prune-empty --tag-name-filter cat -- --all
   
   # Force push (dangerous - coordinate with team)
   git push origin --force --all
   ```
3. **Generate new credentials**
4. **Update all systems using the old credentials**

### If AWS Account ID is Exposed

1. **Review AWS CloudTrail logs** for unauthorized access
2. **Rotate access keys** if any were exposed
3. **Review security groups** and access policies
4. **Consider changing AWS Account ID** if extensively exposed

## 🔍 Manual Security Checks

### Quick Commands

```bash
# Check for AWS Account IDs
grep -r -E '\b[0-9]{12}\b' --exclude-dir=.git --exclude="*.log" .

# Check for AWS Access Keys
grep -r -E '(AKIA|ASIA)[0-9A-Z]{16}' --exclude-dir=.git --exclude="*.log" .

# Check for API Keys
grep -r -E '(sk-[a-zA-Z0-9]{48}|AIza[a-zA-Z0-9_-]{35})' --exclude-dir=.git --exclude="*.log" .

# Check for AWS Resource IDs
grep -r -E '(vpc-[a-z0-9]+|subnet-[a-z0-9]+|sg-[a-z0-9]+)' --exclude-dir=.git --exclude="*.log" .
```

### Git Status Check

```bash
# Check what's staged for commit
git diff --cached --name-only

# Check for sensitive files in staging
git diff --cached --name-only | grep -E '\.(env|key|pem|p12|pfx)$|credentials|secrets|deploy\.sh'
```

## 📋 Security Checklist

Before every commit:

- [ ] Run `make security-check`
- [ ] Verify no AWS Account IDs in tracked files
- [ ] Verify no API keys in tracked files
- [ ] Verify no hardcoded secrets
- [ ] Verify sensitive files are gitignored
- [ ] Check git staging area for sensitive files

Before every release:

- [ ] Run full `make ci-check`
- [ ] Review all documentation for sensitive data
- [ ] Verify deployment scripts use environment variables
- [ ] Test security hooks are working
- [ ] Audit access logs if applicable

## 🔗 Related Documentation

- [Development Guide](development-guide.md) - Complete development workflow
- [Contributing Guide](contributing-guide.md) - Contribution guidelines including security
- [Deployment Guide](deployment-guide.md) - Secure deployment practices

## 📞 Security Contact

If you discover a security vulnerability:

1. **DO NOT** create a public issue
2. **DO NOT** commit the vulnerability details
3. Contact the maintainers privately
4. Provide details about the vulnerability
5. Wait for confirmation before public disclosure 