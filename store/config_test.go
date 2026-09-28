package store_test

import (
	"testing"

	"github.com/railzwaylabs/macro/store"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  store.Config
		wantErr bool
	}{
		{
			name: "postgres",
			config: store.Config{
				Driver: store.Postgres, Host: "localhost", Username: "macro", Database: "macro",
			},
		},
		{
			name: "mysql",
			config: store.Config{
				Driver: store.MySQL, Host: "localhost", Username: "macro", Database: "macro",
			},
		},
		{name: "sqlite", config: store.Config{Driver: store.SQLite, Database: ":memory:"}},
		{name: "missing database", config: store.Config{Driver: store.Postgres, Host: "localhost", Username: "macro"}, wantErr: true},
		{name: "missing host", config: store.Config{Driver: store.Postgres, Username: "macro", Database: "macro"}, wantErr: true},
		{name: "unknown driver", config: store.Config{Driver: "unknown", Database: "macro"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
