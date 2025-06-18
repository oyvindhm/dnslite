package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"dnslite/db"
	"dnslite/dnssec"

	"github.com/miekg/dns"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run tools/signzone.go <zone>")
		os.Exit(1)
	}
	zone := dns.Fqdn(os.Args[1])

	// Load keys
	err := dnssec.LoadKeys("secrets/dnskey.txt", "secrets/key.pem")
	if err != nil {
		log.Fatalf("Failed to load DNSSEC keys: %v", err)
	}

	// Connect to DB
	db.Connect(os.Getenv("DB_URL"))
	defer db.Close()

	// Fetch all (name, type) pairs for the zone
	pairs, err := db.GetRRSetKeysForZone(zone)
	if err != nil {
		log.Fatalf("Failed to fetch RRSet keys: %v", err)
	}

	signed := 0
	for _, p := range pairs {
		name := p.Name
		qtype := p.Type

		rrset, err := db.QueryRecords(name, qtype)
		if err != nil || len(rrset) == 0 {
			continue
		}

		sig, err := dnssec.SignRRSet(rrset, zone)
		if err != nil {
			log.Printf("Sign error for %s %s: %v", name, dns.TypeToString[qtype], err)
			continue
		}

		err = db.StoreRRSIG(name, qtype, sig)
		if err != nil {
			log.Printf("Store RRSIG error: %v", err)
			continue
		}

		log.Printf("✔ Signed %s %s", name, dns.TypeToString[qtype])
		signed++
	}

	log.Printf("✅ Signed %d RRSIGs for zone %s", signed, zone)
}
