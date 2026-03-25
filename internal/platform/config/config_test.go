package config

import "testing"

func TestLoadUsesDatabaseDSN(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable")
	t.Setenv("DATABASE_MAX_CONNECTIONS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Database.DSN != "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable" {
		t.Fatalf("unexpected database dsn %q", cfg.Database.DSN)
	}
}

func TestLoadProvidesDefaultPluginServices(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	service, ok := cfg.Plugins.Services["octo_demo"]
	if !ok {
		t.Fatalf("expected built-in octo_demo service default")
	}
	if service.Address != "127.0.0.1:50051" {
		t.Fatalf("unexpected default address %q", service.Address)
	}
}

func TestLoadPluginServicesFromEnv(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://octo:octo@localhost:5432/octomanger?sslmode=disable")
	t.Setenv("PLUGIN_GRPC_OCTO_DEMO_ADDR", "127.0.0.1:50051")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	service, ok := cfg.Plugins.Services["octo_demo"]
	if !ok {
		t.Fatalf("expected octo_demo service to be loaded from env")
	}
	if service.Address != "127.0.0.1:50051" {
		t.Fatalf("unexpected service address %q", service.Address)
	}
}
