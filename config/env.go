package config

import (
	"log"
	"os"
)

var (
	DBURL string
)

func LoadEnv() {
	DBURL = os.Getenv("DB_URL")
	if DBURL == "" {
		log.Fatal("DB_URL environment variable is not set")
	}
}
