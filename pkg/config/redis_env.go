package config

import (
	"os"
	"strconv"
	"strings"
)

// RedisFromEnv builds a RedisConfig from REDIS_URL, REDIS_PASSWORD and
// REDIS_DB.
//
// REDIS_URL is a bare "host:port" — the form a container network uses — rather
// than a redis:// URL, because that is what orchestrators inject and what
// every consumer of this config already expects. A missing port defaults to
// 6379; an unparseable REDIS_DB is 0, which is the database a single-tenant
// deployment uses anyway.
//
// Provided so services sharing one Redis do not each re-implement the split.
// Projects configuring Redis through config.yaml should read the relevant
// section instead; this is for the ones reading the environment directly.
func RedisFromEnv() RedisConfig {
	host, port, _ := strings.Cut(os.Getenv("REDIS_URL"), ":")
	if port == "" {
		port = "6379"
	}
	db, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	return RedisConfig{
		Host:     host,
		Port:     port,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	}
}
