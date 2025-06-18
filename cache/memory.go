package cache

import (
	"fmt"
	"sync"

	"github.com/miekg/dns"
)

var recordCache sync.Map

func key(name string, qtype uint16) string {
	return fmt.Sprintf("%s:%d", name, qtype)
}

func Get(name string, qtype uint16) []dns.RR {
	val, ok := recordCache.Load(key(name, qtype))
	if !ok {
		return nil
	}
	return val.([]dns.RR)
}

func Set(name string, qtype uint16, records []dns.RR) {
	recordCache.Store(key(name, qtype), records)
}
