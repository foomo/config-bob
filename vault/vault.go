package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/vault/api"
	"golang.org/x/sync/singleflight"
)

const vaultAddr = "127.0.0.1:8200"

// VaultDummy enables a built in dummy
var Dummy = false

// readConcurrency caps parallel requests so large builds do not burst vault
const readConcurrency = 16

var (
	readSlots   = make(chan struct{}, readConcurrency)
	readGroup   singleflight.Group
	readLock    sync.RWMutex
	secretCache = map[string]map[string]string{}
)

// client is created once so all reads share one connection pool
var client = sync.OnceValues(newClient)

// newClient is configured from the VAULT_* env vars like the vault CLI
func newClient() (*api.Client, error) {
	c, err := api.NewClient(nil)
	if err != nil {
		return nil, err
	}
	if c.Token() == "" {
		// same fallback as the default token helper of the vault CLI after `vault login`
		if home, err := os.UserHomeDir(); err == nil {
			if token, err := os.ReadFile(filepath.Join(home, ".vault-token")); err == nil {
				c.SetToken(strings.TrimSpace(string(token)))
			}
		}
	}
	return c, nil
}

// Read data from a vault - env vars need to be set; each path is fetched once per process
// and the returned map is shared, callers must not modify it
func Read(path string) (secret map[string]string, err error) {
	if Dummy {
		return map[string]string{
			"token":    "well-a-token",
			"name":     "call my name",
			"user":     "user-from" + path,
			"password": "dummy-password",
			"escape":   "muha\"haha",
		}, nil
	}

	value, err, _ := readGroup.Do(path, func() (any, error) {
		readLock.RLock()
		cached, ok := secretCache[path]
		readLock.RUnlock()
		if ok {
			return cached, nil
		}
		data, err := read(path)
		if err != nil {
			return nil, err
		}
		readLock.Lock()
		secretCache[path] = data
		readLock.Unlock()
		return data, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(map[string]string), nil
}

func read(path string) (map[string]string, error) {
	c, err := client()
	if err != nil {
		return nil, err
	}
	readSlots <- struct{}{}
	secret, err := c.Logical().Read(path)
	<-readSlots
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, fmt.Errorf("no secret found at %q", path)
	}
	data := make(map[string]string, len(secret.Data))
	for key, value := range secret.Data {
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("value of %q at %q is not a string", key, path)
		}
		data[key] = s
	}
	return data, nil
}

func list(path string) ([]string, error) {
	c, err := client()
	if err != nil {
		return nil, err
	}
	secret, err := c.Logical().List(path)
	if err != nil || secret == nil {
		return nil, err
	}
	keys, _ := secret.Data["keys"].([]any)
	paths := make([]string, 0, len(keys))
	for _, key := range keys {
		if s, ok := key.(string); ok {
			paths = append(paths, s)
		}
	}
	return paths, nil
}
