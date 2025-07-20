# Breaking Changes Review Guide

When reviewing diffs, pay special attention to:

## API Changes
- Function signature modifications (parameters, return types)
- Public interface changes that could break existing code
- Removal of public methods, functions, or exports

## Database Schema Changes
- Column removals or type changes
- Index modifications that could affect performance
- Migration compatibility issues

## Configuration Changes
- Environment variable changes
- Default value modifications
- Required vs optional parameter changes

## Dependency Changes
- Major version upgrades that could introduce breaking changes
- Removal of dependencies that other code might rely on
- New required dependencies

## Review Checklist
- [ ] Are there any public API changes?
- [ ] Do schema changes have proper migrations?
- [ ] Are configuration changes backward compatible?
- [ ] Are dependency changes documented?
- [ ] Is there a migration guide for breaking changes?
