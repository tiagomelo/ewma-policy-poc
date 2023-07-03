package digger

import (
	"log"
	"time"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

type Digger struct {
	logger     *log.Logger
	targetHost string
	dnsClient  *dns.Client
}

func New(logger *log.Logger, targetHost string) *Digger {
	return &Digger{
		logger:     logger,
		targetHost: targetHost,
		dnsClient:  new(dns.Client),
	}
}

func (d *Digger) Dig(domain string) error {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	m.RecursionDesired = true

	start := time.Now()
	r, t, err := d.dnsClient.Exchange(m, d.targetHost)
	if err != nil {
		return errors.Wrapf(err, `calling dns.Exchange for domain "%s"`, domain)
	}

	if r.Rcode != dns.RcodeSuccess {
		return errors.Wrapf(err, `doing dns lookup for domain "%s", code: %d`, domain, r.Rcode)
	}

	for _, ans := range r.Answer {
		Arecord, ok := ans.(*dns.A)
		if ok {
			d.logger.Printf("IP address: %s -- query time: %v msec "+
				"-- server: %s#%s (UDP) -- when: %s\n",
				Arecord.A, t.Milliseconds(),
				d.dnsClient.Net,
				d.targetHost, start.Format("Mon Jan _2 15:04:05 -07 2006"))
		}
	}

	return nil
}
