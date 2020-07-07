package builder

import (
	"os/exec"
)

func onePassword(uuid, field string) (value string, err error) {
	cmd := exec.Command("op", "get", "item", uuid, "--fields", field)
	outBytes, errOut := cmd.Output()
	if errOut != nil {
		return "", errOut
	}
	return string(outBytes), nil
}
