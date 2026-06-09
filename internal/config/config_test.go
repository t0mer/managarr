package config_test

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/galactica/internal/config"
)

func newFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("config", "", "")
	fs.String("listen", ":8080", "")
	fs.String("log-level", "info", "")
	fs.String("log-format", "json", "")
	return fs
}

func TestDefaults(t *testing.T) {
	cfg, err := config.Load(newFlags())
	require.NoError(t, err)
	assert.Equal(t, ":8080", cfg.Server.Listen)
	assert.Equal(t, "sqlite", cfg.Storage.Driver)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.False(t, cfg.Auth.Enabled)
}

func TestEnvOverridesDefault(t *testing.T) {
	t.Setenv("GALACTICA_SERVER_LISTEN", ":9090")
	t.Setenv("GALACTICA_LOG_LEVEL", "debug")
	cfg, err := config.Load(newFlags())
	require.NoError(t, err)
	assert.Equal(t, ":9090", cfg.Server.Listen)
	assert.Equal(t, "debug", cfg.Log.Level)
}

func TestFlagOverridesEnv(t *testing.T) {
	t.Setenv("GALACTICA_SERVER_LISTEN", ":9090")
	fs := newFlags()
	require.NoError(t, fs.Parse([]string{"--listen", ":7070"}))
	cfg, err := config.Load(fs)
	require.NoError(t, err)
	assert.Equal(t, ":7070", cfg.Server.Listen)
}
