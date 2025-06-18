package db

import (
	"context"
	"log"
	"strings"
)

func TruncateAll() error {
	_, err := conn.Exec(context.Background(), `
		TRUNCATE TABLE dnssec_rrsigs, records, zones RESTART IDENTITY CASCADE;
	`)
	return err
}


func Migrate() {
		stmts := []string{
		`CREATE TABLE IF NOT EXISTS zones (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			ttl INT DEFAULT 3600
		);`,

		`CREATE TABLE IF NOT EXISTS records (
			id SERIAL PRIMARY KEY,
			zone_id INT REFERENCES zones(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			ttl INT DEFAULT 3600,
			data TEXT NOT NULL
		);`,

		`CREATE UNIQUE INDEX IF NOT EXISTS idx_records_unique ON records(name, type, data);`,

		`CREATE INDEX IF NOT EXISTS idx_records_name_type ON records(name, type);`,

		`CREATE TABLE IF NOT EXISTS dnssec_rrsigs (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			type_covered TEXT NOT NULL,
			rrsig TEXT NOT NULL
		);`,

		`CREATE UNIQUE INDEX IF NOT EXISTS idx_rrsig_name_type ON dnssec_rrsigs(name, type_covered);`,

		`CREATE OR REPLACE FUNCTION notify_record_change()
			RETURNS trigger AS $$
			BEGIN
			PERFORM pg_notify('record_change', '');
			RETURN NULL;
			END;
		$$ LANGUAGE plpgsql;`,

		`CREATE TRIGGER record_insert AFTER INSERT ON records FOR EACH STATEMENT EXECUTE FUNCTION notify_record_change();`,
		`CREATE TRIGGER record_update AFTER UPDATE ON records FOR EACH STATEMENT EXECUTE FUNCTION notify_record_change();`,
		`CREATE TRIGGER record_delete AFTER DELETE ON records FOR EACH STATEMENT EXECUTE FUNCTION notify_record_change();`,
	}

	for _, stmt := range stmts {
		_, err := conn.Exec(context.Background(), stmt)
		if err != nil {
			log.Fatalf("Migration failed: %v\nQuery: %s", err, stmt)
		}
	}

	// Optional: Add a zone-aware unique constraint on records
	_, err := conn.Exec(context.Background(), `
		ALTER TABLE records
		ADD CONSTRAINT unique_record_entry
		UNIQUE (zone_id, name, type, data)
	`)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Fatalf("Failed to add unique constraint: %v", err)
	}

	log.Println("✅ Database schema migration completed.")
}
