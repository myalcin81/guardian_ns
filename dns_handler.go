package main

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"golang.org/x/time/rate"

	"github.com/asalih/guardian_ns/data"
	"github.com/asalih/guardian_ns/models"
	"github.com/miekg/dns"
)

//DNSHandler the dns handler
type DNSHandler struct {
	Targets       map[string]net.IP
	DBHelper      *data.DNSDBHelper
	IPRateLimiter *models.IPRateLimiter

	mutex sync.Mutex
}

//NewDNSHandler Init dns handler
func NewDNSHandler() *DNSHandler {
	handler := &DNSHandler{nil, &data.DNSDBHelper{},
		models.NewIPRateLimiter(rate.Limit(models.Configuration.RateLimitSec),
			models.Configuration.RateLimitBurst),
		sync.Mutex{}}

	handler.LoadTargets()

	return handler
}

//ServeDNS ...
func (h *DNSHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)

	if !h.IPRateLimiter.IsAllowed(w.RemoteAddr().String()) {
		w.WriteMsg(&msg)
		go h.DBHelper.LogThrottleRequest(w.RemoteAddr().String())

		return
	}

	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		target := strings.ToLower(msg.Question[0].Name)
		address, ok := h.Targets[target]

		fmt.Println("Requested: " + target)
		if ok {
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: target, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
				A:   address,
			})
		}
	}
	w.WriteMsg(&msg)
}

//LoadTargets ...
func (h *DNSHandler) LoadTargets() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.Targets = h.DBHelper.GetTargetsList()
	h.Targets["ntp.ubuntu.com."] = net.ParseIP("91.189.91.157")
}
