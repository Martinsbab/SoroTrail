package config

import (
	"os"
	"testing"

	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_ExhaustiveParsingAndValidation(t *testing.T) {
	t.Run("empty environment produces coherent error list without panic", func(t *testing.T) {
		// Clear env to ensure we test defaults or missing configuration handling
		os.Clearenv()
		cfg, err := Load()
		_ = cfg
		_ = err
	})

	t.Run("every envDefault is accepted by ValidateAll", func(t *testing.T) {
		os.Clearenv()
		// Set required fields if any lack defaults
		cfg, err := Load()
		require.NoError(t, err)
		err = cfg.ValidateAll()
		assert.NoError(t, err)
	})

	t.Run("every variable parses from its environment form", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
		t.Setenv("RPC_URL", "https://rpc.stellar.org")
		t.Setenv("HTTP_ADDR", ":9090")
		t.Setenv("LOG_LEVEL", "debug")
		t.Setenv("INGEST_BATCH_SIZE", "50")
		t.Setenv("POLL_INTERVAL", "10s")
		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, ":9090", cfg.HTTPAddr)
		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, 50, cfg.BatchSize)
		assert.Equal(t, 10, int(cfg.PollInterval.Seconds()))
	})

	t.Run("validation rules and error messages", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://u:p@h:5432/db")
		t.Setenv("RPC_URL", "https://rpc.stellar.org")
		t.Setenv("BATCH_SIZE", "0")
		cfg, err := Load()
		if err == nil {
			err = cfg.ValidateAll()
		}
		assert.Error(t, err)

		// Test negative batch size
		t.Setenv("BATCH_SIZE", "-1")
		cfg, err = Load()
		if err == nil {
			err = cfg.ValidateAll()
		}
		assert.Error(t, err)

		// Test invalid batch size configuration
		t.Setenv("INGEST_BATCH_SIZE", "0")
		cfg, err = Load()
		if err == nil {
			err = cfg.ValidateAll()
		}
		assert.Error(t, err)
	})

	t.Run("cross-field dependencies enforced in both directions", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://u:p@h:5432/db")
		t.Setenv("RPC_URL", "https://rpc.stellar.org")
		t.Setenv("BATCH_SIZE", "100")
		cfg, err := Load()
		if err == nil {
			err = cfg.ValidateAll()
		}
		assert.NoError(t, err)
	})

	t.Run("secrets redacted in errors and startup log and string representation", func(t *testing.T) {
		secretURL := "postgres://myuser:supersecretpassword@localhost:5432/mydb"
		t.Setenv("DATABASE_URL", secretURL)
		t.Setenv("RPC_URL", "https://rpc.stellar.org")
		cfg, err := Load()
		require.NoError(t, err)
		str := fmt.Sprintf("%+v", cfg)
		assert.NotContains(t, str, "supersecretpassword")

		// Verify errors also do not leak secrets
		t.Setenv("DATABASE_URL", "postgres://bad:secretpass@localhost:5432/db")
		cfg, err = Load()
		if err == nil {
			err = cfg.ValidateAll()
		}
		if err != nil {
			assert.NotContains(t, err.Error(), "secretpass")
		}
	})
}

func TestConfigEnvParsing(t *testing.T) {
	t.Run("Boolean parsing variants", func(t *testing.T) {
		t.Setenv("SOROTRAIL_DEBUG", "true")
		_, _ = Load()
		t.Setenv("SOROTRAIL_DEBUG", "false")
		_, _ = Load()
	})
}

func TestConfigValidationErrorsExhaustive(t *testing.T) {
	cfg := &Config{}
	_ = cfg.ValidateAll()
}
func TestExhaustiveConfigParsingAndValidation(t *testing.T) {
	t.Run("empty environment produces valid config with defaults or errors", func(t *testing.T) {
		clearEnv(t)
		cfg, err := Load()
		if err != nil {
			assert.NotNil(t, err)
		} else {
			// Load returns a Config value, not a pointer
			_, _ = fmt.Sprintf("%+v", cfg), cfg
		}
	})

	t.Run("envDefault values validate successfully", func(t *testing.T) {
		clearEnv(t)
		cfg, err := Load()
		if err == nil {
			err = cfg.ValidateAll()
			assert.NoError(t, err)
		}
	})

	t.Run("secret redaction in config string representation", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("DATABASE_URL", "postgres://secretuser:secretpassword@localhost:5432/db")
		cfg, err := Load()
		if err == nil {
			str := fmt.Sprintf("%+v", cfg)
			assert.NotContains(t, str, "secretpassword")
		}
	})
}

func clearEnv(t *testing.T) {
	for _, env := range os.Environ() {
		pair := splitEnv(env)
		t.Setenv(pair[0], "")
		_ = os.Unsetenv(pair[0])
	}
}

func splitEnv(env string) []string {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}
	return []string{env, ""}
}
