package db

import (
	"context"
	"log"

	"github.com/jackc/pgconn"
	"dnslite/cache"
)

// WatchForChanges listens for PostgreSQL NOTIFY on "record_change" and clears cache when triggered
func WatchForChanges() {
	// Connect a separate raw connection for LISTEN
	connRaw, err := pgconn.Connect(context.Background(), conn.Config().ConnString())
	if err != nil {
		log.Printf("❌ Failed to connect for NOTIFY: %v", err)
		return
	}
	defer connRaw.Close(context.Background())

	_, err = connRaw.Exec(context.Background(), "LISTEN record_change").ReadAll()
	if err != nil {
		log.Printf("❌ Failed to LISTEN on channel: %v", err)
		return
	}

	log.Println("📡 Listening for DB changes to invalidate cache...")

	for {
		notification := connRaw.WaitForNotification(context.Background())
		log.Println("🔁 Detected DB change via NOTIFY — clearing cache")
		cache.Clear()
		_ = notification // you can log this if desired
	}
}
