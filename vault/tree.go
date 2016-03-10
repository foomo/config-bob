package vault

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

func treeIndent(level int) string {
	return strings.Repeat("  ", level)
}

// Tree a tree of secrets
func Tree(path string, level int) (err error) {
	cmd := exec.Command("vault", "list", "-format", "json", path)
	jsonBytes, err := cmd.CombinedOutput()
	if err != nil {
		return vaultErr(jsonBytes, err)
	}
	paths := []string{}
	err = json.Unmarshal(jsonBytes, &paths)
	if err != nil {
		return err
	}
	for _, p := range paths {
		current := path + "/" + p
		fmt.Println(treeIndent(level), p)
		if strings.HasSuffix(p, "/") {
			err = Tree(path+"/"+p[:len(p)-1], level+1)
			if err != nil {
				return err
			}
		} else {
			data, err := Read(current)
			if err != nil {
				return err
			}
			padLength := 0
			for key := range data {
				if len(key) > padLength {
					padLength = len(key)
				}
			}
			for key, value := range data {
				fmt.Println(treeIndent(level+1), key, strings.Repeat(" ", padLength-len(key)), ":", value)
			}
		}
	}
	return nil
}
