#!/bin/bash

# Security Check Script for AWS Account IDs and Sensitive Data
# This script helps prevent accidental commits of sensitive information

set -e

echo "🔒 Running Security Check for Sensitive Data..."
echo "================================================"

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Track if any issues are found
ISSUES_FOUND=0

# Function to report issues
report_issue() {
    echo -e "${RED}❌ SECURITY ISSUE FOUND:${NC} $1"
    ISSUES_FOUND=1
}

# Function to report warnings
report_warning() {
    echo -e "${YELLOW}⚠️  WARNING:${NC} $1"
}

# Function to report success
report_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

echo "1. Checking for AWS Account IDs (12-digit numbers)..."
AWS_ACCOUNT_MATCHES=$(grep -r -E '\b[0-9]{12}\b' \
    --exclude-dir=.git \
    --exclude-dir=node_modules \
    --exclude-dir=vendor \
    --exclude-dir=.terraform \
    --exclude-dir=build \
    --exclude="*.log" \
    --exclude="security-check.sh" \
    --exclude="SECURITY.md" \
    . 2>/dev/null | grep -v -E '(like `[0-9]{12}`|example.*[0-9]{12}|placeholder.*[0-9]{12})' || true)

if [ -n "$AWS_ACCOUNT_MATCHES" ]; then
    # Check if matches are only in gitignored files
    GITIGNORED_AWS_MATCHES=$(echo "$AWS_ACCOUNT_MATCHES" | grep -E '\./\.env' || true)
    NON_GITIGNORED_AWS_MATCHES=$(echo "$AWS_ACCOUNT_MATCHES" | grep -v -E '\./\.env' || true)
    
    if [ -n "$NON_GITIGNORED_AWS_MATCHES" ]; then
        report_issue "AWS Account IDs found in tracked files:"
        echo "$NON_GITIGNORED_AWS_MATCHES"
        echo ""
    fi
    
    if [ -n "$GITIGNORED_AWS_MATCHES" ]; then
        report_warning "AWS Account IDs found in gitignored files (this is expected):"
        echo "$GITIGNORED_AWS_MATCHES"
        echo ""
    fi
    
    if [ -z "$NON_GITIGNORED_AWS_MATCHES" ]; then
        report_success "AWS Account IDs only found in gitignored files (safe)"
    fi
else
    report_success "No AWS Account IDs found in tracked files"
fi

echo "2. Checking for AWS Access Keys..."
AWS_KEY_MATCHES=$(grep -r -E '(AKIA|ASIA)[0-9A-Z]{16}' \
    --exclude-dir=.git \
    --exclude-dir=node_modules \
    --exclude-dir=vendor \
    --exclude-dir=.terraform \
    --exclude-dir=build \
    --exclude="*.log" \
    --exclude="security-check.sh" \
    . 2>/dev/null || true)

if [ -n "$AWS_KEY_MATCHES" ]; then
    report_issue "AWS Access Keys found in files:"
    echo "$AWS_KEY_MATCHES"
    echo ""
else
    report_success "No AWS Access Keys found in tracked files"
fi

echo "3. Checking for AWS Secret Keys..."
AWS_SECRET_MATCHES=$(grep -r -E 'aws_secret_access_key|AWS_SECRET_ACCESS_KEY' \
    --exclude-dir=.git \
    --exclude-dir=node_modules \
    --exclude-dir=vendor \
    --exclude-dir=.terraform \
    --exclude-dir=build \
    --exclude="*.log" \
    --exclude="security-check.sh" \
    . 2>/dev/null || true)

if [ -n "$AWS_SECRET_MATCHES" ]; then
    report_issue "AWS Secret references found in files:"
    echo "$AWS_SECRET_MATCHES"
    echo ""
else
    report_success "No AWS Secret references found in tracked files"
fi

echo "4. Checking for AWS Resource IDs (VPC, Subnet, Security Group)..."
AWS_RESOURCE_MATCHES=$(grep -r -E '(vpc-[a-z0-9]+|subnet-[a-z0-9]+|sg-[a-z0-9]+)' \
    --exclude-dir=.git \
    --exclude-dir=node_modules \
    --exclude-dir=vendor \
    --exclude-dir=.terraform \
    --exclude-dir=build \
    --exclude="*.log" \
    --exclude="security-check.sh" \
    --exclude="SECURITY.md" \
    . 2>/dev/null | grep -v -E '(like `[a-z0-9-]+`|example.*[a-z0-9-]+)' || true)

if [ -n "$AWS_RESOURCE_MATCHES" ]; then
    report_issue "AWS Resource IDs found in files:"
    echo "$AWS_RESOURCE_MATCHES"
    echo ""
else
    report_success "No AWS Resource IDs found in tracked files"
fi

echo "5. Checking for API Keys (OpenAI, Gemini, etc.)..."
API_KEY_MATCHES=$(grep -r -E '(sk-[a-zA-Z0-9]{48}|AIza[a-zA-Z0-9_-]{35})' \
    --exclude-dir=.git \
    --exclude-dir=node_modules \
    --exclude-dir=vendor \
    --exclude-dir=.terraform \
    --exclude-dir=build \
    --exclude="*.log" \
    --exclude="security-check.sh" \
    --exclude="credentials.json.example" \
    . 2>/dev/null || true)

