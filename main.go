package main

import (
	"log"
	"os"
	"time"

	"dnslite/config"
	"dnslite/db"
	"dnslite/dnssec"
	"dnslite/handler"
	"dnslite/api"
	"dnslite/slave"
)

func main() {
	config.LoadEnv()
	db.Connect(config.DBURL)
	db.Migrate()
	defer db.Close()

	role := os.Getenv("SERVER_ROLE")
	switch role {
	case "master":
		log.Println("🧠 Running in MASTER mode")
		if err := dnssec.LoadAllZoneKeys("secrets"); err != nil {
			log.Fatalf("DNSSEC load failed: %v", err)
		}
		api.StartAPIServer(":8080")

	case "slave":
		log.Println("🧠 Running in SLAVE mode")
		err := db.TruncateAll()
		if err != nil {
			log.Fatalf("❌ Failed to truncate slave DB: %v", err)
		}
		slave.StartSlaveSync(os.Getenv("MASTER_URL"), 5*time.Minute)

	default:
		log.Fatalf("SERVER_ROLE must be set to 'master' or 'slave'")
	}

	handler.StartDNSServers(":53")
}
