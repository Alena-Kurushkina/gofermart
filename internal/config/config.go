package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress string
	DatabaseURI string
	AccrualAddress string 
}

func InitConfig() *Config {
	cfg := Config{}

	// define flags
	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8081", "address of the server")
	flag.StringVar(&cfg.DatabaseURI, "d", "host=127.0.0.1 user=practicum password=123456 dbname=gophermart sslmode=disable", "connection string to database")	
	flag.StringVar(&cfg.AccrualAddress, "r", "localhost:8080", "address of accrual server")

	// parse flags
	flag.Parse()

	// read environment variables
	sa, exists := os.LookupEnv("RUN_ADDRESS")
	if exists {
		cfg.ServerAddress = sa
	}
	bu, exists := os.LookupEnv("DATABASE_URI")
	if exists {
		cfg.DatabaseURI = bu
	}
	fu, exists := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS")
	if exists {
		cfg.AccrualAddress = fu
	}
	
	return &cfg
}
