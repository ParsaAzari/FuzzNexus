# Contributing to FuzzNexus

Thanks for taking the time to contribute. This guide describes the local checks
and repository conventions for FuzzNexus.

## Before you start

- FuzzNexus requires Go 1.26 or newer.
- Build the CLI with `go build -o fuzznexus .` at the repository root.
- Run `make test` before opening a pull request. The standard suite includes the
  race detector for both Go modules and the frontend tests.
- Run `make check` for formatting, build, vet, and test checks.

## Engineering documentation

Read [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) before making a large change.
The repository also contains characterization tests for CLI help and config
output; run them when changing flags, defaults, or serialization:

```bash
go test -run Characterization -update-golden .
```

The desktop backend is a separate Go module under `fuzznexus-desktop/`, and its
frontend checks run from `fuzznexus-desktop/frontend/`.

## Submitting a pull request

- Base your branch on the latest `main` branch.
- Keep each pull request to one logical change.
- Run `make test`, and make sure `make fmt` and `make vet` are clean.
- If your change affects help, flag, or config output, regenerate the
  characterization goldens and commit them:
  `go test -run Characterization -update-golden .`
- If this is your first contribution, add your name to `CONTRIBUTORS.md`
  (the file is alphabetically ordered).
- Add a short entry describing your change to `CHANGELOG.md` under the current
  release section.

## Reporting a security issue

Please do not open a public issue for a security vulnerability in FuzzNexus.
Report it privately through the process described in
[SECURITY.md](SECURITY.md).
