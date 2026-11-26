# Description

Please provide a concise summary of the changes in this pull request and link any related issue.

Fixes: #(issue number)

## Checklist

- [ ] Branch is based on the latest `main`.
- [ ] `make check` (`fmt`, `build`, `vet`, `test` with `-race`) passes locally, and frontend `npm run check && npm test` passes.
- [ ] If flags, help text, or config serialization changed, characterization goldens are updated (`go test -run Characterization -update-golden .`) and committed.
- [ ] If this is your first contribution, add your name to `CONTRIBUTORS.md` (alphabetically ordered).
- [ ] Add an entry describing the change to `CHANGELOG.md`.

New to the codebase? Review [CONTRIBUTING.md](CONTRIBUTING.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) before submitting.
