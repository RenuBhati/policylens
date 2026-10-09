# Contributing

Bug reports, fixes and improvements are welcome. For a bug report, include the command or steps you ran, expected behavior, actual behavior and relevant versions. Use a minimal manifest with sensitive values removed.

## Local setup

```sh
make setup
make run
```

Before submitting a pull request, run:

```sh
make test vet ui build
make eval
```

Format Go changes with `gofmt`. Add tests for behavior changes and describe how the change was verified in the pull request. Model integration tests use test doubles; `scripts/evaluate-ai.py` makes real provider calls and is optional.

Bundled policy files are kept identical to their upstream versions. When updating a policy, preserve its license and update the revision and SHA-256 values in `internal/policy/sources/manifest.json`.
