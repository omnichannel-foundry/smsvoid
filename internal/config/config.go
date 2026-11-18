// Package config loads SmsVoid settings from flags, environment, and TOML files.
package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gnrfan/smsvoid/internal/runtime"
)

type Config struct {
	SMPPAddr, HTTPAddr, Driver, DSN string
	SMPPUser, SMPPPass              string
	HTTPUser, HTTPPass              string
	FailDeliveries                  bool

	// Runtime discovery.
	Mode         string
	ConfigPath   string
	ConfigFile   string
	Paths        runtime.Paths
	Runtime      runtime.Paths
	RuntimePaths runtime.Paths
	PathsAlias   runtime.Paths

	// CLI actions.
	Info    bool
	Version bool
	Doctor  bool
	Init    bool
}

type fileConfig struct {
	SMPPAddr       string `toml:"smpp_addr"`
	HTTPAddr       string `toml:"http_addr"`
	Driver         string `toml:"db_driver"`
	DSN            string `toml:"db_dsn"`
	SMPPUser       string `toml:"smpp_user"`
	SMPPPass       string `toml:"smpp_pass"`
	HTTPUser       string `toml:"http_user"`
	HTTPPass       string `toml:"http_pass"`
	FailDeliveries *bool  `toml:"fail_deliveries"`
}

func Load(args []string) (Config, error) {
	cfg := Config{
		SMPPAddr: ":2775",
		HTTPAddr: ":8025",
		Driver:   "sqlite",
		DSN:      "smsvoid.db",
	}
	// Environment overrides (higher priority than TOML, lower than CLI).
	envSet := map[string]bool{}
	applyEnv := func(key, flagName string, set func(string)) {
		if v := os.Getenv(key); v != "" {
			set(v)
			envSet[flagName] = true
		}
	}
	applyEnv("SMSVOID_SMPP_ADDR", "smpp-addr", func(v string) { cfg.SMPPAddr = v })
	applyEnv("SMSVOID_HTTP_ADDR", "http-addr", func(v string) { cfg.HTTPAddr = v })
	applyEnv("SMSVOID_DB_DRIVER", "db-driver", func(v string) { cfg.Driver = v })
	applyEnv("SMSVOID_DB_DSN", "db-dsn", func(v string) { cfg.DSN = v })
	applyEnv("SMSVOID_SMPP_USER", "smpp-user", func(v string) { cfg.SMPPUser = v })
	applyEnv("SMSVOID_SMPP_PASS", "smpp-pass", func(v string) { cfg.SMPPPass = v })
	applyEnv("SMSVOID_HTTP_USER", "http-user", func(v string) { cfg.HTTPUser = v })
	applyEnv("SMSVOID_HTTP_PASS", "http-pass", func(v string) { cfg.HTTPPass = v })
	if v := os.Getenv("SMSVOID_FAIL_DELIVERIES"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.FailDeliveries = b
			envSet["fail-deliveries"] = true
		}
	}

	f := flag.NewFlagSet("smsvoid", flag.ContinueOnError)
	f.StringVar(&cfg.SMPPAddr, "smpp-addr", cfg.SMPPAddr, "SMPP listen address")
	f.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	f.StringVar(&cfg.Driver, "db-driver", cfg.Driver, "sqlite or postgres")
	f.StringVar(&cfg.DSN, "db-dsn", cfg.DSN, "database path or PostgreSQL DSN")
	f.StringVar(&cfg.SMPPUser, "smpp-user", cfg.SMPPUser, "SMPP bind system id (empty disables auth)")
	f.StringVar(&cfg.SMPPPass, "smpp-pass", cfg.SMPPPass, "SMPP bind password")
	f.StringVar(&cfg.HTTPUser, "http-user", cfg.HTTPUser, "HTTP basic auth user (empty disables auth)")
	f.StringVar(&cfg.HTTPPass, "http-pass", cfg.HTTPPass, "HTTP basic auth password")
	f.BoolVar(&cfg.FailDeliveries, "fail-deliveries", cfg.FailDeliveries, "simulate failed delivery receipts")
	f.StringVar(&cfg.ConfigPath, "config", "", "explicit TOML config path")
	f.BoolVar(&cfg.Info, "info", false, "print runtime paths and exit")
	f.BoolVar(&cfg.Version, "version", false, "print version and exit")
	f.BoolVar(&cfg.Doctor, "doctor", false, "diagnose runtime paths and exit")
	f.BoolVar(&cfg.Init, "init", false, "create default config file and exit")
	if err := f.Parse(args); err != nil {
		return cfg, err
	}
	if cfg.Driver != "sqlite" && cfg.Driver != "postgres" {
		return cfg, fmt.Errorf("db-driver must be sqlite or postgres")
	}

	// Track explicitly set CLI flags.
	changed := map[string]bool{}
	f.Visit(func(fl *flag.Flag) { changed[fl.Name] = true })

	// Runtime discovery.
	exe, err := runtime.ExecutablePath()
	if err != nil {
		return cfg, err
	}
	home, _ := os.UserHomeDir()
	getenv := func(k string) string { return os.Getenv(k) }
	paths := runtime.Resolve(exe, home, getenv)
	cfg.Mode = string(paths.Mode)
	cfg.Paths = paths
	cfg.Runtime = paths
	cfg.RuntimePaths = paths
	cfg.PathsAlias = paths

	// Config discovery.
	configPath, found, err := runtime.DiscoverConfig(paths, cfg.ConfigPath)
	if err != nil {
		return cfg, err
	}
	cfg.ConfigFile = configPath

	// Apply TOML values for keys not set by env or CLI.
	if found {
		var fc fileConfig
		md, err := toml.DecodeFile(configPath, &fc)
		if err != nil {
			return cfg, fmt.Errorf("invalid config file %s: %w", configPath, err)
		}
		apply := func(key, flagName string, set func()) {
			if md.IsDefined(key) && !changed[flagName] && !envSet[flagName] {
				set()
			}
		}
		apply("smpp_addr", "smpp-addr", func() { cfg.SMPPAddr = fc.SMPPAddr })
		apply("http_addr", "http-addr", func() { cfg.HTTPAddr = fc.HTTPAddr })
		apply("db_driver", "db-driver", func() { cfg.Driver = fc.Driver })
		apply("db_dsn", "db-dsn", func() { cfg.DSN = fc.DSN })
		apply("smpp_user", "smpp-user", func() { cfg.SMPPUser = fc.SMPPUser })
		apply("smpp_pass", "smpp-pass", func() { cfg.SMPPPass = fc.SMPPPass })
		apply("http_user", "http-user", func() { cfg.HTTPUser = fc.HTTPUser })
		apply("http_pass", "http-pass", func() { cfg.HTTPPass = fc.HTTPPass })
		if md.IsDefined("fail_deliveries") && fc.FailDeliveries != nil && !changed["fail-deliveries"] && !envSet["fail-deliveries"] {
			cfg.FailDeliveries = *fc.FailDeliveries
		}
		if cfg.Driver != "sqlite" && cfg.Driver != "postgres" {
			return cfg, fmt.Errorf("db-driver must be sqlite or postgres")
		}
	}
	return cfg, nil
}

