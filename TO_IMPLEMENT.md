# TO_IMPLEMENT - Remaining Work

This document outlines the remaining implementation tasks for the go-review refactoring project.

## Batch 3.1: Create Diff-Specific Logic (Tests Remaining)

### Status: Implementation Complete, Tests Needed

The diff-specific logic has been implemented but requires comprehensive testing:

#### Tests to Write:

1. **`internal/git/diff_parser_test.go`**
   - Test `ParseDiff()` with various diff formats
   - Test edge cases (empty diffs, binary files, renames)
   - Test `DiffData` methods (`GetAddedLines`, `GetRemovedLines`, etc.)
   - Test `FormatForReview()` output

2. **`internal/prompts/diff-review_test.go`**
   - Test `DiffReview()` prompt generation
   - Test guide resolution for diff reviews
   - Test fallback to regular guides when no diff guides exist
   - Test `getDefaultLegacyConfig()` function

3. **`internal/agents/reviewer_test.go`** (update existing)
   - Test new `ReviewDiff()` method
   - Test `callLLM()` helper method
   - Mock LLM responses for testing

4. **`internal/git/git_test.go`** (update existing)
   - Test `GetFileDiffData()` method
   - Test diff parsing integration

## Batch 3.2: Integrate Diff with Patterns

### Goal: Add `diff_context` support to configuration

#### Implementation Tasks:

1. **Update Configuration Examples**
   - Add example `go-review.yml` with `diff_context` patterns
   - Document how `diff_context` differs from regular `context`

2. **Create Diff-Specific Guides**
   - Create guides focused on reviewing changes:
     - `guides/diff/breaking-changes.md`
     - `guides/diff/security-review.md`
     - `guides/diff/performance-impact.md`
   - Update existing guides to have diff-specific versions

3. **Enhance Pattern Matching for Diffs**
   - Add ability to match patterns based on diff content (not just file content)
   - Example: Match if diff adds certain imports or removes certain functions

4. **Update Documentation**
   - Update README with diff review examples
   - Show how to configure diff-specific patterns

## Phase 4: Polish & Documentation

### Batch 4.1: CLI Enhancements

#### Implementation Tasks:

1. **Add `--config` Flag**
   ```go
   type CLI struct {
       Config string `short:"c" help:"Path to config file" type:"existingfile"`
       // ... existing fields
   }
   ```

2. **Add `validate-config` Command**
   ```go
   type ValidateConfigCmd struct {
       Config string `arg:"" help:"Path to config file to validate"`
   }
   ```
   - Validate YAML syntax
   - Validate regex patterns
   - Check if referenced guide files exist
   - Report all issues found

3. **Add Pattern Testing Command**
   ```go
   type TestPatternCmd struct {
       File    string `arg:"" help:"File to test against patterns"`
       Config  string `short:"c" help:"Path to config file"`
       Verbose bool   `short:"v" help:"Show detailed matching info"`
   }
   ```
   - Show which patterns match a given file
   - Display which guides would be loaded
   - Useful for debugging pattern configuration

4. **Add `--dry-run` Flag**
   - For both `review` and `diff` commands
   - Shows what would be reviewed without calling LLM
   - Displays matched patterns and selected guides

### Batch 4.2: Final Documentation

#### Documentation Tasks:

1. **Migration Guide**
   - Create `MIGRATION.md`
   - Step-by-step guide from old router to new config
   - Example configurations for common scenarios

2. **Configuration Examples**
   - Create `examples/` directory with:
     - `examples/react-typescript.yml`
     - `examples/golang-backend.yml`
     - `examples/python-ml.yml`
     - `examples/multi-language.yml`

3. **Guide Writing Best Practices**
   - Create `guides/WRITING_GUIDES.md`
   - Tips for effective review guides
   - Difference between full review and diff review guides

4. **API Documentation**
   - Document all public functions and types
   - Add godoc comments throughout codebase
   - Generate and publish API docs

5. **Performance Optimization Guide**
   - Document content scanning strategies
   - Benchmarks for different strategies
   - Recommendations for large codebases

6. **GitHub Actions Integration**
   - Create `.github/workflows/example.yml`
   - Document environment variables
   - Show PR comment integration
   - Add status check configuration

## Testing Strategy

### Integration Tests
- Create `test/integration/` directory
- Test full review workflow with sample files
- Test diff review with git repositories
- Test configuration loading and pattern matching

### Benchmarks
- Benchmark pattern matching performance
- Compare content scanning strategies
- Measure memory usage for large files

### Example Repository
- Create `examples/sample-project/`
- Include various file types
- Include `.go-review.yml` configuration
- Demonstrate real-world usage

## Release Checklist

- [ ] All tests passing
- [ ] Documentation complete
- [ ] Examples provided
- [ ] Migration guide written
- [ ] Performance benchmarks documented
- [ ] GitHub Action published
- [ ] Release notes prepared
