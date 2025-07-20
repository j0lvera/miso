# Writing Effective Review Guides

This document provides best practices for creating review guides that help AI reviewers provide valuable, actionable feedback on your code.

## Guide Structure

### Basic Template

```markdown
# [Guide Name]

Brief description of what this guide covers and when it applies.

## Key Areas to Review

### [Area 1]
- Specific things to look for
- Common issues to flag
- Best practices to enforce

### [Area 2]
- More specific guidance
- Examples of good/bad patterns

## Common Issues

### [Issue Type]
**Problem**: Description of the issue
**Solution**: How to fix it
**Example**: Code example if helpful

## Review Checklist

- [ ] Specific item to check
- [ ] Another item to verify
- [ ] Final validation point
```

### Guide Types

**General Guides**: Apply to broad categories (e.g., `go.md`, `security.md`)
**Specific Guides**: Target narrow use cases (e.g., `database-migrations.md`, `react-hooks.md`)
**Diff Guides**: Focus on change impact (e.g., `diff/breaking-changes.md`)

## Writing Principles

### 1. Be Specific and Actionable

❌ **Vague**: "Check for security issues"
✅ **Specific**: "Verify that user inputs are validated and sanitized before database queries"

❌ **Generic**: "Follow best practices"
✅ **Actionable**: "Use meaningful variable names that describe the data they contain"

### 2. Focus on Common Issues

Prioritize the most frequent problems in your codebase:

```markdown
## Common React Issues

### Missing Key Props
**Problem**: List items without unique keys cause rendering issues
**Look for**: `.map()` calls without `key` prop
**Solution**: Add unique `key={item.id}` to each list item

### Uncontrolled Components
**Problem**: Form inputs without controlled state
**Look for**: `<input>` without `value` and `onChange`
**Solution**: Add state management for form inputs
```

### 3. Provide Context

Help the AI understand why something matters:

```markdown
### Database Query Optimization

**Why it matters**: Unoptimized queries can cause performance bottlenecks in production

**What to check**:
- N+1 query patterns in loops
- Missing database indexes for frequent lookups
- Large result sets without pagination

**Red flags**:
- Queries inside loops
- `SELECT *` on large tables
- Missing `LIMIT` clauses
```

### 4. Use Examples

Show concrete examples of good and bad patterns:

```markdown
### Error Handling

**Bad**:
```go
result, _ := someFunction()
// Ignoring errors
```

**Good**:
```go
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to process: %w", err)
}
```
```

## Guide Categories

### Language-Specific Guides

Focus on language idioms and best practices:

- `go.md` - Go-specific patterns, error handling, goroutines
- `typescript.md` - Type safety, interface design, generics
- `python.md` - Pythonic code, type hints, async patterns

### Domain-Specific Guides

Target specific problem areas:

- `security.md` - Authentication, authorization, input validation
- `performance.md` - Optimization patterns, caching, profiling
- `testing.md` - Test structure, coverage, mocking

### Framework-Specific Guides

Address framework conventions:

- `react.md` - Component patterns, hooks, state management
- `express.md` - Middleware, routing, error handling
- `django.md` - Model design, view patterns, security

### Diff-Specific Guides

Focus on change impact (place in `guides/diff/`):

- `breaking-changes.md` - API changes, schema modifications
- `security-review.md` - Security implications of changes
- `performance-impact.md` - Performance effects of modifications

## Content Strategies

### For Different File Sizes

**Small files** (`< 100 lines`):
```yaml
content_strategy: "full_file"
```

**Medium files** (`100-500 lines`):
```yaml
content_strategy: "first_lines"
lines: 100
```

**Large files** (`> 500 lines`):
```yaml
content_strategy: "smart"
content_lines: [100, 50, 50]  # first, last, random
```

### Pattern Matching Tips

**Filename patterns**:
```yaml
# Specific file types
filename: "\\.test\\.(ts|js)$"

# Directory-based
filename: "/components/"

# Multiple extensions
filename: "\\.(tsx|jsx|ts|js)$"
```

