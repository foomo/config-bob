package providers

import (
	"fmt"
	"strings"

	"github.com/foomo/config-bob/vault"
	"github.com/pkg/errors"
)

type Vault struct {
}

//TODO: Vault Provider
func NewVaultProvider() (*Vault, error) {
	return &Vault{}, nil
}

func (v *Vault) GetSecret(path string) (value string, err error) {
	parts := strings.Split(path, ".")
	if len(parts) != 2 {
		return "", errors.New("Invalid secret format")

	}
	secretData, err := vault.Read(parts[0])
	if err != nil {
		return "", errors.Wrap(err, "could not retrieve secret")
	}
	prop := parts[1]
	s, ok := secretData[prop]
	if !ok {
		return "", fmt.Errorf("property %q is not set for secret %q %q", prop, parts[0], secretData)
	}
	return fmt.Sprint(s), nil
}
