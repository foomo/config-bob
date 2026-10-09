package builder

import (
	"os/exec"
	"strings"
)

func onePassword(uuid, field string) (value string, err error) {
	cmd := exec.Command("op", "item", "get", uuid, "--fields", field, "--reveal")
	outBytes, errOut := cmd.Output()
	if errOut != nil {
		return "", errOut
	}
	return strings.TrimSuffix(string(outBytes), "\n"), nil
}
