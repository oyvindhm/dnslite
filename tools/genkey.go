package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
	"github.com/miekg/dns"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run tools/genkey.go <zone>")
		os.Exit(1)
	}

	zone := dns.Fqdn(os.Args[1])
	ttl := 3600

	// 1. Generate RSA Key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	// 2. Prepare secret dir
	dir := filepath.Join("secrets", zone)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		panic(err)
	}

	// 3. Save private key
	privPath := filepath.Join(dir, "key.pem")
	privOut, err := os.Create(privPath)
	if err != nil {
		panic(err)
	}
	defer privOut.Close()
	_ = pem.Encode(privOut, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})

	// 4. Build DNSKEY
	pubBytes := x509.MarshalPKCS1PublicKey(&priv.PublicKey)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubBytes)

	dnskey := &dns.DNSKEY{
		Hdr: dns.RR_Header{
			Name:   zone,
			Rrtype: dns.TypeDNSKEY,
			Class:  dns.ClassINET,
			Ttl:    uint32(ttl),
		},
		Flags:     256,
		Protocol:  3,
		Algorithm: dns.RSASHA256,
		PublicKey: pubKeyB64,
	}

	// 5. Save public key as dnskey.txt
	pubPath := filepath.Join(dir, "dnskey.txt")
	pubOut, err := os.Create(pubPath)
	if err != nil {
		panic(err)
	}
	defer pubOut.Close()
	_, _ = pubOut.WriteString(dnskey.String() + "\n")

	fmt.Println("✅ Key pair generated for", zone)

	// 6. Insert into database
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		fmt.Println("❌ DB_URL not set in env")
		return
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		panic("❌ DB connect failed: " + err.Error())
	}
	defer conn.Close(context.Background())

	var zoneID int
	err = conn.QueryRow(context.Background(), `
		INSERT INTO zones (name)
		VALUES ($1)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, zone).Scan(&zoneID)
	if err != nil {
		panic("❌ Failed to insert zone: " + err.Error())
	}

	// 7. Insert DNSKEY into records
	_, err = conn.Exec(context.Background(), `
		INSERT INTO records (zone_id, name, type, ttl, data)
		VALUES ($1, $2, 'DNSKEY', $3, $4)
		ON CONFLICT DO NOTHING
	`, zoneID, zone, ttl, pubKeyB64)
	if err != nil {
		panic("❌ Failed to insert DNSKEY record: " + err.Error())
	}

	fmt.Println("✅ DNSKEY inserted into records for zone:", zone)
}
