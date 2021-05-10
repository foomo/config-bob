package providers

import (
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
	providers map[string]SecretProvider
	lock      sync.RWMutex
}

func NewSecretProviderManager() SecretProviderManager {
	return SecretProviderManager{
		providers: map[string]SecretProvider{},
		lock:      sync.RWMutex{},
	}
}

func NewSecretProviderManagerFromEnv(l *zap.Logger) (SecretProviderManager, error) {
	manager := SecretProviderManager{
		providers: map[string]SecretProvider{},
		lock:      sync.RWMutex{},
	}

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

	return manager, nil
}

func (stp *SecretProviderManager) Register(tag string, provider SecretProvider) error {
	stp.lock.Lock()
	defer stp.lock.Unlock()
	if _, ok := stp.providers[tag]; ok {
		return errors.Errorf("secret provider with tag %q already exists", stp)
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

	secretProvider, ok := stp.providers[provider]
	if !ok {
		return "", errors.Errorf("provider %q was not registered, and does not exist", provider)
	}

	return secretProvider.GetSecret(path)
}
