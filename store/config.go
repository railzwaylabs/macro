package store

import (
	"fmt"
	"strings"
	"time"
)

type Driver string

const (
	Postgres Driver = "postgres"
	MySQL    Driver = "mysql"
	SQLite   Driver = "sqlite"
)

type Config struct {
	Driver                Driver
	Host                  string
	Port                  uint16
	Username              string
	Password              string
	Database              string
	SSLMode               string
	TimeZone              string
	Parameters            map[string]string
	MaxOpenConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
	ConnectionMaxIdleTime time.Duration
}

func (c Config) Validate() error {
	switch c.Driver {
	case Postgres, MySQL, SQLite:
	default:
		return fmt.Errorf("store: unsupported driver %q", c.Driver)
	}

	if strings.TrimSpace(c.Database) == "" {
		return fmt.Errorf("store: database is required for %s", c.Driver)
	}
	if c.Driver != SQLite {
		if strings.TrimSpace(c.Host) == "" {
			return fmt.Errorf("store: host is required for %s", c.Driver)
		}
		if strings.TrimSpace(c.Username) == "" {
			return fmt.Errorf("store: username is required for %s", c.Driver)
		}
	}
	if c.MaxOpenConnections < 0 || c.MaxIdleConnections < 0 {
		return fmt.Errorf("store: connection limits cannot be negative")
	}
	if c.ConnectionMaxLifetime < 0 || c.ConnectionMaxIdleTime < 0 {
		return fmt.Errorf("store: connection durations cannot be negative")
	}

	return nil
}

func (c Config) withDefaults() Config {
	if strings.TrimSpace(c.TimeZone) == "" {
		c.TimeZone = "UTC"
	}

	switch c.Driver {
	case Postgres:
		if c.Port == 0 {
			c.Port = 5432
		}
		if strings.TrimSpace(c.SSLMode) == "" {
			c.SSLMode = "disable"
		}
	case MySQL:
		if c.Port == 0 {
			c.Port = 3306
		}
	}

	return c
}
