package forward

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/coredns/coredns/plugin/pkg/proxy"
	"github.com/coredns/coredns/plugin/pkg/rand"
	"github.com/miekg/dns"
)

// Policy defines a policy we use for selecting upstreams.
type Policy interface {
	List([]*proxy.Proxy) []*proxy.Proxy
	String() string
}

// random is a policy that implements random upstream selection.
type random struct{}

func (r *random) String() string { return "random" }

func (r *random) List(p []*proxy.Proxy) []*proxy.Proxy {
	switch len(p) {
	case 1:
		return p
	case 2:
		if rn.Int()%2 == 0 {
			return []*proxy.Proxy{p[1], p[0]} // swap
		}
		return p
	}

	perms := rn.Perm(len(p))
	rnd := make([]*proxy.Proxy, len(p))

	for i, p1 := range perms {
		rnd[i] = p[p1]
	}
	return rnd
}

// roundRobin is a policy that selects hosts based on round robin ordering.
type roundRobin struct {
	robin uint32
}

func (r *roundRobin) String() string { return "round_robin" }

func (r *roundRobin) List(p []*proxy.Proxy) []*proxy.Proxy {
	poolLen := uint32(len(p))
	i := atomic.AddUint32(&r.robin, 1) % poolLen

	robin := []*proxy.Proxy{p[i]}
	robin = append(robin, p[:i]...)
	robin = append(robin, p[i+1:]...)

	return robin
}

// sequential is a policy that selects hosts based on sequential ordering.
type sequential struct{}

func (r *sequential) String() string { return "sequential" }

func (r *sequential) List(p []*proxy.Proxy) []*proxy.Proxy {
	return p
}

// latency is a load-balancing policy
// that selects a proxy based on the latency of recent requests.
type latency struct {
	// latencyStats stores the Exponentially Weighted Moving Average (EWMA)
	// of request latencies for each proxy, indexed by the proxy's address.
	latencyStats map[string]ewma.MovingAverage
	mux          sync.Mutex
}

func (r *latency) String() string { return "latency" }

// List function sorts the list of proxies based on their EWMA latency.
// Proxies with lower latency (faster response time) are prioritized.
func (r *latency) List(p []*proxy.Proxy) []*proxy.Proxy {
	r.mux.Lock()
	defer r.mux.Unlock()

	proxies := make([]*proxy.Proxy, len(p))
	copy(proxies, p)

	// sort the proxies based on their latency.
	sort.Slice(proxies, func(i, j int) bool {
		currentProxyEWMA, currentProxyExists := r.latencyStats[proxies[i].Addr()]
		nextProxyEWMA, nextProxyExists := r.latencyStats[proxies[j].Addr()]

		// assume it has the maximum latency
		if !currentProxyExists {
			return false
		}
		// assume it has the maximum latency
		if !nextProxyExists {
			return true
		}

		// order them based on their EWMA latency.
		return currentProxyEWMA.Value() < nextProxyEWMA.Value()
	})
	return proxies
}

// OnComplete updates the EWMA latency stats for a proxy once a request is complete.
func (r *latency) OnComplete(proxyAddr string, rtt time.Duration, msg *dns.Msg) {
	r.mux.Lock()
	defer r.mux.Unlock()

	// if we already have latency data for this proxy, retrieve it.
	// if we don't, initialize a new EWMA.
	ewmaVal, ok := r.latencyStats[proxyAddr]
	if !ok {
		ewmaVal = ewma.NewMovingAverage()
		r.latencyStats[proxyAddr] = ewmaVal
	}
	// update the EWMA with the new round-trip time (rtt) measurement.
	ewmaVal.Add(float64(rtt))
}

var rn = rand.New(time.Now().UnixNano())
