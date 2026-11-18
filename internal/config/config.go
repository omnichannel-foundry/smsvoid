// Package config loads SmsVoid settings from flags and environment variables.
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	SMPPAddr, HTTPAddr, Driver, DSN string
	SMPPUser, SMPPPass              string
	HTTPUser, HTTPPass              string
	FailDeliveries                  bool
}

func Load(args []string) (Config, error) {
	c := Config{
		SMPPAddr: env("SMSVOID_SMPP_ADDR", ":2775"),
		HTTPAddr: env("SMSVOID_HTTP_ADDR", ":8025"),
		Driver:   env("SMSVOID_DB_DRIVER", "sqlite"),
		DSN:      env("SMSVOID_DB_DSN", "smsvoid.db"),
		SMPPUser: env("SMSVOID_SMPP_USER", ""),
		SMPPPass: env("SMSVOID_SMPP_PASS", ""),
		HTTPUser: env("SMSVOID_HTTP_USER", ""),
		HTTPPass: env("SMSVOID_HTTP_PASS", ""),
	}
	failDeliveries := false
	if v := env("SMSVOID_FAIL_DELIVERIES", ""); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			failDeliveries = b
		}
	}
	c.FailDeliveries = failDeliveries

	f := flag.NewFlagSet("smsvoid", flag.ContinueOnError)
	f.StringVar(&c.SMPPAddr, "smpp-addr", c.SMPPAddr, "SMPP listen address")
	f.StringVar(&c.HTTPAddr, "http-addr", c.HTTPAddr, "HTTP listen address")
	f.StringVar(&c.Driver, "db-driver", c.Driver, "sqlite or postgres")
	f.StringVar(&c.DSN, "db-dsn", c.DSN, "database path or PostgreSQL DSN")
	f.StringVar(&c.SMPPUser, "smpp-user", c.SMPPUser, "SMPP bind system id (empty disables auth)")
	f.StringVar(&c.SMPPPass, "smpp-pass", c.SMPPPass, "SMPP bind password")
	f.StringVar(&c.HTTPUser, "http-user", c.HTTPUser, "HTTP basic auth user (empty disables auth)")
	f.StringVar(&c.HTTPPass, "http-pass", c.HTTPPass, "HTTP basic auth password")
	f.BoolVar(&c.FailDeliveries, "fail-deliveries", c.FailDeliveries, "simulate failed delivery receipts")
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
