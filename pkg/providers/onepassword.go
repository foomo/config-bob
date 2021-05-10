package providers

import (
	"os"
	"strings"

	"github.com/pkg/errors"
)

const (
	OnePasswordEnvVault         = "OP_VAULT"
	OnePasswordEnvSessionPrefix = "OP_SESSION_"
	OnePasswordEnvLocalAccount  = "OP_LOCAL_ACCOUNT"
	OnePasswordEnvConnectToken  = "OP_CONNECT_TOKEN"
	OnePasswordEnvConnectHost   = "OP_CONNECT_HOST"

	OnePasswordPathSeparator      = "."
	OnePasswordConfigBobUserAgent = "ConfigBob/1.0 OnePasswordConnect Provider"
)

var (
	ErrVaultNotConfigured = errors.New("Vault not configured")
)

func parseOnePasswordPath(path string) (title, section, field string, err error) {
	parts := strings.Split(path, OnePasswordPathSeparator)
	if len(parts) != 3 {
		return "", "", "", errors.Errorf("wrong number of path parts, required %d got %d", 3, len(parts))
	}
	return parts[0], parts[1], parts[2], nil
}

func IsOnePasswordConnectConfigured() bool {
	return isEnvDefined(OnePasswordEnvVault, OnePasswordEnvConnectToken, OnePasswordEnvConnectHost)
}

func IsOnePasswordLocalConfigured() bool {
	return isEnvDefined(OnePasswordEnvVault, OnePasswordEnvLocalAccount)
}

func isEnvDefined(variables ...string) bool {
	for _, v := range variables {
		if _, ok := os.LookupEnv(v); !ok {
			return false
		}
	}
	return true
}
