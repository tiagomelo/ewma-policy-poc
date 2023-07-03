package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tiagomelo/ewma-policy-poc/config"
	"github.com/tiagomelo/ewma-policy-poc/digger"
	"github.com/tiagomelo/ewma-policy-poc/dnsserver"
	"github.com/tiagomelo/ewma-policy-poc/parser"
	"github.com/tiagomelo/ewma-policy-poc/screen"
	"github.com/tiagomelo/ewma-policy-poc/screen/stats"
	"github.com/tiagomelo/ewma-policy-poc/task"
	"github.com/tiagomelo/ewma-policy-poc/task/worker"
)

// to control worker pool's channel close.
var closeOnce sync.Once

const logFileName = "logs/tester.txt"

type Options struct {
	NumberOfDigs      int `short:"n" long:"number-of-digs" description:"Number of digs to perform" default:"-1"`
	TestTime          int `short:"t" long:"test-time" description:"Duration of test in seconds" default:"-1"`
	RequestsPerSecond int `short:"r" long:"rps" description:"Requests per second" required:"true"`
}

type serverConfig struct {
	name    string
	port    int
	latency int
}

func serverConfigs(cfg *config.Config) []serverConfig {
	return []serverConfig{
		{name: cfg.DnsServer1Name, port: cfg.DnsServer1Port, latency: 1},
		{name: cfg.DnsServer2Name, port: cfg.DnsServer2Port, latency: 1},
		{name: cfg.DnsServer3Name, port: cfg.DnsServer3Port, latency: 1},
	}
}

func metricsHandler() http.Handler {
	return promhttp.Handler()
}

func metricsServer(cfg *config.Config) {
	port := fmt.Sprintf(":%d", cfg.MetricsServerPort)
	http.Handle("/metrics", metricsHandler())
	log.Fatal(http.ListenAndServe(port, nil))
}

func randomServerLatency(logger *log.Logger, cfg *config.Config, servers []*dnsserver.Server) {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	randomAmount := r.Intn(cfg.RslMaxValueInMs-cfg.RslMinValueInMs+1) + cfg.RslMinValueInMs

	ticker := time.NewTicker(time.Duration(cfg.RslPeriodInSeconds) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Randomly select a server.
		index := rand.Intn(len(servers))
		server := servers[index]

		if server.IsRunning() {
			// Assign a random latency.
			newLatency := time.Duration(r.Intn(randomAmount)) * time.Millisecond
			server.SetLatency(newLatency)
			logger.Printf("Set latency of server %s to %s\n", server.GetName(), newLatency)
		}
	}
}

func stopOrStartServer(logger *log.Logger, cfg *config.Config, stats *stats.Statistics, servers []*dnsserver.Server) {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	randomAmount := r.Intn(cfg.SsMaxPeriodInSeconds-cfg.SsMinPeriodInSeconds+1) + cfg.SsMinPeriodInSeconds

	ticker := time.NewTicker(time.Duration(randomAmount) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Randomly select a server.
		index := rand.Intn(len(servers))
		server := servers[index]

		// stop or restart it.
		if server.IsRunning() {
			if err := server.Stop(); err != nil {
				logger.Printf("Error when stopping server %s: %v\n", server.GetName(), err)
			} else {
				logger.Printf("Stopped server %s\n", server.GetName())
				stats.IncrTotalUnavailableServers()
				stats.DecrTotalAvailableServers()
			}
		} else {
			server.Run()
			logger.Printf("Re-started server %s\n", server.GetName())
			stats.IncrTotalAvailableServers()
			stats.DecrTotalUnavailableServers()
		}
	}
}

func parseTemplateFilesForMetrics(cfg *config.Config) error {
	if err := parser.ParseWithLocalIpAddr(cfg.PromTargetServerPort, cfg.PromTemplateFile, cfg.PromOutputFile); err != nil {
		return errors.Wrap(err, "parsing prometheus template file")
	}
	if err := parser.ParseWithLocalIpAddr(cfg.DsServerPort, cfg.DsTemplateFile, cfg.DsOutputFile); err != nil {
		return errors.Wrap(err, "parsing prometheus datasource template file")
	}
	return nil
}

