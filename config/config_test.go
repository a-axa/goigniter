package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetEnv(t *testing.T) {
	t.Run("uses environment value when set", func(t *testing.T) {
		t.Setenv("TEST_KEY", "from-env")

		if got := getEnv("TEST_KEY", "fallback"); got != "from-env" {
			t.Errorf("getEnv() = %q, want %q", got, "from-env")
		}
	})

	t.Run("uses fallback when unset", func(t *testing.T) {
		os.Unsetenv("TEST_KEY_MISSING")

		if got := getEnv("TEST_KEY_MISSING", "fallback"); got != "fallback" {
			t.Errorf("getEnv() = %q, want %q", got, "fallback")
		}
	})

	t.Run("empty value is still respected", func(t *testing.T) {
		t.Setenv("TEST_KEY_EMPTY", "")

		if got := getEnv("TEST_KEY_EMPTY", "fallback"); got != "" {
			t.Errorf("getEnv() = %q, want empty string", got)
		}
	})
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		set      bool
		fallback int
		want     int
	}{
		{name: "valid number", value: "9090", set: true, fallback: 8080, want: 9090},
		{name: "invalid number falls back", value: "abc", set: true, fallback: 8080, want: 8080},
		{name: "unset falls back", set: false, fallback: 8080, want: 8080},
		{name: "negative number is parsed", value: "-5", set: true, fallback: 8080, want: -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv("TEST_INT_KEY", tt.value)
			} else {
				os.Unsetenv("TEST_INT_KEY")
			}

			if got := getEnvInt("TEST_INT_KEY", tt.fallback); got != tt.want {
				t.Errorf("getEnvInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLoadReadsDotEnvAndDefaults(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	content := "APP_NAME=TestApp\nAPP_PORT=9999\nDB_HOST=db.local\nJWT_EXPIRE_HOURS=12\n"
	if err := os.WriteFile(envFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Chdir(dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.App.Name != "TestApp" {
		t.Errorf("App.Name = %q, want %q", cfg.App.Name, "TestApp")
	}
	if cfg.App.Port != 9999 {
		t.Errorf("App.Port = %d, want 9999", cfg.App.Port)
	}
	if cfg.App.Env != "development" {
		t.Errorf("App.Env = %q, want default %q", cfg.App.Env, "development")
	}
	if cfg.DB.Host != "db.local" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "db.local")
	}
	if cfg.DB.Port != 3306 {
		t.Errorf("DB.Port = %d, want default 3306", cfg.DB.Port)
	}
	if cfg.JWT.ExpireHours != 12 {
		t.Errorf("JWT.ExpireHours = %d, want 12", cfg.JWT.ExpireHours)
	}
}

func TestLoadMissingDotEnv(t *testing.T) {
	t.Chdir(t.TempDir())

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error when .env is missing")
	}
}
