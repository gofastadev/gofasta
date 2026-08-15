// Package config loads configuration from config.yaml and environment
// variables. It provides LoadConfig() to get an AppConfig struct with all
// settings, and SetupDB() to create a database connection. Supports Postgres,
// MySQL, SQLite, SQL Server, and ClickHouse.
package config
