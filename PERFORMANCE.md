# Performance Optimization Guide

This guide explains the different content scanning strategies available in go-review and how to optimize performance for different file sizes and project structures.

## Content Scanning Strategies

go-review supports three content scanning strategies to balance review quality with performance:

### 1. First Lines Strategy (`first_lines`)

**Best for**: Small to medium files (< 500 lines)

Scans only the first N lines of a file. This is the fastest strategy and works well for files where important patterns (imports, package declarations, etc.) appear at the top.

```yaml
content_defaults:
  strategy: "first_lines"
  lines: 50  # Scan first 50 lines
```

**Performance**: ⭐⭐⭐⭐⭐ (Fastest)
**Coverage**: ⭐⭐ (Limited to file beginning)

**Use cases**:
- Checking import statements
- Package/module declarations
- File headers and license checks
- Quick pattern matching

### 2. Full File Strategy (`full_file`)

**Best for**: Small files (< 200 lines) or when complete analysis is required

Scans the entire file content. Provides the most comprehensive analysis but can be slow for large files.

```yaml
content_defaults:
  strategy: "full_file"
```

**Performance**: ⭐ (Slowest for large files)
**Coverage**: ⭐⭐⭐⭐⭐ (Complete)

**Use cases**:
- Configuration files
- Small utility files
- Critical security-sensitive files
- Complete code analysis

### 3. Smart Strategy (`smart`)

**Best for**: Large files (> 500 lines) where you need broader coverage

Intelligently samples content from the beginning, end, and random sections of the file. Provides good coverage while maintaining reasonable performance.

```yaml
content_defaults:
  strategy: "smart"
  
# Per-pattern override
patterns:
  - name: "large-files"
    filename: "\\.(go|ts|js)$"
    content_strategy: "smart"
    content_lines: [100, 50, 50]  # [first_lines, last_lines, random_lines]
```

**Performance**: ⭐⭐⭐ (Balanced)
**Coverage**: ⭐⭐⭐⭐ (Good sampling)

**Use cases**:
- Large source files
- Files with important code at the end (cleanup, exports)
- Balanced performance/coverage needs

## Strategy Selection Guidelines

### By File Size

| File Size | Recommended Strategy | Reasoning |
|-----------|---------------------|-----------|
| < 100 lines | `full_file` | Fast enough for complete analysis |
| 100-500 lines | `first_lines` (100-200 lines) | Good balance, most patterns at top |
| 500-2000 lines | `smart` [200, 100, 100] | Broader coverage needed |
| > 2000 lines | `smart` [300, 200, 200] | Large sampling for big files |

### By File Type

```yaml
patterns:
  # Configuration files - need complete analysis
  - name: "config-files"
    filename: "\\.(yml|yaml|json|toml)$"
    content_strategy: "full_file"
  
  # Source files - smart sampling
  - name: "source-files"
    filename: "\\.(go|ts|js|py)$"
    content_strategy: "smart"
    content_lines: [150, 100, 100]
  
  # Test files - focus on imports and setup
  - name: "test-files"
    filename: "_test\\.(go|ts|js)$"
    content_strategy: "first_lines"
    content_lines: [200]
  
  # Documentation - first lines for metadata
  - name: "docs"
    filename: "\\.md$"
    content_strategy: "first_lines"
    content_lines: [50]
```

## Performance Optimization Tips

### 1. Use Pattern Ordering Strategically

Place more specific, commonly-matched patterns first with `stop: true`:

```yaml
patterns:
  # Fast filename-only patterns first
  - name: "test-files"
    filename: "_test\\.(go|ts|js)$"
    context: ["testing.md"]
    stop: true  # Don't check other patterns
  
  # More expensive content patterns later
  - name: "database-code"
    content: "SELECT|INSERT|UPDATE|DELETE"
    content_strategy: "smart"
    context: ["database.md"]
```

### 2. Minimize Content Scanning

Use filename patterns when possible to avoid reading file contents:

