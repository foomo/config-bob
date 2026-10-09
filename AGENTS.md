# AGENTS.md

config-bob is a single-binary Go CLI that renders template trees with data files and Vault or 1Password secrets. Its output feeds deployments, often Helm values full of production secrets, so correctness, secret hygiene and backward compatibility outweigh new features.

## Layout

- `config-bob.go`: CLI entry, one `*Command` function per subcommand, flags parsed with `flag.NewFlagSet` before the positional paths.
- `builder/`: argument parsing, data file merging, folder walk, parallel template rendering, output writing, template helpers (`template.go`), the `op` helper (`op.go`).
- `vault/`: shared Vault API client with a per-path cache, local vault start, unseal and copy (`local.go`), `vault-tree`, `vault-htpasswd`.
- `example/` and `builder/testdata/`: fixtures used by tests.

## Commands

```bash
mise install                 # vault from .mise.toml, needed for the real vault tests
make test                    # go test ./..., the same as CI
make build                   # ./config-bob
make vault-example           # build example/source-vault against the test vault
go test -race ./...
go vet ./... && GOOS=windows go vet ./...
```

Run all of them before pushing. Check the exit status of `go test` itself; a pipe into `tail` without `set -o pipefail` hides failures. The Go version comes from `go.mod`, and CI reads it from there.

## Tests

- Unit tests never need a real `vault` or `op` binary or credentials. Fake binaries go on a `t.TempDir()` `PATH`, Vault HTTP goes through `httptest`, and `vault.Dummy = true` stubs secret reads.
- Tests against a real Vault run when `vault` is on `PATH` (CI installs it with mise) and skip otherwise: `TestLocalOpenCopyWithRealVault` and `TestBuildWithTestVault`.
- `example/vault` is a committed test vault with throwaway credentials (`testVaultKey`, `testVaultToken` in `config-bob_test.go`) and secrets `secret/app` and `secret/example.com`. Open it with `build --vault-dir`, which works on a copy; `vault-local` rewrites its storage. To add a secret, unseal a copy, write it, and copy the changed `db` files back.
- Tests that depend on file modes, umask, signals or shell scripts live in `*_unix_test.go` files with `//go:build unix`.
- Set the template file mode explicitly in tests; a strict umask otherwise changes what the test checks.
- Use `assert`, not `require`, inside handler goroutines.
- A bug fix comes with a regression test that fails on `main`.
- Commands call `os.Exit`, so `config-bob_test.go` re-runs the test binary as the CLI via `CONFIG_BOB_TEST_ARGS`.

## Compatibility contract

Downstream repositories pin a release and run config-bob in Makefiles and CI pipelines. Keep these stable, or ship an opt-out flag and a README note:

- CLI shape: `build [flags] [data files...] <source folders...> <target>`. The common call has zero or one data file and the current directory (`.`) as target, which already holds other files.
- Exit codes: any template, data or secret error exits non-zero and must not leave a half-written success behind. Pipelines gate deploys on `|| exit 1`. Nothing parses stdout or stderr, so log wording may change.
- `version` exits 0; installers use it as a probe.
- Helper names and argument order, especially `secret "path.prop"` (hundreds of call sites), `indent (yaml .x) "    "`, `join .list ","`. `secret` splits on the last dot and accepts a leading slash.
- Byte-identical output, including file modes, for the same templates and data across releases.
- Release artifacts: binary `config-bob` at the archive root, assets named `config-bob_<version>_<os>_<arch>.tar.gz` (Windows `.zip`), tags `v<version>`. Installers use mise's GitHub backends, the Homebrew cask, and plain `curl` of the asset URL.

For changes to rendering, merging or the walk, build the old and new binary, render a real template tree with each (use `check --dummy-secrets` or a throwaway vault with placeholder values), and diff the trees including modes.

## Security invariants

- Never persist Vault keys or tokens. They come from `CFB_KEYS`/`CFB_TOKEN`, the `VAULT_*` environment or the Vault token helper, or a prompt.
- Never print secret values. Errors name templates, secret paths and property names only.
- Writes stay inside the target folder through `os.Root`. Files are written to an owner-only temp file and renamed into place.
- Output folders are `0700`. Output files keep the template mode minus group and other write (`&^ 0o022`). Files are not forced to `0600` because services may read them as another user.
- Template rendering and Vault reads are bounded (`errgroup` limit, `readConcurrency`); keep new fan-out bounded.
- A new dependency needs a reason in the PR; the Vault API client already pulls in many transitive modules.

## Git and PRs

- Branches `feature/<topic>`. Commits `type: description` (`feat fix perf test ci build docs chore refactor`), imperative, no trailing period.
- PR titles in Title Case without a type prefix. PR bodies cover what, why, tests, and what was not tested.
- Merge with a merge commit after CI (`test`, CodeQL) passes. Update a branch by merging `origin/main` into it; never force-push.
- GitHub Actions are pinned by commit SHA with the version in a comment, and workflows default to `permissions: contents: read`.

## Releases

1. Release Drafter keeps a draft release named `v<next>` with the merged PRs.
2. Publishing the draft creates the tag and runs `.github/workflows/release.yml`.
3. goreleaser builds the assets and pushes the cask to `foomo/homebrew-config-bob` with a GitHub App token scoped to that repository; `GITHUB_TOKEN` cannot push there.
4. Check that the release has all assets, then downstream repositories bump their pins.

Validate `.goreleaser.yml` changes locally with `goreleaser release --snapshot --clean`. Behavior changes that can alter output need a README note with the opt-out and a line in the release notes.

## Documentation

`README.md` is the user reference: install, usage, helpers, secrets. Update it in the same PR as any CLI, flag or helper change. Keep this file for contributor and agent rules.
