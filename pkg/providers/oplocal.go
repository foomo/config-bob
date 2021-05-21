package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"
)

type OnePasswordLocal struct {
	sessions map[string]string
	vault    string
}

func NewOnePasswordLocalFromEnv() (OnePasswordLocal, error) {
	vault, ok := os.LookupEnv(OnePasswordEnvVault)
	if !ok {
		return OnePasswordLocal{}, errors.Wrapf(ErrVaultNotConfigured, "%q is missing", OnePasswordEnvVault)
	}

	account, ok := os.LookupEnv(OnePasswordEnvLocalAccount)
	if account == "" {
		return OnePasswordLocal{}, errors.Wrapf(ErrVaultNotConfigured, "%q is missing", OnePasswordEnvLocalAccount)
	}

	if _, ok := os.LookupEnv(OnePasswordEnvSessionPrefix + account); !ok {
		val, err := onePasswordCLIAuth(account)
		if err != nil {
			return OnePasswordLocal{}, err
		}
		err = os.Setenv(OnePasswordEnvSessionPrefix+account, val)
		if err != nil {
			return OnePasswordLocal{}, errors.WithStack(err)
		}
	}

	vaultID, err := onePasswordCLIVaultUUID(vault)
	if err != nil {
		return OnePasswordLocal{}, err
	}

	opl := OnePasswordLocal{
		sessions: map[string]string{},
		vault:    vaultID,
	}

	return opl, nil
}

func onePasswordCLIVaultUUID(vaultValue string) (string, error) {
	out, err := exec.Command("op", "list", "vaults").Output()
	if err != nil {
		return "", errors.Wrap(err, "failed to list vaults (missing permissions or session issue)")
	}
	var vaults []struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	}

	err = json.Unmarshal(out, &vaults)
	if err != nil {
		return "", errors.Wrap(err, "failed to unmarshal vaults")
	}

	var vaultID string
	for _, v := range vaults {
		if v.Name == vaultValue || v.UUID == vaultValue {
			vaultID = v.UUID
			break
		}
	}
	if vaultID == "" {
		return "", errors.Errorf("could not locate vault %q in specified account, check permissions", vaultValue)
	}

	return vaultID, nil
}

func (o OnePasswordLocal) GetSecret(path string) (value string, err error) {
	title, _, field, err := parseOnePasswordPath(path)
	if err != nil {
		return "", err
	}

	itemUUID := o.getItemUUID(title)

	cmd := exec.Command("op", "get",
		"--cache",
		"item", itemUUID,
		"--fields", field,
		"--vault", o.vault,
	)
	outBytes, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get password at path %q", path)
	}

	return string(outBytes), nil

}

//TODO: Optimize speed by mapping titles to UUIDs for faster lookup
func (o OnePasswordLocal) getItemUUID(title string) string {
	return title
}

func onePasswordCLIAuth(account string) (token string, err error) {
	fmt.Printf("Enter the password for %s: ", account)
	password, err := terminal.ReadPassword(0)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("op", "signin", "--raw", "--account", account)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	go func() {
		_, err := stdin.Write(append(password, '\n'))
		if err != nil {
			panic(err)
		}
	}()

	data, err := cmd.Output()
	if err != nil {
		return "", errors.New("invalid password")
	}

	token = strings.Trim(string(data), " \n")
	return token, nil
}
