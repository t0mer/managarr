package config

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig  `mapstructure:"server"`
	Storage   StorageConfig `mapstructure:"storage"`
	Log       LogConfig     `mapstructure:"log"`
	Auth      AuthConfig    `mapstructure:"auth"`
	SecretKey string        `mapstructure:"secret_key"`
}

type ServerConfig struct {
	Listen string `mapstructure:"listen"`
}

type StorageConfig struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type AuthConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// Load builds Config from flags, environment, and optional YAML file.
// Precedence: flags > GALACTICA_* env > YAML file > built-in defaults.
func Load(flags *pflag.FlagSet) (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("GALACTICA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("server.listen", ":8080")
	v.SetDefault("storage.driver", "sqlite")
	v.SetDefault("storage.dsn", "file:/data/galactica.db?cache=shared&_fk=1")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("auth.enabled", false)

	if f := flags.Lookup("listen"); f != nil {
		if err := v.BindPFlag("server.listen", f); err != nil {
			return nil, fmt.Errorf("binding --listen flag: %w", err)
		}
	}
	if f := flags.Lookup("log-level"); f != nil {
		if err := v.BindPFlag("log.level", f); err != nil {
			return nil, fmt.Errorf("binding --log-level flag: %w", err)
		}
	}
	if f := flags.Lookup("log-format"); f != nil {
		if err := v.BindPFlag("log.format", f); err != nil {
			return nil, fmt.Errorf("binding --log-format flag: %w", err)
		}
	}

	if f := flags.Lookup("config"); f != nil && f.Value.String() != "" {
		v.SetConfigFile(f.Value.String())
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("reading config file %q: %w", f.Value.String(), err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}
	return &cfg, nil
}
