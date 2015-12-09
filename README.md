# Bob renders config hierarchies

Bob helps you to render directory trees of configurations using [golangs templating engine](http://golang.org/pkg/text/template). He renders recursively over an arbitrary number of directory hierarchies executing all files as templates.

The result will be written into one target directory.

## Example call

```bash
config-bob build path/to/data.json path/to/src/dir/a path/to/src/dir/b path/to/target/dir
```

## Intergration with vault

### Running a local vault with Bobs help

### Bob supports secrets from vault

```bash
#
config-bob init-vault

```
