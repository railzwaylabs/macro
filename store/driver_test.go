package store

import (
	"strings"
	"testing"
)

func TestPostgresConnectionStringEscapesCredentials(t *testing.T) {
	t.Parallel()

	cfg := (Config{
		Driver:   Postgres,
		Host:     "localhost",
		Username: "macro-user",
		Password: "secret@word",
		Database: "billing",
	}).withDefaults()

	connectionString := postgresConnectionString(cfg)
	if strings.Contains(connectionString, "secret@word") {
		t.Fatalf("password was not URL encoded: %s", connectionString)
	}
	if !strings.Contains(connectionString, "localhost:5432") {
		t.Fatalf("default PostgreSQL port missing: %s", connectionString)
	}
}

func TestMySQLConnectionStringUsesDefaults(t *testing.T) {
	t.Parallel()

	cfg := (Config{
		Driver:   MySQL,
		Host:     "localhost",
		Username: "macro",
		Database: "billing",
	}).withDefaults()

	connectionString, err := mysqlConnectionString(cfg)
	if err != nil {
		t.Fatalf("mysqlConnectionString() error = %v", err)
	}
	if !strings.Contains(connectionString, "tcp(localhost:3306)") {
		t.Fatalf("default MySQL port missing: %s", connectionString)
	}
	if !strings.Contains(connectionString, "parseTime=true") {
		t.Fatalf("parseTime is not enabled: %s", connectionString)
	}
}
