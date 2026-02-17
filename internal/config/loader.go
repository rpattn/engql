package config

import (
	"fmt"
	"strings"

	"github.com/rpattn/engql/internal/db"
	"github.com/spf13/viper"
)

func LoadDBConfig(configPath string) (db.Config, error) {
	cfg := db.DefaultConfig()

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)
	
	// 1. Allow reading from Environment Variables
	v.AutomaticEnv()
    // Optional: Keep this if you want, but explicit binding below is clearer
	v.SetEnvPrefix("ENGQL") 
    // Replace dots with underscores for auto-matching (e.g. database.host -> ENGQL_DATABASE_HOST)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) 

	// 2. EXPLICITLY bind simple Env Vars to internal Config keys
	// This lets you set "DB_HOST" in Coolify instead of "ENGQL_DATABASE_HOST"
	_ = v.BindEnv("database.host", "DB_HOST")
	_ = v.BindEnv("database.port", "DB_PORT")
	_ = v.BindEnv("database.user", "DB_USER")
	_ = v.BindEnv("database.password", "DB_PASSWORD")
	_ = v.BindEnv("database.dbname", "DB_NAME")
	_ = v.BindEnv("database.sslmode", "DB_SSLMODE")

	if err := v.ReadInConfig(); err != nil {
		fmt.Println("No config.yaml found, using environment variables")
	}

	// 3. Populate struct
    // Viper will now check: Env Var (DB_HOST) -> Config File -> Default
	if v.IsSet("database.host") { cfg.Host = v.GetString("database.host") }
	if v.IsSet("database.port") { cfg.Port = v.GetInt("database.port") }
	if v.IsSet("database.user") { cfg.User = v.GetString("database.user") }
	if v.IsSet("database.password") { cfg.Password = v.GetString("database.password") }
	if v.IsSet("database.dbname") { cfg.DBName = v.GetString("database.dbname") }
	if v.IsSet("database.sslmode") { cfg.SSLMode = v.GetString("database.sslmode") }

	return cfg, nil
}