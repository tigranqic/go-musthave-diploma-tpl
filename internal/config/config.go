package config

import (
	"flag"
	"os"
	"strconv"
)

type Config struct {
	LogLevel    string
	LogFormat   string
	ServerAddr  string
	DatabaseDSN string
	RateLimit   int
}

const (
	DefaultLogLevel   = "info"
	DefaultLogFormat  = "json"
	DefaultServerAddr = "localhost:8081"
	DefaultDBDSN      = "postgres://postgres:postgres@localhost:15451/go-musthave-diploma-tpl?sslmode=disable"
	DefaultRateLimit  = 5
)

func getenvInt(key string, def int) (int, bool) {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			return n, true
		}
	}
	return def, false
}

func getenvString(key string, def string) (string, bool) {
	if val := os.Getenv(key); val != "" {
		return val, true
	}
	return def, false
}

func Load(isAgent bool) (*Config, error) {
	chooseString := func(envVal string, envSet bool, flagVal string, def string) string {
		if envSet {
			return envVal
		} else if flagVal != "" && flagVal != def {
			return flagVal
		}
		return def
	}

	chooseInt := func(envVal int, envSet bool, flagVal int, def int) int {
		if envSet {
			return envVal
		} else if flagVal != def {
			return flagVal
		}
		return def
	}

	envAddr, envAddrSet := getenvString("RUN_ADDRESS", DefaultServerAddr)
	envDBDSN, envDBDSNSet := getenvString("DATABASE_URI", DefaultDBDSN)
	envRateLimit, envRateLimitSet := getenvInt("RATE_LIMIT", DefaultRateLimit)

	logLevel := flag.String("log-level", DefaultLogLevel, "Log level: debug, info, warn, error")
	logFormat := flag.String("log-format", DefaultLogFormat, "Log format: text or json")
	serverAddrFlag := flag.String("a", DefaultServerAddr, "HTTP server address")
	dbDSNFlag := flag.String("d", DefaultDBDSN, "Database DSN connection string")
	rateLimitFlag := flag.Int("l", DefaultRateLimit, "Rate limit")

	flag.Parse()

	serverAddr := chooseString(envAddr, envAddrSet, *serverAddrFlag, DefaultServerAddr)
	databaseDSN := chooseString(envDBDSN, envDBDSNSet, *dbDSNFlag, DefaultDBDSN)
	rateLimit := chooseInt(envRateLimit, envRateLimitSet, *rateLimitFlag, DefaultRateLimit)

	return &Config{
		LogLevel:    *logLevel,
		LogFormat:   *logFormat,
		ServerAddr:  serverAddr,
		DatabaseDSN: databaseDSN,
		RateLimit:   rateLimit,
	}, nil
}
