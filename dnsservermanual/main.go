// This is a simple DNS server implemented in Go.
// No matter what domain name is queried, it responds with
// an 'A' type DNS record that always points 'example.net' to the IP address '1.2.3.4'.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
)

var (
	logger = log.New(os.Stdout, "DNS SERVER: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	cfg    Config
)

type Config struct {
	ServerName string `envconfig:"SERVER_NAME" required:"true"`
	ServerPort int    `envconfig:"SERVER_PORT" required:"true"`
	Latency    int    `envconfig:"LATENCY" required:"true"`
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
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

	logger.Printf(`server "%s" sleeping %d ms before serving the request...`, cfg.ServerName, cfg.Latency)
	time.Sleep(time.Duration(cfg.Latency) * time.Millisecond)
	err := w.WriteMsg(m)
	logger.Printf(`server "%s" served A record "%s"`, cfg.ServerName, recordA)
	if err != nil {
		logger.Printf(`server "%s" failed to write message: %s`, cfg.ServerName, err)
	}
}

// dig @localhost -p <PORT> example.net
func run() error {
	logger.Printf(`main: initializing dns server "%s"`, cfg.ServerName)
	defer logger.Println("main: completed")

	const protocol = "udp"
	port := fmt.Sprintf(":%d", cfg.ServerPort)

	dns.HandleFunc(".", handleDNSRequest)

	server := &dns.Server{
		Addr:    port,
		Net:     protocol,
		Handler: dns.DefaultServeMux,
	}

	logger.Printf(`main: server "%s" listening on port %s`, cfg.ServerName, server.Addr)
	logger.Printf("main: this server has a latency of %d\n", cfg.Latency)

	fmt.Println("************** antes")

	err := server.ListenAndServe()
	defer server.Shutdown()

	fmt.Println("************** depois")

	if err != nil {
		return errors.Wrapf(err, "starting dns server at port %s", server.Addr)
	}

	return nil
}

func main() {
	if err := envconfig.Process("", &cfg); err != nil {
		fmt.Println("error processing env vars:", err)
		os.Exit(1)
	}
	if err := run(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}
