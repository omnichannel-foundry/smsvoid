// Package config loads SmsVoid settings from flags and environment variables.
package config

import (
	"flag"
	"fmt"
	"os"
)

type Config struct{ SMPPAddr, HTTPAddr, Driver, DSN string }

func Load(args []string) (Config, error) {
	c := Config{SMPPAddr: env("SMSVOID_SMPP_ADDR", ":2775"), HTTPAddr: env("SMSVOID_HTTP_ADDR", ":8025"), Driver: env("SMSVOID_DB_DRIVER", "sqlite"), DSN: env("SMSVOID_DB_DSN", "smsvoid.db")}
	f := flag.NewFlagSet("smsvoid", flag.ContinueOnError)
	f.StringVar(&c.SMPPAddr, "smpp-addr", c.SMPPAddr, "SMPP listen address")
	f.StringVar(&c.HTTPAddr, "http-addr", c.HTTPAddr, "HTTP listen address")
	f.StringVar(&c.Driver, "db-driver", c.Driver, "sqlite or postgres")
	f.StringVar(&c.DSN, "db-dsn", c.DSN, "database path or PostgreSQL DSN")
	if err := f.Parse(args); err != nil {
		return c, err
	}
	if c.Driver != "sqlite" && c.Driver != "postgres" {
		return c, fmt.Errorf("db-driver must be sqlite or postgres")
	}
	return c, nil
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
