package dnssec

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/miekg/dns"
)

type KeyPair struct {
	Private *rsa.PrivateKey
	Public  *dns.DNSKEY
}

var zoneKeys = map[string]*KeyPair{}

func LoadAllZoneKeys(secretsDir string) error {
	entries, err := os.ReadDir(secretsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		zone := dns.Fqdn(entry.Name())
		pubPath := filepath.Join(secretsDir, entry.Name(), "dnskey.txt")
		privPath := filepath.Join(secretsDir, entry.Name(), "key.pem")

		keypair, err := loadKeyPair(pubPath, privPath)
		if err != nil {
			continue
		}
		zoneKeys[zone] = keypair
	}
	return nil
}

func loadKeyPair(pubPath, privPath string) (*KeyPair, error) {
	privData, err := ioutil.ReadFile(privPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(privData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("invalid PEM private key")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pubData, err := ioutil.ReadFile(pubPath)
	if err != nil {
		return nil, err
	}
	pubRR, err := dns.NewRR(string(pubData))
	if err != nil {
		return nil, err
	}
	dnskey, ok := pubRR.(*dns.DNSKEY)
	if !ok {
		return nil, errors.New("invalid DNSKEY")
	}

	return &KeyPair{Private: priv, Public: dnskey}, nil
}

func GetKeyPair(zone string) *KeyPair {
	return zoneKeys[dns.Fqdn(zone)]
}

func GetAllZones() []string {
	keys := make([]string, 0, len(zoneKeys))
	for zone := range zoneKeys {
		keys = append(keys, zone)
	}
	return keys
}
