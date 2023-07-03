package dnsserver

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	dnsRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dns_requests_total",
			Help: "Number of DNS requests.",
		},
		[]string{"server"},
	)
	serverStops = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "server_stop_total",
			Help: "Number of server stops.",
		},
		[]string{"server"},
	)
	serverStarts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "server_start_total",
			Help: "Number of server starts.",
		},
		[]string{"server"},
	)
	dnsRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dns_request_duration_seconds",
		Help:    "Time taken for dns request",
		Buckets: prometheus.DefBuckets,
	}, []string{"server"})
)

func init() {
	prometheus.MustRegister(dnsRequests)
	prometheus.MustRegister(serverStops)
	prometheus.MustRegister(serverStarts)
	prometheus.MustRegister(dnsRequestDuration)
}

type Server struct {
	name        string
	port        int
	latency     time.Duration
	dnsSrv      *dns.Server
	logFileName string
	logger      *log.Logger

	mux       sync.Mutex
	isRunning bool
}

func NewServer(name string, port int, latency int) (*Server, error) {
	logFileName := fmt.Sprintf("logs/dnsserver_%s.txt", name)
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, errors.Wrapf(err, `opening log file "%s"`, logFileName)
	}
	logger := log.New(logFile, "TESTER: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	return &Server{
		name:        name,
		port:        port,
		latency:     time.Duration(latency) * time.Millisecond,
		logger:      logger,
		logFileName: logFileName,
	}, nil
}

func (s *Server) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	start := time.Now()
	const recordA = "example.net. 3600 IN A 1.2.3.4"
	m := new(dns.Msg)
	m.SetReply(r)

	switch r.Question[0].Qtype {
	case dns.TypeA:
		rr, err := dns.NewRR(recordA)
		if err == nil {
			m.Answer = append(m.Answer, rr)
		}
	}

	s.logger.Printf(`server "%s" sleeping %s before serving the request...`, s.name, s.latency)
	time.Sleep(s.latency)
	err := w.WriteMsg(m)
	if err != nil {
		s.logger.Printf(`server "%s" failed to write message: %s`, s.name, err)
	}
	s.logger.Printf(`server "%s" served A record "%s"`, s.name, recordA)
	dnsRequests.With(prometheus.Labels{"server": s.name}).Inc()
	dnsRequestDuration.With(prometheus.Labels{"server": s.name}).Observe(time.Since(start).Seconds())

}

func (s *Server) Run() {
	dnsSrv := &dns.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Net:     "udp",
		Handler: dns.HandlerFunc(s.handleDNSRequest),
	}
	s.dnsSrv = dnsSrv
	go func() {
		if err := dnsSrv.ListenAndServe(); err != nil {
			s.logger.Fatalf("Failed to start server: %s", err.Error())
		}
	}()

	s.logger.Printf(`main: server "%s" listening on port %d`, s.name, s.port)
	s.logger.Printf("main: this server has a latency of %s\n", s.latency)
	s.mux.Lock()
	defer s.mux.Unlock()
	s.isRunning = true
	serverStarts.With(prometheus.Labels{"server": s.name}).Inc()
}

func (s *Server) Stop() error {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.isRunning = false
	serverStops.With(prometheus.Labels{"server": s.name}).Inc()
	return s.dnsSrv.Shutdown()
}

func (s *Server) GetName() string {
	return s.name
}

func (s *Server) GetLogFileName() string {
	return s.logFileName
}

func (s *Server) IsRunning() bool {
	s.mux.Lock()
	defer s.mux.Unlock()
	return s.isRunning
}

func (s *Server) SetLatency(latency time.Duration) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.latency = latency
}
