# Release Process

Use this sequence when cutting a new release.

## Prerequisites

- Work from a clean branch tip you intend to release.
- Pick the release tag in advance, for example `v0.7.0`.

## Steps

1. Run release prep.

   ```sh
   make release-prep TAG=v0.7.0
   ```

   This target:
   - requires a clean working tree
   - runs `go test ./...`
   - generates the changelog entry from the latest existing tag through `HEAD`
   - stages `CHANGELOG.md`
   - creates the release prep commit

2. Review the generated changelog commit.

   If you need to edit the release notes, update `CHANGELOG.md` and amend or replace the prep commit before tagging.

3. Create the annotated tag on the release commit.

   ```sh
   git tag -a v0.7.0 -m "v0.7.0"
   ```

4. Build release artifacts.

   ```sh
   make release
   ```

5. Push the release commit and tag.

   ```sh
   git push origin main
   git push origin v0.7.0
   ```

## Notes

- `make changefile` prepends a new section to `CHANGELOG.md`.
- If the target tag already exists in `CHANGELOG.md`, `make changefile TAG=...` exits without modifying the file.
- `make release-prep TAG=...` fails if the working tree is dirty.
- If you want a custom prep commit message, override `RELEASE_COMMIT_MSG`, for example:

  ```sh
  make release-prep TAG=v0.7.0 RELEASE_COMMIT_MSG="Cut v0.7.0 release"
  ```

- The old `make tag` helper was removed because tagging is now a deliberate manual step after the changelog commit.
