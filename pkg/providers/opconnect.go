package providers

import (
	"os"
	"strings"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/pkg/errors"
)

type OnePasswordConnect struct {
	client connect.Client
	vault  string
}

func NewOnePasswordConnectFromEnv() (OnePasswordConnect, error) {
	vault, ok := os.LookupEnv(OnePasswordEnvVault)
	if !ok {
		return OnePasswordConnect{}, errors.Wrapf(ErrVaultNotConfigured, "%q is missing", OnePasswordEnvVault)
	}
	token, ok := os.LookupEnv(OnePasswordEnvConnectToken)
	if !ok {
		return OnePasswordConnect{}, errors.Wrapf(ErrVaultNotConfigured, "%q is missing", OnePasswordEnvConnectToken)
	}

	host, ok := os.LookupEnv(OnePasswordEnvConnectHost)
	if !ok {
		return OnePasswordConnect{}, errors.Wrapf(ErrVaultNotConfigured, "%q is missing", OnePasswordEnvConnectHost)
	}

	return NewOnePasswordConnectProvider(host, token, vault)
}

func NewOnePasswordConnectProvider(host, token, vault string) (OnePasswordConnect, error) {
	client := connect.NewClientWithUserAgent(host, token, OnePasswordConfigBobUserAgent)

	// Check  connection & token
	vaults, err := client.GetVaults()
	if err != nil {
		return OnePasswordConnect{}, errors.Wrap(err, "failed to initialize provider")
	}

	var vaultID string
	for _, v := range vaults {
		if v.ID == vault || v.Name == vault {
			vaultID = v.ID
			break
		}
	}

	if vaultID == "" {
		return OnePasswordConnect{}, errors.Errorf("vault with id %q not found", vault)
	}

	return OnePasswordConnect{
		client: client,
		vault:  vaultID,
	}, nil
}

// Path must be in format title/section/label
func (op OnePasswordConnect) GetSecret(path string) (string, error) {
	title, section, field, err := parseOnePasswordPath(path)
	if err != nil {
		return "", err
	}

	eq := func(a, b string) bool {
		return strings.ToLower(a) == strings.ToLower(b)
	}

	item, err := op.client.GetItemByTitle(title, op.vault)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get item for title %q in vault %q", title, op.vault)
	}

	for _, f := range item.Fields {
		if eq(f.Label, field) && eq(item.SectionLabelForID(f.Section.ID), section) {
			return f.Value, nil
		}
	}
	return "", errors.Errorf("failed to find %q in section %q inside %q in vault %q", field, section, title, op.vault)
}
