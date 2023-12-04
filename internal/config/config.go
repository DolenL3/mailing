package config

import "os"

// DBConfig is config with sensitive data, needed for working with db.
type DBConfig struct {
	User         string
	Host         string
	DBName       string
	Password     string
	MigrationURL string
}

// NewDBConfig returns DBConfig with sensitive data, needed for working with db.
func NewDBConfig() *DBConfig {
	return &DBConfig{
		User:         os.Getenv("POSTGRES_USER"),
		Host:         os.Getenv("POSTGRES_HOST"),
		DBName:       os.Getenv("POSTGRES_DB"),
		Password:     os.Getenv("POSTGRES_PASSWORD"),
		MigrationURL: os.Getenv("MIGRATION_URL"),
	}
}

// SenderConfig is config with sensitive data, needed for working with sender.
type SenderConfig struct {
	JWT string
}

// NewSenderConfig returns SenderConfig with sensitive data, needed for working with sender.
func NewSenderConfig() *SenderConfig {
	return &SenderConfig{
		JWT: os.Getenv("SENDER_JWT"),
	}
}

// HTTPConfig is config with sensitive data, needed for rest API.
type HTTPConfig struct {
	Host string
}

// NewHTTPConfig returns HTTPConfig with sensitive data, needed for rest API.
func NewHTTPConfig() *HTTPConfig {
	return &HTTPConfig{
		Host: os.Getenv("HTTP_HOST"),
	}
}
