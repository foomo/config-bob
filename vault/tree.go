package vault

import (
	"fmt"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
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
	leaves, err := leafPaths(strings.TrimSuffix(path, "/"))
	if err != nil {
		return nil, err
	}
	vaultData := make(map[string]map[string]string, len(leaves))
	var (
		g  errgroup.Group
		mu sync.Mutex
	)
	for _, leaf := range leaves {
		g.Go(func() error {
			data, err := Read(leaf)
			if err != nil {
				return err
			}
			mu.Lock()
			vaultData[leaf] = data
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return vaultData, nil
}

// leafPaths lists folders recursively and returns all secret paths below path
func leafPaths(path string) ([]string, error) {
	paths, err := list(path)
	if err != nil {
		return nil, err
	}
	var leaves []string
	for _, p := range paths {
		current := path + "/" + strings.TrimPrefix(p, "/")
		if strings.HasSuffix(p, "/") {
			sub, err := leafPaths(strings.TrimSuffix(current, "/"))
			if err != nil {
				return nil, err
			}
			leaves = append(leaves, sub...)
		} else {
			leaves = append(leaves, current)
		}
	}
	return leaves, nil
}
