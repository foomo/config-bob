[![Travis CI](https://travis-ci.org/foomo/config-bob.svg?branch=master)](https://travis-ci.org/foomo/config-bob)

# Bob renders config hierarchies

Bob helps you to render directory trees of configurations using [golangs templating engine](http://golang.org/pkg/text/template). He renders recursively over an arbitrary number of directory hierarchies executing all files as templates.

The result will be written into one target directory.

## Motivation / why config Bob

We needed a simple tool to populate our app configurations with data and **secrets** to run in docker environments.

## Building

```bash
config-bob build path/to/data.json path/to/src/dir/a path/to/src/dir/b path/to/target/dir
```

### Bobs templating add-ons

Apart from standard templating functions we have added a few extra ones, which should come in handy, when writing configurations:

```
// secret helpers
{{ secret secret/path/to/secret.prop }}
{{ secret_js secret/path/to/secret.prop }}
{{ secret_json secret/path/to/secret.prop }}
{{ secret_yaml secret/path/to/secret.prop }}

// others
{{ yaml_string .some.data.prop }}
```

We expect this list of helpers to grow.

## Updating htpasswd files

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

Example call:

```bash
config-bob vault-htpasswd path/to/htpasswd.yml
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

## Intergration with [vault](//vaultproject.io/)

When using the secret templating syntax metioned above Bob will be looking up those secrets in a vault server using vault http interface v1.

Bob expects the environment variables `VAULT_ADDR` and `VAULT_TOKEN` to be set to know to which vault server to talk to.

### Running a local vault with Bobs help

If you want to keep your secrets under version control and you do not want to run a vault server permanently config-bob has a little helper for you.

```bash
config-bob vault-local path/to/vault-folder
```

## Requirements

So far Bob has been running on OSX and Linux.

- [vault](//vaultproject.io) tested with Vault v0.3.1, but as long as REST API v1 is there I do not expect
