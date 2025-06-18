package db

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/miekg/dns"
)

var conn *pgx.Conn

func Connect(url string) {
	var err error
	conn, err = pgx.Connect(context.Background(), url)
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}
}

func Close() {
	conn.Close(context.Background())
}

// QueryRecords returns all RRs of a name/qtype
func QueryRecords(name string, qtype uint16) ([]dns.RR, error) {
	name = dns.Fqdn(strings.ToLower(name))

	rows, err := conn.Query(context.Background(), `
		SELECT type, ttl, data FROM records
		WHERE name = $1 AND type = $2
	`, name, dns.TypeToString[qtype])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []dns.RR
	for rows.Next() {
		var rtype, data string
		var ttl int
		if err := rows.Scan(&rtype, &ttl, &data); err != nil {
			continue
		}

		rrStr := fmt.Sprintf("%s %d IN %s %s", name, ttl, rtype, data)
		rr, err := dns.NewRR(rrStr)
		if err != nil {
			log.Println("Failed to parse RR:", rrStr, err)
			continue
		}
		results = append(results, rr)
	}
	return results, nil
}

// QueryRRSIG returns a single matching RRSIG
func QueryRRSIG(name string, qtype uint16) (dns.RR, error) {
	name = dns.Fqdn(strings.ToLower(name))

	row := conn.QueryRow(context.Background(), `
		SELECT rrsig FROM dnssec_rrsigs
		WHERE name = $1 AND type_covered = $2
	`, name, dns.TypeToString[qtype])

	var rrsigStr string
	if err := row.Scan(&rrsigStr); err != nil {
		return nil, err
	}

	return dns.NewRR(rrsigStr)
}

// StoreRRSIG inserts or updates an RRSIG
func StoreRRSIG(name string, qtype uint16, rrsig dns.RR) error {
	name = dns.Fqdn(strings.ToLower(name))

	_, err := conn.Exec(context.Background(), `
		INSERT INTO dnssec_rrsigs (name, type_covered, rrsig)
		VALUES ($1, $2, $3)
		ON CONFLICT (name, type_covered) DO UPDATE SET rrsig = EXCLUDED.rrsig
	`, name, dns.TypeToString[qtype], rrsig.String())

	return err
}

// InsertZone adds or gets a zone ID
func InsertZone(name string) (int, error) {
	name = dns.Fqdn(strings.ToLower(name))

	var id int
	err := conn.QueryRow(context.Background(), `
		INSERT INTO zones (name)
		VALUES ($1)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, name).Scan(&id)
	return id, err
}

func GetRRSetKeysForZone(zone string) ([]RRSetKey, error) {
	normalized := strings.TrimSuffix(zone, ".")
	log.Printf("🔍 Querying RRSetKeys for zone: input='%s', normalized='%s'\n", zone, normalized)

	rows, err := conn.Query(context.Background(), `
		SELECT r.name, r.type FROM records r
		JOIN zones z ON r.zone_id = z.id
		WHERE z.name = $1 OR z.name = $1 || '.'
	`, normalized)
	if err != nil {
		log.Printf("❌ Query error for zone '%s': %v\n", normalized, err)
		return nil, err
	}
	defer rows.Close()

	var keys []RRSetKey
	for rows.Next() {
		var name, typeStr string
		err = rows.Scan(&name, &typeStr)
		if err != nil {
			log.Printf("⚠️ Failed to scan RRSetKey row: %v\n", err)
			continue
		}

		qtype := dns.StringToType[typeStr]
		log.Printf("📦 Found RRSetKey: name='%s', type=%d (%s)\n", name, qtype, typeStr)

		keys = append(keys, RRSetKey{
			Name: name,
			Type: qtype,
		})
	}

	log.Printf("✅ Total %d RRSetKeys found for zone '%s'\n", len(keys), normalized)
	return keys, nil
}


func UpsertRecord(name string, qtype uint16, rr dns.RR) error {
	name = dns.Fqdn(strings.ToLower(name))

	// Get the correct zone_id
	var zoneID int
	err := conn.QueryRow(context.Background(), `
		SELECT id FROM zones
		WHERE $1 LIKE '%' || name
	`, name).Scan(&zoneID)
	if err != nil {
		return err
	}

	var data string
	switch r := rr.(type) {
	case *dns.A:
		data = r.A.String()
	case *dns.AAAA:
		data = r.AAAA.String()
	case *dns.CNAME:
		data = r.Target
	case *dns.MX:
		data = fmt.Sprintf("%d %s", r.Preference, r.Mx)
	case *dns.TXT:
		data = strings.Join(r.Txt, " ")
	case *dns.NS:
		data = r.Ns
	case *dns.DNSKEY:
		data = fmt.Sprintf("%d %d %d %s", r.Flags, r.Protocol, r.Algorithm, r.PublicKey)
	default:
		// Fallback to full string minus first 4 fields (name ttl class type)
		parts := strings.Fields(rr.String())
		if len(parts) >= 5 {
			data = strings.Join(parts[4:], " ")
		} else {
			return fmt.Errorf("unhandled RR format: %s", rr.String())
		}
	}

	_, err = conn.Exec(context.Background(), `
		INSERT INTO records (zone_id, name, type, ttl, data)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (name, type, data) DO NOTHING
	`, zoneID, name, dns.TypeToString[qtype], rr.Header().Ttl, data)

	return err
}

// GetAllZoneNames returns all zone names
func GetAllZoneNames() ([]string, error) {
	rows, err := conn.Query(context.Background(), `SELECT name FROM zones`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var zones []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		zones = append(zones, name)
	}
	return zones, nil
}

// RRSetKey identifies a unique name/type pair
type RRSetKey struct {
	Name string
	Type uint16
}