**Content patterns**:
```yaml
# Import statements
content: "import.*react"

# Database operations
content: "SELECT|INSERT|UPDATE|DELETE"

# Security-sensitive code
content: "password|token|secret|auth"
```

## Guide Organization

### Directory Structure

```
guides/
├── general/
│   ├── security.md
│   ├── performance.md
│   └── testing.md
├── languages/
│   ├── go.md
│   ├── typescript.md
│   └── python.md
├── frameworks/
│   ├── react.md
│   ├── express.md
│   └── django.md
├── domains/
│   ├── database.md
│   ├── api-design.md
│   └── frontend.md
└── diff/
    ├── breaking-changes.md
    ├── security-review.md
    └── performance-impact.md
```

### Guide Composition

Layer multiple guides for comprehensive coverage:

```yaml
patterns:
  - name: "react-components"
    filename: "\\.(tsx|jsx)$"
    context:
      - general/performance.md      # General performance guidance
      - frameworks/react.md         # React-specific patterns
      - domains/frontend.md         # Frontend best practices
    diff_context:
      - diff/breaking-changes.md    # Component API changes
      - frameworks/react.md         # React patterns for diffs
```

## Testing Your Guides

### Validation Checklist

- [ ] **Clarity**: Can a developer understand the guidance without context?
- [ ] **Actionability**: Does each point suggest specific actions?
- [ ] **Relevance**: Do the guidelines address real issues in your codebase?
- [ ] **Completeness**: Are the most important issues covered?
- [ ] **Examples**: Are there concrete code examples where helpful?

### Iterative Improvement

1. **Start simple**: Begin with basic guidelines
2. **Gather feedback**: See what issues the AI misses or flags incorrectly
3. **Refine**: Add more specific guidance based on real review results
4. **Expand**: Add new sections as you identify patterns

### Common Pitfalls

❌ **Too generic**: "Write good code"
❌ **Too verbose**: Overwhelming walls of text
❌ **Outdated**: Guidelines that don't match current practices
❌ **Contradictory**: Conflicting advice between guides
❌ **Missing context**: Rules without explaining why they matter

## Example: Complete Guide

```markdown
# React Component Guidelines

Guidelines for reviewing React components focusing on performance, maintainability, and best practices.

## Component Structure

### Props Interface
- Define clear TypeScript interfaces for all props
- Use descriptive names that indicate purpose
- Mark optional props with `?` operator
- Provide default values where appropriate

### Component Organization
- Keep components under 200 lines when possible
- Extract custom hooks for complex logic
- Separate concerns (UI vs business logic)
- Use meaningful component and variable names

## Performance Patterns

### Avoid Unnecessary Re-renders
**Problem**: Components re-rendering on every parent update
**Look for**: Missing `React.memo`, `useMemo`, or `useCallback`
**Solution**: Wrap components in `React.memo` and memoize expensive calculations

### Efficient State Management
**Problem**: Too many state variables causing frequent updates
**Look for**: Multiple `useState` calls for related data
**Solution**: Combine related state into objects or use `useReducer`

## Common Issues

### Missing Key Props
**Problem**: List rendering without unique keys
**Look for**: `.map()` without `key` prop
**Example**:
```jsx
// Bad
{items.map(item => <Item data={item} />)}

// Good  
{items.map(item => <Item key={item.id} data={item} />)}
```

### Uncontrolled Components
**Problem**: Form inputs without proper state management
**Look for**: `<input>` without `value` and `onChange`
**Solution**: Implement controlled components with state

## Review Checklist

- [ ] All props have TypeScript interfaces
- [ ] List items have unique keys
- [ ] Form inputs are controlled
- [ ] No inline object/function creation in JSX
- [ ] Custom hooks extracted for reusable logic
- [ ] Component names are descriptive and PascalCase
- [ ] No unused imports or variables
```

This guide provides specific, actionable guidance that helps AI reviewers give valuable feedback on React components.
