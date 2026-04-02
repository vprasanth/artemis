# Release Process

Use this sequence when cutting a new release.

## Prerequisites

- Work from a clean branch tip you intend to release.
- Pick the release tag in advance, for example `v0.7.0`.

## Steps

1. Verify the repo state.

   ```sh
   git status --short
   go test ./...
   ```

2. Generate the changelog entry from the latest existing tag through `HEAD`.

   ```sh
   make changefile TAG=v0.7.0
   ```

3. Review and edit `CHANGELOG.md`.

4. Commit the release prep changes.

   ```sh
   git add CHANGELOG.md RELEASE.md Makefile
   git commit -m "Prepare v0.7.0 release"
   ```

5. Create the annotated tag on that release commit.

   ```sh
   git tag -a v0.7.0 -m "v0.7.0"
   ```

6. Build release artifacts.

   ```sh
   make release
   ```

7. Push the release commit and tag.

   ```sh
   git push origin main
   git push origin v0.7.0
   ```

## Notes

- `make changefile` prepends a new section to `CHANGELOG.md`.
- If the target tag already exists in `CHANGELOG.md`, `make changefile TAG=...` exits without modifying the file.
- The old `make tag` helper was removed because tagging is now a deliberate manual step after the changelog commit.
