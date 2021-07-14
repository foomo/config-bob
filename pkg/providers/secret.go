package providers

import (
	"strings"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	_ SecretProvider = &OnePasswordConnect{}
	_ SecretProvider = &Vault{}
)

type SecretProvider interface {
	GetSecret(path string) (value string, err error)
}

type SecretProviderManager struct {
	cache     *Cache
	providers map[string]SecretProvider
	lock      sync.RWMutex
}

func NewSecretProviderManager() SecretProviderManager {
	return SecretProviderManager{
		cache:     NewCache(),
		providers: map[string]SecretProvider{},
		lock:      sync.RWMutex{},
	}
}

func NewSecretProviderManagerFromEnv(l *zap.Logger) (SecretProviderManager, error) {
	manager := NewSecretProviderManager()

	if IsOnePasswordConnectConfigured() {
		l.Info("Found 1Password connect configuration from env")
		provider, err := NewOnePasswordConnectFromEnv()
		if err != nil {
			return SecretProviderManager{}, err
		}
		err = manager.Register("op", provider)
		if err != nil {
			return SecretProviderManager{}, err
		}
	}

	if IsOnePasswordLocalConfigured() {
		l.Info("Found 1Password local configuration from env")
		provider, err := NewOnePasswordLocalFromEnv()
		if err != nil {
			return SecretProviderManager{}, err
		}
		err = manager.Register("op", provider)
		if err != nil {
			return SecretProviderManager{}, err
		}
	}

	if IsGoogleSecretsConfigured() {
		l.Info("Found GoogleSecrets configuration from env")
		provider, err := NewGoogleSecretsProviderFromEnv()
		if err != nil {
			return SecretProviderManager{}, err
		}
		err = manager.Register("gs", provider)
		if err != nil {
			return SecretProviderManager{}, err
		}
	}

	return manager, nil
}

func (stp *SecretProviderManager) Register(tag string, provider SecretProvider) error {
	stp.lock.Lock()
	defer stp.lock.Unlock()
	if _, ok := stp.providers[tag]; ok {
		return errors.Errorf("secret provider with tag %q already exists", tag)
	}

	stp.providers[tag] = provider

	return nil
}

func (stp *SecretProviderManager) GetSecret(params ...string) (value string, err error) {
	if len(stp.providers) == 0 {
		return "", errors.New("no secret providers registered")
	}

	var provider, path string
	stp.lock.RLock()
	defer stp.lock.RUnlock()

	switch len(params) {
	case 1:
		if len(stp.providers) != 1 {
			return "", errors.Errorf("provider for secret must be specified, multiple providers present")
		}
		for providerName := range stp.providers {
			provider = providerName
		}
		path = params[0]
	case 2:
		provider, path = params[0], params[1]
	default:
		return "", errors.Errorf("invalid number of arguments, required 1 or 2, but got %d", len(params))
	}

	// Check if the value is cached
	if value, ok := stp.cache.Get(provider, path); ok {
		return value, nil
	}

	secretProvider, ok := stp.providers[provider]
	if !ok {
		return "", errors.Errorf("provider %q was not registered, and does not exist", provider)
	}

	secret, err := secretProvider.GetSecret(path)
	if err != nil {
		return "", err
	}
	value = strings.Trim(secret, " \n")
	// Cache for speedup
	stp.cache.Set(provider, path, value)

	return value, nil
}
