package config

import "testing"

func TestLoadDefaultsAndOverrides(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		c, err := Load(nil)
		if err != nil {
			t.Fatal(err)
		}
		if c.SMPPAddr != ":2775" || c.HTTPAddr != ":8025" || c.Driver != "sqlite" || c.DSN != "smsvoid.db" {
			t.Fatalf("unexpected defaults: %+v", c)
		}
	})
	t.Run("flags", func(t *testing.T) {
		c, err := Load([]string{"-smpp-addr", "127.0.0.1:2776", "-http-addr", ":9000", "-db-driver", "postgres", "-db-dsn", "postgres://localhost/smsvoid"})
		if err != nil {
			t.Fatal(err)
		}
		if c.SMPPAddr != "127.0.0.1:2776" || c.HTTPAddr != ":9000" || c.Driver != "postgres" {
			t.Fatalf("unexpected config: %+v", c)
		}
	})
	t.Run("invalid driver", func(t *testing.T) {
		if _, err := Load([]string{"-db-driver", "mysql"}); err == nil {
			t.Fatal("expected driver validation error")
		}
	})
	t.Run("bad flag", func(t *testing.T) {
		if _, err := Load([]string{"-unknown"}); err == nil {
			t.Fatal("expected flag error")
		}
	})
}
