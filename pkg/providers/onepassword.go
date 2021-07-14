package providers

import (
	"strings"

	"github.com/pkg/errors"
)

var (
	ErrVaultNotConfigured = errors.New("Vault not configured")
)

func parseOnePasswordPath(path string) (title, section, field string, err error) {
	parts := strings.Split(path, OnePasswordPathSeparator)
	if len(parts) < 2 || len(parts) > 3 {
		return "", "", "", errors.Errorf("wrong number of path parts, required 2 or 3 got %d", len(parts))
	}

	if len(parts) == 2 {
		return parts[0], "", parts[1], nil
	}

	return parts[0], parts[1], parts[2], nil
}
