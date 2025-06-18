package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"dnslite/db"
	"dnslite/dnssec"
)

var (
	lastSlaveSync time.Time
	syncMu        sync.RWMutex
)

type ZoneFile struct {
	Zone    string   `json:"zone"`
	Records []string `json:"records"`
}

func StartAPIServer(addr string) {
	http.HandleFunc("/zone-sync", handleZoneSync)
	http.HandleFunc("/status", handleStatus)
	go http.ListenAndServe(addr, nil)
}

func handleZoneSync(w http.ResponseWriter, r *http.Request) {
	log.Println("▶ Loading zones from database...")
	zones, err := db.GetAllZoneNames()
	if err != nil {
		log.Println("❌ Failed to load zones:", err)
		http.Error(w, "Failed to load zones", http.StatusInternalServerError)
		return
	}
	log.Printf("✅ Loaded %d zones\n", len(zones))

	var zoneFiles []ZoneFile

	for _, zone := range zones {
		log.Printf("Manually checking RRSet keys for %s", zone)
		pairs, err := db.GetRRSetKeysForZone(zone)
		log.Printf("%s pairs = %+v, err = %v", zone, pairs, err)
		if err != nil {
			log.Printf("⚠️ Could not load RRSetKeys for %s: %v\n", zone, err)
			continue
		}
		log.Printf("🧾 Zone %s has %d RRSet keys\n", zone, len(pairs))

		var zoneRecords []string
		for _, p := range pairs {
			rrset, err := db.QueryRecords(p.Name, p.Type)
			if err != nil {
				log.Printf("⚠️ QueryRecords error: %v", err)
				continue
			}
			sig, _ := db.QueryRRSIG(p.Name, p.Type)

			for _, rr := range rrset {
				log.Println("📦 RR:", rr.String())
				zoneRecords = append(zoneRecords, rr.String())
			}
			if sig != nil {
				log.Println("🔐 SIG:", sig.String())
				zoneRecords = append(zoneRecords, sig.String())
			}
		}

		if len(zoneRecords) > 0 {
			zoneFiles = append(zoneFiles, ZoneFile{
				Zone:    zone,
				Records: zoneRecords,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(zoneFiles)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	role := os.Getenv("SERVER_ROLE")
	response := map[string]any{
		"role": role,
	}

	if role == "master" {
		response["dnssec_zones"] = dnssec.GetAllZones()
		dbZones, _ := db.GetAllZoneNames()
		response["db_zones"] = dbZones
	} else if role == "slave" {
		syncMu.RLock()
		response["last_sync"] = lastSlaveSync.Format(time.RFC3339)
		syncMu.RUnlock()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func UpdateLastSync(t time.Time) {
	syncMu.Lock()
	lastSlaveSync = t
	syncMu.Unlock()
}
