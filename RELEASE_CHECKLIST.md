# Release Checklist

This checklist outlines the steps to create a new release for `miso`.

## 1. Preparation

- [ ] Ensure all tests are passing on the `main` branch: `go test ./...`
- [ ] Confirm the `main` branch has all the features and fixes intended for the release.
- [ ] (Optional) Update a `CHANGELOG.md` with the changes for the new version.

## 2. Versioning

- [ ] Determine the new version number (e.g., `v0.1.0`) following [SemVer](https://semver.org/).
- [ ] Update the `version` variable in `cmd/main/main.go` to the new version number.
  ```go
  // In cmd/main/main.go
  var version = "v0.1.0" // Update this from "dev"
  ```
- [ ] Commit the version change: `git commit -m "chore: bump version to v0.1.0"`

## 3. Tagging and Release

- [ ] Create a new git tag for the version: `git tag -a v0.1.0 -m "Release v0.1.0"`
- [ ] Push the commit and the tag to GitHub. This will trigger the release workflow.
  ```bash
  git push origin main
  git push origin v0.1.0
  ```

## 4. Verification

- [ ] Go to the repository's "Releases" page on GitHub and verify that the new release was created successfully with all binaries and a changelog.
