---
name: Bug Report
about: Create a report to help us improve
title: '[BUG] '
labels: ['bug', 'needs-triage']
assignees: ''
---

# Bug Report

## Summary
<!-- A clear and concise description of what the bug is -->

## Environment
- **OS**: <!-- e.g., Ubuntu 20.04, macOS 12.0, Windows 11 -->
- **Go Version**: <!-- e.g., 1.24.3 -->
- **Application Version**: <!-- e.g., commit hash or release version -->
- **Deployment Method**: <!-- e.g., Docker, binary, AWS ECS -->

## Steps to Reproduce
<!-- Steps to reproduce the behavior -->
1. 
2. 
3. 
4. 

## Expected Behavior
<!-- A clear and concise description of what you expected to happen -->

## Actual Behavior
<!-- A clear and concise description of what actually happened -->

## Error Messages/Logs
<!-- If applicable, add error messages or log output -->
```
# Paste error messages or relevant log entries here
```

## Configuration
<!-- If applicable, share relevant configuration (remove sensitive data) -->
```json
{
  "example": "configuration"
}
```

## Request/Response Examples
<!-- If applicable, provide example requests and responses -->
```bash
# Example request
curl -X POST http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "test", "messages": [{"role": "user", "content": "Hello"}]}'
```

```json
// Response or error
{
  "error": "example error"
}
```

## Screenshots
<!-- If applicable, add screenshots to help explain your problem -->

## Additional Context
<!-- Add any other context about the problem here -->

## Possible Solution
<!-- If you have ideas on how to fix this, please share -->

## Impact
<!-- How does this bug affect your use of the application? -->
- [ ] Blocks core functionality
- [ ] Causes data loss
- [ ] Performance degradation
- [ ] Minor inconvenience
- [ ] Other: ___________

## Workaround
<!-- If you found a temporary workaround, please describe it -->

---

## For Maintainers
### Triage Checklist
- [ ] Bug confirmed and reproducible
- [ ] Severity assessed
- [ ] Labels applied
- [ ] Milestone assigned (if applicable)
- [ ] Related issues linked 