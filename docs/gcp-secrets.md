# GCP Secret Provider

## Requirements

- Requires GCP Account with billing
- Requires that ``secretmanager.googleapis.com`` is enabled on the specified project
- Requires that the specified service account has `secretmanager.versions.access` permissions for the specified secrets

## Configuration

To use the GCP secret provider in the vault configuration you must set the correct env variables
- ``GCP_APPLICATION_CREDENTIALS`` to point to the correct GCP account 
- ``GCP_PROJECT`` where the secrets reside.

The project used for secrets must have ``secretmanager.googleapis.com`` by running the command

```
gcloud services enable secretmanager.googleapis.com --project=MYPROJECTID
```

https://cloud.google.com/endpoints/docs/openapi/enable-api

The account which will access the secrets needs `secretmanager.versions.access` permission or `Secret Manager Secret Accessor` role.

You can use https://console.cloud.google.com/security/secret-manager?project=my-project to access the available secrets.

## Example

Check out the code in ``example/gcp-secrets``

1. Add secret to render

```shell
printf "s3cr3t" | gcloud secrets create my-secret --data-file=-
```

2. Create a template such as

```gotemplate
token: {{ secret "my-secret" }}
```

3. Invoke config-bob

```shell
GOOGLE_APPLICATION_CREDENTIALS=./credentials.json GOOGLE_PROJECT=my-project config-bob build -t templates/exampleyaml -o output 
```