func run(logger *log.Logger, opts Options) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger.Println("main: initializing tests")
	defer logger.Println("main: completed")

	// Reading config.
	cfg, err := config.Read()
	if err != nil {
		return errors.Wrap(err, "reading config")
	}

	// parsing template files for metrics visualization
	// in grafana.
	if err := parseTemplateFilesForMetrics(cfg); err != nil {
		return err
	}

	serverConfigs := serverConfigs(cfg)

	servers := make([]*dnsserver.Server, len(serverConfigs))

	fmt.Println("check execution logs:")
	fmt.Println("tester:", logFileName)

	for i, config := range serverConfigs {
		var err error
		servers[i], err = dnsserver.NewServer(config.name, config.port, config.latency)
		if err != nil {
			return errors.Wrapf(err, `creating server "%s"`, config.name)
		}
		fmt.Printf("server %s: %s\n", servers[i].GetName(), servers[i].GetLogFileName())
		servers[i].Run()
	}

	// wait for all servers to be ready to serve requests.
	fmt.Printf("\nWaiting %d seconds for servers to be up and running...\n", cfg.WaitTimeForServers)
	time.Sleep(time.Duration(cfg.WaitTimeForServers) * time.Second)
	fmt.Println("... ok, let's begin.")

	// statistics to be presented on screen.
	stats := stats.New()
	stats.SetRequestsPerSecond(opts.RequestsPerSecond)
	stats.SetTotalAvailableServers(len(serverConfigs))

	// screen.
	screen, err := screen.New()
	if err != nil {
		return errors.Wrap(err, "initializing screen")
	}

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// randomly update servers latencies.
	go randomServerLatency(logger, cfg, servers)
	// randomly stop/start servers.
	go stopOrStartServer(logger, cfg, stats, servers)

	// Start the metrics server.
	go metricsServer(cfg)

	start := time.Now()

	// displaying stats on screen.
	go func() {
		for {
			time.Sleep(time.Second * time.Duration(1))
			stats.UpdateElapsedTime(time.Since(start))
			screen.UpdateContent(stats, false)
		}
	}()

	// using a worker pool to perform DNS requests.
	maxGoRoutines := runtime.GOMAXPROCS(0)
	pool := task.New(ctx, maxGoRoutines)
	worker := &worker.Worker{
		Domain: cfg.Domain,
		Digger: digger.New(logger, cfg.CorednsHost),
		Logger: logger,
		Stats:  stats,
	}

	var counter int32
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(opts.RequestsPerSecond))
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pool.Do(worker)
				stats.IncrTotalDnsRequests()
				atomic.AddInt32(&counter, 1)

				if opts.NumberOfDigs != -1 && int(atomic.LoadInt32(&counter)) >= opts.NumberOfDigs {
					cancel()
					return
				}
				if opts.TestTime != -1 && time.Since(start) >= time.Duration(opts.TestTime)*time.Second {
					cancel()
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for any error or interrupt signal.
	select {
	case <-shutdown:
		screen.UpdateContent(stats, true)
		closeOnce.Do(func() {
			close(shutdown)
			pool.Shutdown()
		})
	case <-ctx.Done():
		screen.UpdateContent(stats, true)
		closeOnce.Do(func() {
			close(shutdown)
			pool.Shutdown()
		})
	}
	return nil
}

func main() {
	var opts Options
	flags.Parse(&opts)
	if (opts.NumberOfDigs != -1 && opts.TestTime != -1) || (opts.NumberOfDigs == -1 && opts.TestTime == -1) {
		fmt.Println("Error: You must provide either --number-of-digs or --test-time, not both or none.")
		os.Exit(1)
	}
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf(`opening log file "%s": %v`, logFileName, err)
		os.Exit(1)
	}
	logger := log.New(logFile, "TESTER: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	if err := run(logger, opts); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
