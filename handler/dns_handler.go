package handler

import (
	"log"
	"strings"

	"github.com/miekg/dns"
	"dnslite/cache"
	"dnslite/db"
	"dnslite/dnssec"
)

func StartDNSServers(addr string) {
	dns.HandleFunc(".", handleDNS)

	go func() {
		log.Println("Starting UDP DNS on", addr)
		log.Fatal((&dns.Server{Addr: addr, Net: "udp"}).ListenAndServe())
	}()

	log.Println("Starting TCP DNS on", addr)
	log.Fatal((&dns.Server{Addr: addr, Net: "tcp"}).ListenAndServe())
}

func handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	msg.Authoritative = true

	for _, q := range r.Question {
		name := strings.ToLower(dns.Fqdn(q.Name))
		qtype := q.Qtype

		records := cache.Get(name, qtype)
		if records == nil {
			dbRecords, err := db.QueryRecords(name, qtype)
			if err != nil {
				log.Printf("DB error: %v", err)
				continue
			}
			cache.Set(name, qtype, dbRecords)
			records = dbRecords
		}

		if len(records) > 0 {
			// Find matching zone
			zone := findZoneFor(name)

			// Try to fetch precomputed RRSIG from DB
			sig, err := db.QueryRRSIG(name, qtype)
			if err != nil && zone != "" {
				sigRR, signErr := dnssec.SignRRSet(records, zone)
				if signErr == nil {
					_ = db.StoreRRSIG(name, qtype, sigRR)
					records = append(records, sigRR)
				}
			} else if err == nil {
				records = append(records, sig)
			}
		}

		msg.Answer = append(msg.Answer, records...)
	}

	// Always respond with DNSKEY for matching zone
	for _, q := range r.Question {
		if q.Qtype == dns.TypeDNSKEY {
			zone := findZoneFor(q.Name)
			if kp := dnssec.GetKeyPair(zone); kp != nil && kp.Public.Header().Name == q.Name {
				msg.Answer = append(msg.Answer, kp.Public)
			}
		}
	}

	w.WriteMsg(&msg)
}

// Matches the most specific zone that ends with qname
func findZoneFor(name string) string {
	name = dns.Fqdn(name)
	var match string
	for _, zone := range dnssec.GetAllZones() {
		if strings.HasSuffix(name, zone) && len(zone) > len(match) {
			match = zone
		}
	}
	return match
}