// DefaultConfigTemplate returns a commented TOML template.
func DefaultConfigTemplate() string {
	return strings.TrimSpace(`
# SmsVoid configuration
smpp_addr = ":2775"
http_addr = ":8025"
db_driver = "sqlite"
db_dsn = "smsvoid.db"
smpp_user = ""
smpp_pass = ""
http_user = ""
http_pass = ""
fail_deliveries = false
`) + "\n"
}

// WriteDefaultConfig creates a default config file if absent.
func WriteDefaultConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(DefaultConfigTemplate()), 0644)
}

// Convenience aliases for runtime discovery.
type Mode = runtime.Mode
type Paths = runtime.Paths
type RuntimePaths = runtime.Paths

const (
	ModeDevelopment = runtime.ModeDevelopment
	ModeUser        = runtime.ModeUser
	ModeSystem      = runtime.ModeSystem
	Development     = runtime.Development
	User            = runtime.User
	System          = runtime.System
)

func DetectMode(args ...interface{}) Mode    { return runtime.DetectMode(args...) }
func Resolve(args ...interface{}) Paths      { return runtime.Resolve(args...) }
func ResolvePaths(args ...interface{}) Paths { return runtime.ResolvePaths(args...) }
func DiscoverConfig(args ...interface{}) (string, bool, error) {
	return runtime.DiscoverConfig(args...)
}
