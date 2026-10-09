[![Test Branch](https://github.com/foomo/config-bob/actions/workflows/test.yml/badge.svg)](https://github.com/foomo/config-bob/actions/workflows/test.yml)

# Bob renders config hierarchies

Bob helps you to render directory trees of configurations using [golangs templating engine](http://golang.org/pkg/text/template). He renders recursively over an arbitrary number of directory hierarchies executing all files as templates.

The result will be written into one target directory.

## Motivation / why config Bob

We needed a simple tool to populate our app configurations with data and **secrets** to run in docker environments.

## Building

```bash
config-bob build path/to/data.json path/to/src/dir/a path/to/src/dir/b path/to/target/dir
```

Several data files are deep merged in the given order: nested maps merge key by key, and any other value (scalars, lists, `null`) from a later file replaces the earlier one. This lets a shared base file carry the defaults and small files carry the overrides. Merge files of the same format: a nested JSON map and a nested YAML map replace each other instead of merging.

```bash
config-bob build base.yaml stage.yaml path/to/src/dir path/to/target/dir
```

Before deep merging, a later file replaced whole top-level keys. To drop nested keys from an earlier file, now set the parent key to `null` or to a new value explicitly, or keep the old behavior with `--deep-merge=false` (flags go before the paths):

```bash
config-bob build --deep-merge=false base.yaml stage.yaml path/to/src/dir path/to/target/dir
```

A failing build reports every broken template at once, not only the first one.

## Checking

```bash
config-bob check [--dummy-secrets] [--deep-merge=false] path/to/data.json path/to/src/dir/a path/to/src/dir/b
```

`check` renders all templates in memory and writes nothing, so rendered secrets never land on disk. With `--dummy-secrets` every `secret` call renders as `dummy-secret:<path.prop>` without contacting vault, which lets CI validate template syntax and data keys without vault credentials. Without it, `check` reads vault like `build` and also proves that every referenced secret exists. `--deep-merge=false` merges data files like `build --deep-merge=false`.

- Flags go before the paths.
- A dummy secret is a placeholder string, so templates that rely on the format of a real secret value only fail in a real build.
- Files listed in `.bobcopy` are copied, not rendered, so `check` does not inspect them.

### Bobs template helpers

Apart from standard template functions we have added a few extra ones, which should come in handy, when writing configurations:

```
// secrets helpers
{{ secret "secret/path/to/secret.prop" }}

// combining secrets with escaping might come in handy
{{ json (secret "secret/path/to/secret.prop") }}
```

Data in this example

```go
data := map[string]any{
    "hello": "test",
    "nested": map[string]string{
        "foo": "bar",
    },
}
```

```
// template dump some yaml into a file
{{ yaml . }}
// output
hello: test
nested:
  foo: bar

// template indent sth - yaml in this case
{{ indent (yaml .) "  " }}
// output
  hello: test
  nested:
    foo: bar


// template json
{{ json . }}

// output
{"hello":"test","nested":{"foo":"bar"}}

// json indented parameters are prefix and indent
{{ jsonindent . "////" "+++|" }}

// output - note that there is no prefix in the first line also see https://golang.org/pkg/encoding/json/#MarshalIndent
{
////"hello": "test",
////+++|"nested": {
////+++|+++|"foo": "bar"
////+++|}
////}

// template substr, which is essentially string slice access
{{ substr .hello ":2"}}`
// output
te

{{ substr .hello "1:"}}`
// output
est

{{ substr .hello "1:2"}}`
// output
e

```

We expect this list of helpers to grow.

## Updating htpasswd files

```bash
config-bob vault-htpasswd path/to/htpasswd.yml
```

Config bob knows how to sync vault with htpasswd files.

Example config file contents:

```yaml
# example htpasswd.yml
relative/path/to/htpasswd-file:
  - secret/foo
  - secret/bar
/absolute/path/to/other/htpasswd-file:
  - secret/baz
```

Behaviour:

- creates all necessary folder and files
- updates existing files with passwords from vault
- fails, if passwords can not be updated
- fails, if existing files can not be parsed

How to add a compatible vault entry:

```bash
vault write secret/foo user=foo password=secret
```

## Intergration with [vault](https://vaultproject.io/)

When using the secret templating syntax metioned above Bob will be looking up those secrets in a vault server using vault http interface v1.

Bob expects the environment variables `VAULT_ADDR` and `VAULT_TOKEN` to be set to know to which vault server to talk to.

### Running a local vault with Bobs help

If you want to keep your secrets under version control and you do not want to run a vault server permanently config-bob has a little helper for you.

```bash
config-bob vault-local path/to/vault-folder
```

Bob asks for the unseal keys and the token on every start and never stores them. To skip the prompts set `CFB_KEYS` (comma separated) and `CFB_TOKEN`.

Older versions saved the token and unseal keys in plain text in `~/.cfb/vault-store.json`. Delete that file.

To build against such a vault in one step, for example in CI, pass its folder to `build`:

```bash
# CFB_KEYS and CFB_TOKEN injected by CI, flags go before the paths
config-bob build --vault-dir path/to/vault-folder path/to/src/dir path/to/target/dir
```

Bob starts the vault on a temporary copy of the folder's `db`, unseals it, builds, and stops it again, also when the build fails or is interrupted. The committed vault storage is never rewritten. Keys and token come from `CFB_KEYS` and `CFB_TOKEN`, or an interactive prompt.

The copy runs with Bob's own server config instead of the folder's `config.hcl`: file storage in `db`, plain HTTP on a random `127.0.0.1` port, and mlock disabled. Bob points `VAULT_ADDR` of the build at that port, so a vault already running on `8200` and an inherited `VAULT_ADDR` do not get in the way. This covers vaults with file storage in `db` that unseal with keys; other `config.hcl` settings, such as auto-unseal `seal` blocks, are not used.

## Integration with 1Password

We have added a template helper to get fields from 1Password

```yaml
secret-from-1password: {{ op "name-uuid-or-url-of-entry" "field-name" }}
```

In order to make this work follow this document [https://support.1password.com/command-line-getting-started/](https://support.1password.com/command-line-getting-started/)

## Requirements

So far Bob has been running on OSX and Linux.

- [vault](https://vaultproject.io) tested with Vault v0.3.1, but as long as REST API v1 is there I do not expect

