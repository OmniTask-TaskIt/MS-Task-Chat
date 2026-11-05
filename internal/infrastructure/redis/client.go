package redis

import (
	"crypto/tls"
	"net"
	"os"
	"strconv"

	goredis "github.com/redis/go-redis/v9"
)

func NewClientFromEnv() *goredis.Client {
	db, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	options := &goredis.Options{
		Addr:     envOrDefault("REDIS_ADDR", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	}
	if os.Getenv("REDIS_TLS") == "true" {
		options.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: redisHost(options.Addr),
		}
	}
	return goredis.NewClient(options)
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func redisHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
