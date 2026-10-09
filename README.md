[![Test Branch](https://github.com/foomo/config-bob/actions/workflows/test.yml/badge.svg)](https://github.com/foomo/config-bob/actions/workflows/test.yml)

# config-bob

Render directory trees of configuration templates with data files and secrets from [Vault](https://www.vaultproject.io/) or 1Password.

Bob walks one or more source folders, executes every file as a Go [`text/template`](https://pkg.go.dev/text/template), and writes the results into one target folder. Use it to give apps and Helm charts their configuration and secrets without committing rendered secrets.

## Install

```bash
# Homebrew (macOS, Linux)
brew install --cask foomo/config-bob/config-bob

# Go
go install github.com/foomo/config-bob@latest
```

Binaries for macOS and Linux (amd64, arm64) and Windows (amd64) are attached to every [release](https://github.com/foomo/config-bob/releases). Tags carry a `v` prefix, asset names do not:

```bash
curl -fsSL https://github.com/foomo/config-bob/releases/download/v0.9.0/config-bob_0.9.0_linux_amd64.tar.gz | tar -xz config-bob
```

## Usage

```bash
config-bob build [flags] [data files...] <source folder>... <target folder>
config-bob check [flags] [data files...] <source folder>...
config-bob vault-local <vault folder> [script args...]
config-bob vault-tree <path in vault>
config-bob vault-htpasswd <htpasswd.yaml>
config-bob version
```

Flags go before the paths. Arguments ending in `.json`, `.yml` or `.yaml` are data files, folders are source folders, and the last argument of `build` is the target folder.

### build

```bash
config-bob build base.yaml stage.yaml templates/common templates/stage out
```

- Later source folders overwrite files of earlier ones with the same relative path.
- Data files are deep merged in order: nested maps merge key by key, any other value (scalars, lists, `null`) from a later file replaces the earlier one. Merge files of the same format; a nested JSON map and a nested YAML map replace each other.
- `--deep-merge=false` restores the old behavior, where a later file replaced whole top-level keys.
- `--vault-dir <folder>` builds against a local vault in one step, see [Local vault](#local-vault).
- A failing build reports every broken template at once and exits non-zero.

Output folders are created `0700`. Output files keep the template's mode without group or other write access, so make templates that hold secrets `chmod 600`.

### check

```bash
config-bob check --dummy-secrets base.yaml stage.yaml templates/common
```

Renders every template in memory and writes nothing. With `--dummy-secrets`, each `secret` call renders as `dummy-secret:<path.prop>` without contacting Vault, so CI can validate template syntax and data keys without credentials. Without it, `check` reads Vault like `build` and proves every referenced secret exists. Files listed in `.bobcopy` are not checked.

### Source folder control files

| File         | Effect                                                                 |
|--------------|------------------------------------------------------------------------|
| `.bobignore` | Paths relative to the source folder, one per line, that are skipped.   |
| `.bobcopy`   | Files or folders, one per line, copied as is instead of being rendered. |

## Templates

Templates run with `missingkey=error`, so a missing data key fails the build. All [built-in functions](https://pkg.go.dev/text/template#hdr-Functions) are available, plus:

| Helper       | Example                                     | Result                                                   |
|--------------|---------------------------------------------|----------------------------------------------------------|
| `secret`     | `{{ secret "secret/db.password" }}`         | Property after the last dot of a Vault secret            |
| `op`         | `{{ op "item-name-or-id" "password" }}`     | Field of a 1Password item via the `op` CLI               |
| `env`        | `{{ env "HOME" }}`                          | Environment variable, fails when empty                   |
| `yaml`       | `{{ yaml .resources }}`                     | Value as YAML                                            |
| `json`       | `{{ json (secret "secret/db.password") }}`  | Value as JSON, also useful for quoting strings           |
| `jsonindent` | `{{ jsonindent . "" "  " }}`                | Value as indented JSON (prefix, indent)                  |
| `indent`     | `{{ indent (yaml .resources) "    " }}`     | Every line prefixed                                      |
| `join`       | `{{ join .hosts "," }}`                     | List joined with a separator                             |
| `replace`    | `{{ .host \| replace "." "-" }}`            | All occurrences replaced                                 |
| `substr`     | `{{ substr .name "1:3" }}`                  | Byte slice `[start:end]`, either side may be empty       |
| `jsescape`   | `{{ jsescape .text }}`                      | String escaped for JavaScript                            |
| `absPath`    | `{{ absPath "relative/path" }}`             | Absolute path                                            |

## Secrets

### Vault

`secret` reads KV v1 style secrets whose values are strings. Bob configures its client like the Vault CLI: `VAULT_ADDR`, `VAULT_TOKEN` and the other `VAULT_*` variables, falling back to the Vault token helper. Each secret path is read once per run.

`config-bob vault-tree secret` lists all secrets below a path.

### Local vault

Keep an encrypted, file-backed vault next to your templates and start it only when needed. Requires the `vault` binary on `PATH`.

```bash
config-bob vault-local path/to/vault-folder
```

Bob creates the folder layout on first use, starts Vault on `127.0.0.1:8200`, unseals it, and opens a login shell (or runs the given script) with `VAULT_ADDR` and `VAULT_TOKEN` set. It stops Vault when the shell exits. Unseal keys and token are prompted for and never stored; set `CFB_KEYS` (comma separated) and `CFB_TOKEN` to skip the prompts.

To build in one step, for example in CI:

```bash
CFB_KEYS=... CFB_TOKEN=... config-bob build --vault-dir path/to/vault-folder templates out
```

Bob serves a temporary copy of the folder's `db` on a random loopback port, builds, and stops Vault again, also when the build fails or is interrupted. The committed storage is never rewritten. The folder's `config.hcl` is not used, so this covers file-storage vaults unsealed with keys, not auto-unseal setups.

Older versions stored the keys and token in plain text in `~/.cfb/vault-store.json`. Delete that file.

### 1Password

`op` calls `op item get <item> --fields <field> --reveal`, so the [1Password CLI](https://developer.1password.com/docs/cli/) must be installed and signed in.

### htpasswd files

`vault-htpasswd` writes bcrypt htpasswd files from Vault secrets that have `user` and `password` properties:

```yaml
# htpasswd.yaml: htpasswd file -> vault paths
relative/path/to/htpasswd: [secret/foo, secret/bar]
/absolute/path/to/htpasswd: [secret/baz]
```

```bash
vault write secret/foo user=foo password=secret
config-bob vault-htpasswd htpasswd.yaml
```

## Development

```bash
make test   # go test ./...
make build  # ./config-bob
```

See [AGENTS.md](AGENTS.md) for conventions, compatibility constraints and the release process.

## License

[MPL-2.0](LICENSE)
