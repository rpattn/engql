package config

import (
	"fmt"

	"github.com/rpattn/engql/internal/db"
	"github.com/spf13/viper"
)

func LoadDBConfig(configPath string) (db.Config, error) {
	// Start with default
	cfg := db.DefaultConfig()

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)
	v.AutomaticEnv()     // allow environment overrides
	v.SetEnvPrefix("DB") // map env vars like DB_HOST, DB_PORT

	// Optional: Map nested keys to flat env vars
	v.BindEnv("database.host")
	v.BindEnv("database.port")
	v.BindEnv("database.user")
	v.BindEnv("database.password")
	v.BindEnv("database.dbname")
	v.BindEnv("database.sslmode")

	if err := v.ReadInConfig(); err != nil {
		// Config file not found? Just log it, use defaults + env
		fmt.Println("No config.yaml found, using defaults and env vars")
	} else {
		fmt.Println("Loaded config.yaml")
	}

	// Override defaults if values exist
	if v.IsSet("database.host") {
		cfg.Host = v.GetString("database.host")
	}
	if v.IsSet("database.port") {
		cfg.Port = v.GetInt("database.port")
	}
	if v.IsSet("database.user") {
		cfg.User = v.GetString("database.user")
	}
	if v.IsSet("database.password") {
		cfg.Password = v.GetString("database.password")
	}
	if v.IsSet("database.dbname") {
		cfg.DBName = v.GetString("database.dbname")
	}
	if v.IsSet("database.sslmode") {
		cfg.SSLMode = v.GetString("database.sslmode")
	}

	return cfg, nil
}
