package providers

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewOnePasswordLocalFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    OnePasswordLocal
		wantErr bool
	}{
		{"empty", nil, OnePasswordLocal{}, true},
		{"invalid vault", map[string]string{"OP_LOCAL_ACCOUNT": "TEST"}, OnePasswordLocal{}, true},
		{"invalid session", map[string]string{"OP_VAULT": "TEST"}, OnePasswordLocal{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				err := os.Setenv(k, v)
				require.NoError(t, err)
				defer os.Unsetenv(k)
			}

			got, err := NewOnePasswordLocalFromEnv()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOnePasswordLocalFromEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewOnePasswordLocalFromEnv() got = %v, want %v", got, tt.want)
			}
		})
	}
}
