package vault

import (
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/cliconfig"
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

// newClient is configured from the VAULT_* env vars and the token helper like the vault CLI
func newClient() (*api.Client, error) {
	c, err := api.NewClient(nil)
	if err != nil {
		return nil, err
	}
	if c.Token() == "" {
		helper, err := cliconfig.DefaultTokenHelper()
		if err != nil {
			return nil, err
		}
		token, err := helper.Get()
		if err != nil {
			return nil, err
		}
		c.SetToken(strings.TrimSpace(token))
	}
	return c, nil
}

// sanitizePath trims like the vault CLI, so "/secret/db" and "secret/db" are the same secret
func sanitizePath(path string) string {
	return strings.Trim(strings.TrimSpace(path), "/")
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

	path = sanitizePath(path)
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
		if value == nil {
			// the vault CLI output decoded null as an empty string
			data[key] = ""
			continue
		}
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
	secret, err := c.Logical().List(sanitizePath(path))
	if err != nil {
		return nil, err
	}
	var keys []any
	if secret != nil {
		keys, _ = secret.Data["keys"].([]any)
	}
	if len(keys) == 0 {
		// like the vault CLI, an empty or missing folder is an error
		var warnings []string
		if secret != nil {
			warnings = secret.Warnings
		}
		return nil, fmt.Errorf("no entries found at %q %v", path, warnings)
	}
	paths := make([]string, 0, len(keys))
	for _, key := range keys {
		if s, ok := key.(string); ok {
			paths = append(paths, s)
		}
	}
	return paths, nil
}
