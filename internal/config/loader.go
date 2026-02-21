package config

import (
	"fmt"
	"strings"

	"github.com/rpattn/engql/internal/db"
	"github.com/spf13/viper"
)

type Config struct {
	Database         db.Config
	EnablePlayground bool
	ServerPort       string
}

func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = v.BindEnv("database.host", "DB_HOST")
	_ = v.BindEnv("database.port", "DB_PORT")
	_ = v.BindEnv("database.user", "DB_USER")
	_ = v.BindEnv("database.password", "DB_PASSWORD")
	_ = v.BindEnv("database.dbname", "DB_NAME")
	_ = v.BindEnv("database.sslmode", "DB_SSLMODE")

	_ = v.BindEnv("options.enable_playground", "ENABLE_PLAYGROUND")

	_ = v.BindEnv("server.port", "PORT")

	// 4. Set Defaults
	v.SetDefault("server.port", "8080")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("options.enable_playground", false)

	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Note: No config.yaml found at %s, relying on defaults and environment variables\n", configPath)
	}

	cfg := &Config{}

	cfg.Database = db.Config{
		Host:     v.GetString("database.host"),
		Port:     v.GetInt("database.port"),
		User:     v.GetString("database.user"),
		Password: v.GetString("database.password"),
		DBName:   v.GetString("database.dbname"),
		SSLMode:  v.GetString("database.sslmode"),
	}

	cfg.EnablePlayground = v.GetBool("options.enable_playground")
	cfg.ServerPort = v.GetString("server.port")

	if cfg.Database.Host == "" && v.GetString("DB_HOST") == "" {
		fmt.Println("Warning: Database host is not set.")
	}

	return cfg, nil
}