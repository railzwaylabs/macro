package store

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func dialector(cfg Config) (gorm.Dialector, error) {
	switch cfg.Driver {
	case Postgres:
		return postgres.Open(postgresConnectionString(cfg)), nil
	case MySQL:
		connectionString, err := mysqlConnectionString(cfg)
		if err != nil {
			return nil, err
		}
		return gormmysql.Open(connectionString), nil
	case SQLite:
		return sqlite.Open(cfg.Database), nil
	default:
		return nil, fmt.Errorf("store: unsupported driver %q", cfg.Driver)
	}
}

func postgresConnectionString(cfg Config) string {
	connectionURL := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))),
		Path:   cfg.Database,
	}
	if cfg.Password == "" {
		connectionURL.User = url.User(cfg.Username)
	} else {
		connectionURL.User = url.UserPassword(cfg.Username, cfg.Password)
	}

	query := connectionURL.Query()
	query.Set("sslmode", cfg.SSLMode)
	query.Set("timezone", cfg.TimeZone)
	for key, value := range cfg.Parameters {
		query.Set(key, value)
	}
	connectionURL.RawQuery = query.Encode()

	return connectionURL.String()
}

func mysqlConnectionString(cfg Config) (string, error) {
	location, err := time.LoadLocation(cfg.TimeZone)
	if err != nil {
		return "", fmt.Errorf("store: load MySQL timezone %q: %w", cfg.TimeZone, err)
	}

	parameters := make(map[string]string, len(cfg.Parameters)+1)
	parameters["charset"] = "utf8mb4"
	for key, value := range cfg.Parameters {
		parameters[key] = value
	}

	return (&mysqldriver.Config{
		User:      cfg.Username,
		Passwd:    cfg.Password,
		Net:       "tcp",
		Addr:      net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))),
		DBName:    cfg.Database,
		Params:    parameters,
		ParseTime: true,
		Loc:       location,
	}).FormatDSN(), nil
}
