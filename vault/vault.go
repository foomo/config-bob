package vault

import (
	"errors"
	"os/exec"
	"strings"
)

// GetVaultVersion get the version of the installed vault command line program
func GetVaultVersion() (version string, err error) {
	out, err := exec.Command("vault", "-v").Output()
	if err != nil {
		return "", errors.New("can not find vault")
	}
	return strings.Trim(string(out), "\n"), nil
}

// CheckEnv checks if vault is installed in the right version and set up
func CheckEnv() bool {
	return false
}
