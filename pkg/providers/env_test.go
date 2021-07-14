package providers

import (
	"os"
	"testing"

	"github.com/google/uuid"
)

func TestLookupEnv(t *testing.T) {
	validKey := uuid.New().String()
	invalidKey := uuid.New().String()

	os.Setenv(validKey, "valid")
	defer os.Unsetenv(validKey)

	type args struct {
		key string
	}
	tests := []struct {
		name      string
		args      args
		wantValue string
		wantErr   bool
	}{
		{"valid", args{key: validKey}, "valid", false},
		{"invalid", args{key: invalidKey}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, err := LookupEnv(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("LookupEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotValue != tt.wantValue {
				t.Errorf("LookupEnv() gotValue = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
