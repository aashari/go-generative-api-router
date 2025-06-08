#!/bin/bash

echo "Verifying project structure..."

# Check key directories exist
dirs=("configs" "deployments/docker" "docs" 
      "examples/curl" "examples/clients" "scripts" "testdata/fixtures" "testdata/analysis")

for dir in "${dirs[@]}"; do
    if [ -d "$dir" ]; then
        echo "✅ $dir"
    else
        echo "❌ Missing: $dir"
    fi
done

# Check key files exist
files=("Makefile" ".gitignore" "configs/models.json" "deployments/docker/Dockerfile"
       "scripts/deploy.sh" "scripts/test.sh" "scripts/build.sh" "scripts/setup.sh")

for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file"
    else
        echo "❌ Missing: $file"
    fi
done

# Check no config files in root
if ls *.json 2>/dev/null | grep -q .; then
    echo "⚠️  JSON files found in root directory"
else
    echo "✅ No JSON files in root"
fi

echo "Verification complete!" 