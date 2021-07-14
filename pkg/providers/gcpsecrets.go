package providers

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"

	"google.golang.org/api/option"
)

type GoogleSecrets struct {
	project string
	client  *secretmanager.Client
}

func NewGoogleSecretsProviderFromEnv() (GoogleSecrets, error) {
	credentials, err := LookupEnv(GoogleSecretsAccountCredentials)
	if err != nil {
		return GoogleSecrets{}, err
	}

	project, err := LookupEnv(GoogleSecretsProject)
	if err != nil {
		return GoogleSecrets{}, err
	}
	return NewGoogleSecretsProvider(credentials, project)
}

func NewGoogleSecretsProvider(credentialsFilePath, project string) (GoogleSecrets, error) {
	ctx := context.Background()

	client, err := secretmanager.NewClient(ctx,
		option.WithCredentialsFile(credentialsFilePath),
	)
	if err != nil {
		return GoogleSecrets{}, err
	}

	return GoogleSecrets{
		project: project,
		client:  client,
	}, nil
}

func (gs GoogleSecrets) Close() error {
	return gs.client.Close()
}

func (gs GoogleSecrets) GetSecret(path string) (value string, err error) {
	ctx := context.Background()
	gcpPath := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", gs.project, path)
	secret, err := gs.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: gcpPath})
	if err != nil {
		return "", err
	}
	return string(secret.GetPayload().GetData()), nil
}