if [ -n "$API_KEY_MATCHES" ]; then
    # Check if matches are only in gitignored files
    GITIGNORED_MATCHES=$(echo "$API_KEY_MATCHES" | grep -E '\./configs/credentials\.json|\./\.env' || true)
    NON_GITIGNORED_MATCHES=$(echo "$API_KEY_MATCHES" | grep -v -E '\./configs/credentials\.json|\./\.env' || true)
    
    if [ -n "$NON_GITIGNORED_MATCHES" ]; then
        report_issue "API Keys found in tracked files:"
        echo "$NON_GITIGNORED_MATCHES"
        echo ""
    fi
    
    if [ -n "$GITIGNORED_MATCHES" ]; then
        report_warning "API Keys found in gitignored files (this is expected):"
        echo "$GITIGNORED_MATCHES"
        echo ""
    fi
    
    if [ -z "$NON_GITIGNORED_MATCHES" ]; then
        report_success "API Keys only found in gitignored files (safe)"
    fi
else
    report_success "No API Keys found in tracked files"
fi

echo "6. Checking for sensitive files that should be gitignored..."
SENSITIVE_FILES=()

# Check for .env files (except .env.example)
if [ -f ".env" ] && ! grep -q "^\.env$" .gitignore 2>/dev/null; then
    SENSITIVE_FILES+=(".env")
fi

# Check for credentials.json
if [ -f "configs/credentials.json" ] && ! grep -q "configs/credentials\.json" .gitignore 2>/dev/null; then
    SENSITIVE_FILES+=("configs/credentials.json")
fi

# Check for deploy.sh
if [ -f "scripts/deploy.sh" ] && ! grep -q "scripts/deploy\.sh" .gitignore 2>/dev/null; then
    SENSITIVE_FILES+=("scripts/deploy.sh")
fi

if [ ${#SENSITIVE_FILES[@]} -gt 0 ]; then
    report_issue "Sensitive files not properly gitignored:"
    for file in "${SENSITIVE_FILES[@]}"; do
        echo "  - $file"
    done
    echo ""
else
    report_success "All sensitive files are properly gitignored"
fi

echo "7. Checking git status for staged sensitive files..."
if command -v git >/dev/null 2>&1 && [ -d .git ]; then
    STAGED_SENSITIVE=$(git diff --cached --name-only | grep -E '\.(env|key|pem|p12|pfx)$|credentials|secrets|deploy\.sh' || true)
    
    if [ -n "$STAGED_SENSITIVE" ]; then
        report_issue "Sensitive files are staged for commit:"
        echo "$STAGED_SENSITIVE"
        echo ""
    else
        report_success "No sensitive files staged for commit"
    fi
fi

echo "8. Checking for hardcoded secrets in environment variable assignments..."
HARDCODED_SECRETS=$(grep -r -E '(PASSWORD|SECRET|KEY|TOKEN)=[^$]' \
    --exclude-dir=.git \
    --exclude-dir=node_modules \
    --exclude-dir=vendor \
    --exclude-dir=.terraform \
    --exclude-dir=build \
    --exclude="*.log" \
    --exclude="security-check.sh" \
    --exclude=".env.example" \
    --exclude="credentials.json.example" \
    --exclude="SECURITY.md" \
    --exclude="user-guide.md" \
    --exclude="plan-correlation.md" \
    . 2>/dev/null | grep -v -E '(PASSWORD|SECRET|KEY|TOKEN)=\$|your-.*-here|example.*=|placeholder.*=|<your-.*>|your-password|# MongoDB.*optional' || true)

if [ -n "$HARDCODED_SECRETS" ]; then
    # Check if matches are only in gitignored files
    GITIGNORED_SECRET_MATCHES=$(echo "$HARDCODED_SECRETS" | grep -E '\./\.env' || true)
    NON_GITIGNORED_SECRET_MATCHES=$(echo "$HARDCODED_SECRETS" | grep -v -E '\./\.env' || true)
    
    if [ -n "$NON_GITIGNORED_SECRET_MATCHES" ]; then
        report_issue "Hardcoded secrets found in tracked files:"
        echo "$NON_GITIGNORED_SECRET_MATCHES"
        echo ""
    fi
    
    if [ -n "$GITIGNORED_SECRET_MATCHES" ]; then
        report_warning "Secrets found in gitignored files (this is expected):"
        echo "$GITIGNORED_SECRET_MATCHES"
        echo ""
    fi
    
    if [ -z "$NON_GITIGNORED_SECRET_MATCHES" ]; then
        report_success "Secrets only found in gitignored files (safe)"
    fi
else
    report_success "No hardcoded secrets found"
fi

echo "================================================"

if [ $ISSUES_FOUND -eq 0 ]; then
    echo -e "${GREEN}🎉 SECURITY CHECK PASSED!${NC}"
    echo "No sensitive data issues found. Safe to commit."
    exit 0
else
    echo -e "${RED}🚨 SECURITY CHECK FAILED!${NC}"
    echo "Please fix the issues above before committing."
    echo ""
    echo "Common fixes:"
    echo "1. Move sensitive data to .env files (already gitignored)"
    echo "2. Use environment variables instead of hardcoded values"
    echo "3. Add sensitive files to .gitignore"
    echo "4. Remove sensitive data from tracked files"
    echo ""
    echo "To fix AWS Account ID issues:"
    echo "- Move AWS_ACCOUNT_ID to .env file"
    echo "- Use \${AWS_ACCOUNT_ID} in scripts instead of hardcoded values"
    echo "- Ensure scripts/deploy.sh is gitignored"
    exit 1
fi 