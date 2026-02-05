package config

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	LogLevel      string
	LogFormat     string
	ServerAddr    string
	DatabaseDSN   string
	AccrualAddr   string
	RateLimit     int
	JWTSecret     []byte
	JWTExpiration time.Duration
}

const (
	DefaultLogLevel   = "info"
	DefaultLogFormat  = "json"
	DefaultServerAddr = "localhost:8081"
	DefaultDBDSN      = "postgres://postgres:postgres@localhost:15451/go-musthave-diploma-tpl?sslmode=disable"
	DefaultRateLimit  = 5
	DefaultJWTSecret  = "defaultsecret"
)

func getenvString(key string) (string, bool) {
	val, ok := os.LookupEnv(key)
	return val, ok
}

func getenvInt(key string) (int, bool) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, false
	}
	return n, true
}

func Load() (*Config, error) {
	logLevelFlag := flag.String("log-level", DefaultLogLevel, "Log level: debug, info, warn, error")
	logFormatFlag := flag.String("log-format", DefaultLogFormat, "Log format: text or json")
	serverAddrFlag := flag.String("a", DefaultServerAddr, "HTTP server address")
	dbDSNFlag := flag.String("d", DefaultDBDSN, "Database DSN connection string")
	accrualAddrFlag := flag.String("r", "", "Accrual Address connection string")
	rateLimitFlag := flag.Int("l", DefaultRateLimit, "Rate limit")
	flag.Parse()

	envServerAddr, envServerSet := getenvString("RUN_ADDRESS")
	envDBDSN, envDBSet := getenvString("DATABASE_URI")
	envAccrualAddr, envAccrualSet := getenvString("ACCRUAL_SYSTEM_ADDRESS")
	envRateLimit, envRateSet := getenvInt("RATE_LIMIT")
	envJWTSecret, envJWTSet := getenvString("JWT_SECRET")

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

	serverAddr := chooseString(envServerAddr, envServerSet, *serverAddrFlag, DefaultServerAddr)
	databaseDSN := chooseString(envDBDSN, envDBSet, *dbDSNFlag, DefaultDBDSN)
	accrualAddr := chooseString(envAccrualAddr, envAccrualSet, *accrualAddrFlag, "")
	rateLimit := chooseInt(envRateLimit, envRateSet, *rateLimitFlag, DefaultRateLimit)

	jwtSecret := DefaultJWTSecret
	if envJWTSet && envJWTSecret != "" {
		jwtSecret = envJWTSecret
	} else {
		log.Println("JWT_SECRET not set or empty, using default secret for JWT.")
	}

	return &Config{
		LogLevel:      *logLevelFlag,
		LogFormat:     *logFormatFlag,
		ServerAddr:    serverAddr,
		DatabaseDSN:   databaseDSN,
		AccrualAddr:   accrualAddr,
		RateLimit:     rateLimit,
		JWTSecret:     []byte(jwtSecret),
		JWTExpiration: 24 * time.Hour,
	}, nil
}