```yaml
# ✅ Fast - filename only
- name: "react-components"
  filename: "/components/.*\\.(tsx|jsx)$"
  context: ["react.md"]

# ❌ Slower - requires content scanning
- name: "react-components"
  content: "import.*react"
  context: ["react.md"]
```

### 3. Optimize Content Line Counts

For `smart` strategy, balance coverage with performance:

```yaml
# ✅ Balanced for most files
content_lines: [100, 50, 50]  # 200 lines total

# ❌ Too aggressive for large files
content_lines: [500, 500, 500]  # 1500 lines total

# ✅ Conservative for very large files
content_lines: [200, 100, 50]  # 350 lines total
```

### 4. Use Regex Efficiently

Optimize regular expressions for performance:

```yaml
# ✅ Specific and fast
filename: "\\.test\\.(ts|js)$"

# ❌ Too broad, slower
filename: "test"

# ✅ Anchored patterns
content: "^import.*react"

# ❌ Unanchored, searches entire content
content: "react"
```

## Memory Considerations

### Large File Handling

For projects with very large files, consider:

1. **Line-by-line scanning** for memory efficiency:
   ```go
   // Use ScanFileLines for large files
   patterns, err := matcher.ScanFileLines(filename, 1000)
   ```

2. **Streaming content strategies**:
   ```yaml
   # Limit content scanning for huge files
   - name: "large-generated-files"
     filename: "\\.generated\\."
     content_strategy: "first_lines"
     content_lines: [50]  # Very conservative
   ```

### Repository Size Optimization

For large repositories:

1. **Skip generated files**:
   ```yaml
   # Don't review generated or vendor files
   - name: "skip-generated"
     filename: "(vendor/|node_modules/|\\.generated\\.|_pb\\.)"
     context: []  # No guides = skip review
     stop: true
   ```

2. **Use directory-based patterns**:
   ```yaml
   # Target specific directories
   - name: "src-files"
     filename: "^src/.*\\.(ts|js)$"
     content_strategy: "smart"
   ```

## Benchmarking Your Configuration

### Measuring Performance

Test your configuration performance:

```bash
# Time pattern matching
time go-review test-pattern large-file.go -v

# Profile full reviews
time go-review review large-file.go --dry-run
```

### Performance Metrics

Monitor these metrics:

- **Pattern matching time**: Should be < 100ms per file
- **Content scanning time**: Should be < 500ms for large files
- **Memory usage**: Should stay under 100MB for typical files

### Optimization Checklist

- [ ] Use filename patterns when possible
- [ ] Order patterns by frequency (most common first)
- [ ] Use `stop: true` for exclusive patterns
- [ ] Choose appropriate content strategies by file size
- [ ] Optimize regex patterns for speed
- [ ] Skip generated/vendor files
- [ ] Test with your largest files
- [ ] Monitor memory usage with large repositories

## Example Optimized Configuration

```yaml
content_defaults:
  strategy: "first_lines"
  lines: 100

patterns:
  # Fast filename-only patterns first
  - name: "skip-generated"
    filename: "(vendor/|node_modules/|\\.generated\\.|dist/|build/)"
    context: []
    stop: true
  
  - name: "test-files"
    filename: "\\.(test|spec)\\.(ts|js|go)$"
    context: ["testing.md"]
    stop: true
  
  # Medium priority with content scanning
  - name: "small-config"
    filename: "\\.(yml|yaml|json)$"
    content_strategy: "full_file"
    context: ["config.md"]
  
  - name: "source-files-small"
    filename: "\\.(ts|js|go)$"
    content_strategy: "first_lines"
    content_lines: [150]
    context: ["general.md"]
  
  # Expensive patterns last
  - name: "large-source-files"
    filename: "\\.(ts|js|go)$"
    content: "class|interface|func|export"
    content_strategy: "smart"
    content_lines: [200, 100, 100]
    context: ["architecture.md"]
```

This configuration prioritizes speed while maintaining good coverage for code review quality.
