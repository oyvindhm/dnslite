package main

import (
	"log"
	"os"

	"dnslite/db"
	"dnslite/dnssec"

	"github.com/miekg/dns"
)

func main() {
	err := dnssec.LoadAllZoneKeys("secrets")
	if err != nil {
		log.Fatal("Failed to load keys:", err)
	}

	db.Connect(os.Getenv("DB_URL"))
	defer db.Close()

	zones := dnssec.GetAllZones()
	total := 0

	for _, zone := range zones {
		pairs, err := db.GetRRSetKeysForZone(zone)
		if err != nil {
			log.Printf("Skipping %s: %v", zone, err)
			continue
		}

		for _, p := range pairs {
			rrset, err := db.QueryRecords(p.Name, p.Type)
			if err != nil || len(rrset) == 0 {
				continue
			}
			sig, err := dnssec.SignRRSet(rrset, zone)
			if err != nil {
				log.Printf("Sign error for %s %s: %v", p.Name, dns.TypeToString[p.Type], err)
				continue
			}
			err = db.StoreRRSIG(p.Name, p.Type, sig)
			if err != nil {
				log.Printf("Store error for %s: %v", p.Name, err)
			} else {
				total++
			}
		}
	}

	log.Printf("✅ Re-signed %d RRSIGs across %d zones", total, len(zones))
}
