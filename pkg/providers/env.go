package providers

import (
	"fmt"
	"os"
)

const (
	OnePasswordEnvVault         = "OP_VAULT"
	OnePasswordEnvSessionPrefix = "OP_SESSION_"
	OnePasswordEnvLocalAccount  = "OP_LOCAL_ACCOUNT"
	OnePasswordEnvConnectToken  = "OP_CONNECT_TOKEN"
	OnePasswordEnvConnectHost   = "OP_CONNECT_HOST"

	OnePasswordPathSeparator      = "."
	OnePasswordConfigBobUserAgent = "ConfigBob/1.0 OnePasswordConnect Provider"

	GoogleSecretsAccountCredentials = "GCP_APPLICATION_CREDENTIALS"
	GoogleSecretsProject            = "GCP_PROJECT"
)

func IsOnePasswordConnectConfigured() bool {
	return isEnvDefined(OnePasswordEnvVault, OnePasswordEnvConnectToken, OnePasswordEnvConnectHost)
}

func IsOnePasswordLocalConfigured() bool {
	return isEnvDefined(OnePasswordEnvVault, OnePasswordEnvLocalAccount)
}

func IsGoogleSecretsConfigured() bool {
	return isEnvDefined(GoogleSecretsAccountCredentials, GoogleSecretsProject)
}

func LookupEnv(key string) (value string, err error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("env variable %q is not defined", key)
	}
	return value, nil
}

func isEnvDefined(variables ...string) bool {
	for _, v := range variables {
		if _, ok := os.LookupEnv(v); !ok {
			return false
		}
	}
	return true
}
