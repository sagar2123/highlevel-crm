package config

import (
	"os"
	"strconv"
)

type Config struct {
	App App
	DB  DB
	ES  ES
}

type App struct {
	Port string
}

type DB struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
	MaxIdle  int
	MaxOpen  int
}

type ES struct {
	URL string
}

func Load() Config {
	return Config{
		App: App{
			Port: getEnv("APP_PORT", "8080"),
		},
		DB: DB{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     getEnv("DB_NAME", "crm_platform"),
			User:     getEnv("DB_USER", "app_user"),
			Password: getEnv("DB_PASSWORD", "app_password"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
			MaxIdle:  getEnvInt("DB_MAX_IDLE", 5),
			MaxOpen:  getEnvInt("DB_MAX_OPEN", 20),
		},
		ES: ES{
			URL: getEnv("ES_URL", "http://localhost:9200"),
		},
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}
