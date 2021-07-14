# One Password Local provider

## Requirements

Requires installation of onepassword CLI https://1password.com/downloads/command-line/

## Configuration

To configure the onepassword local provider, we need to set

- ``OP_LOCAL_ACCOUNT``  - local account
- ``OP_VAULT`` - vault to use in 1Password (must be UUID)

## Example

Check out the code in ``example/one-password-local``

1. Add secret to 1Password and vault with the specified uuid. Example would be ``$secret.$section.$key``
2. Generate a template with the secret file referenced

```gotemplate
token: {{ secret "secret.section.key" }}
```

3. Invoke config-bob
```shell
OP_LOCAL_ACCOUNT=myaccount@company.com OP_VAULT=random-uuid-part config-bob build -t templates/exampleyaml -o output
```