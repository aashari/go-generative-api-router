#!/bin/bash

# Install Git Hooks for Security Checks
# This script sets up pre-commit hooks to prevent sensitive data commits

set -e

echo "🔧 Installing Git hooks for security checks..."

# Create .git/hooks directory if it doesn't exist
mkdir -p .git/hooks

# Create pre-commit hook
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash

# Pre-commit hook to check for sensitive data
# This prevents commits containing AWS Account IDs, API keys, etc.

echo "🔒 Running pre-commit security checks..."

# Run the security check script
if [ -f "scripts/security-check.sh" ]; then
    ./scripts/security-check.sh
    SECURITY_EXIT_CODE=$?
    
    if [ $SECURITY_EXIT_CODE -ne 0 ]; then
        echo ""
        echo "🚨 COMMIT BLOCKED: Security issues found!"
        echo "Please fix the issues above before committing."
        echo ""
        echo "To bypass this check (NOT RECOMMENDED):"
        echo "  git commit --no-verify"
        echo ""
        echo "To fix issues:"
        echo "  1. Move sensitive data to .env files"
        echo "  2. Use environment variables instead of hardcoded values"
        echo "  3. Add sensitive files to .gitignore"
        echo "  4. Remove sensitive data from tracked files"
        exit 1
    fi
else
    echo "⚠️  Warning: Security check script not found at scripts/security-check.sh"
fi

echo "✅ Pre-commit security checks passed!"
EOF

# Make the pre-commit hook executable
chmod +x .git/hooks/pre-commit

echo "✅ Git pre-commit hook installed successfully!"
echo ""
echo "The pre-commit hook will now:"
echo "  - Check for AWS Account IDs"
echo "  - Check for AWS Access Keys"
echo "  - Check for API Keys in tracked files"
echo "  - Check for hardcoded secrets"
echo "  - Verify sensitive files are gitignored"
echo ""
echo "To test the hook:"
echo "  git add . && git commit -m 'test commit'"
echo ""
echo "To bypass the hook (NOT RECOMMENDED):"
echo "  git commit --no-verify" 