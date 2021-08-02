package vault

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// Tree a tree of secrets
func Tree(path string) error {
	data, err := tree(path)
	if err != nil {
		return err
	}

	for p, d := range data {
		fmt.Printf("\n\n%s", p)
		for k, v := range d {
			v := strings.Replace(v, "\n", "\\n", -1)
			fmt.Printf("\n\t%s=%s", k, v)
		}
	}
	fmt.Printf("\nlisting done\n")
	return nil
}

func tree(path string) (map[string]map[string]string, error) {
	path = strings.TrimSuffix(path, "/")
	cmd := exec.Command("vault", "list", "-format", "json", path)
	jsonBytes, err := cmd.CombinedOutput()
	if err != nil {
		err := vaultErr(jsonBytes, err)
		return nil, errors.Wrapf(err, "failed to read path %q", path)
	}
	var paths []string
	if string(jsonBytes) == "No entries found\n" {
		// thank you for the json
		return nil, nil
	}
	err = json.Unmarshal(jsonBytes, &paths)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal data at %q", path)
	}

	vaultData := map[string]map[string]string{}
	for _, p := range paths {
		current := fmt.Sprintf("%s/%s", strings.TrimSuffix(path, "/"), strings.TrimPrefix(p, "/"))
		if strings.HasSuffix(p, "/") {
			path := path + "/" + p[:len(p)-1]

			data, err := tree(path)
			if err != nil {
				return nil, err
			}
			for key, value := range data {
				vaultData[key] = value
			}

		} else {
			data, err := Read(current)
			if err != nil {
				return nil, err
			}
			vaultData[current] = map[string]string{}
			for key, value := range data {
				vaultData[current][key] = fmt.Sprint(value)
			}
		}

	}

	return vaultData, nil
}
