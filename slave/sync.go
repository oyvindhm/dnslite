package slave

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"dnslite/db"
	"github.com/miekg/dns"
	"dnslite/api"
)

func StartSlaveSync(masterURL string, interval time.Duration) {
	api.UpdateLastSync(time.Now())
	go func() {
		for {
			SyncFromMaster(masterURL)
			time.Sleep(interval)
		}
	}()
}

func SyncFromMaster(masterURL string) {
	resp, err := http.Get(masterURL)
	if err != nil {
		log.Println("❌ Failed to contact master:", err)
		return
	}
	defer resp.Body.Close()

	type ZoneFile struct {
		Zone    string   `json:"zone"`
		Records []string `json:"records"`
	}

	var zones []ZoneFile
	if err := json.NewDecoder(resp.Body).Decode(&zones); err != nil {
		log.Println("❌ Failed to decode master response:", err)
		return
	}

	synced := 0
	for _, z := range zones {
		log.Printf("📥 Processing zone: %s", z.Zone)

		_, err := db.InsertZone(z.Zone)
		if err != nil {
			log.Printf("⚠️ Could not insert or find zone %s: %v", z.Zone, err)
			continue
		}

		for _, rrStr := range z.Records {
			rr, err := dns.NewRR(rrStr)
			if err != nil {
				log.Printf("⚠️ Invalid RR in zone %s: %s (%v)", z.Zone, rrStr, err)
				continue
			}

			name := rr.Header().Name
			qtype := rr.Header().Rrtype

			if qtype == dns.TypeRRSIG {
				err = db.StoreRRSIG(name, qtype, rr)
				if err != nil {
					log.Printf("❌ Failed to store RRSIG for %s: %v", name, err)
				}
			} else {
				err = db.UpsertRecord(name, qtype, rr)
				if err != nil {
					log.Printf("❌ Failed to upsert RR %s (%s): %v", name, dns.TypeToString[qtype], err)
				}
			}
		}
		synced++
	}

	log.Printf("🔄 Synced %d zones from master", synced)
}

