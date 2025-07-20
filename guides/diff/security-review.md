# Security-Focused Diff Review

Focus on security implications of code changes:

## Authentication & Authorization
- New authentication mechanisms
- Permission checks being added or removed
- Session handling changes

## Input Validation
- New user inputs without proper validation
- Removal of existing validation
- SQL injection prevention

## Data Exposure
- New API endpoints exposing sensitive data
- Logging changes that might leak secrets
- Error messages revealing internal information

## Dependencies
- New dependencies with known vulnerabilities
- Outdated packages being introduced

## Secrets Management
- Hardcoded secrets or credentials
- Improper secret storage
- Environment variable handling

## Review Focus
- Look for added `+` lines that introduce security risks
- Check removed `-` lines for security controls being disabled
- Verify new code follows security best practices
