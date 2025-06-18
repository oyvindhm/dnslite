package dnssec

import (
	"time"

	"github.com/miekg/dns"
)

func SignRRSet(rrset []dns.RR, zone string) (*dns.RRSIG, error) {
	keypair := GetKeyPair(zone)
	if keypair == nil {
		return nil, nil
	}

	sig := &dns.RRSIG{
		Hdr: dns.RR_Header{
			Name:   rrset[0].Header().Name,
			Rrtype: dns.TypeRRSIG,
			Class:  dns.ClassINET,
			Ttl:    rrset[0].Header().Ttl,
		},
		TypeCovered: rrset[0].Header().Rrtype,
		Algorithm:   keypair.Public.Algorithm,
		Labels: uint8(dns.CountLabel(rrset[0].Header().Name)),
		OrigTtl:     rrset[0].Header().Ttl,
		Expiration:  uint32(time.Now().Add(24 * time.Hour).Unix()),
		Inception:   uint32(time.Now().Add(-5 * time.Minute).Unix()),
		KeyTag:      keypair.Public.KeyTag(),
		SignerName:  dns.Fqdn(zone),
	}
	if err := sig.Sign(keypair.Private, rrset); err != nil {
		return nil, err
	}
	return sig, nil
}
