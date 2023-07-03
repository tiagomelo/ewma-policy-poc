SHELL = /bin/bash

# ==============================================================================
# Help

.PHONY: help
## help: shows this help message
help:
	@ echo "Usage: make [target]"
	@ sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

# ==============================================================================
# CoreDNS execution with latency policy

.PHONY: coredns-latency-policy
## coredns-latency-policy: runs coredns with latency policy
coredns-latency-policy:
	@ cd coredns ; \
	go run coredns.go -conf ../conf/LatencyCorefile


# ==============================================================================
# CoreDNS execution with round-robin policy

.PHONY: coredns-roundrobin-policy
## coredns-roundrobin-policy: runs coredns with round-robin policy
coredns-roundrobin-policy:
	@ cd coredns ; \
	go run coredns.go -conf ../conf/Corefile

# ==============================================================================
# Tester execution

.PHONY: run-by-time
## run-by-time: runs the tester by a specific time in seconds
run-by-time:
	@ if [ -z "$(TIME)" ]; then echo >&2 please set time in seconds via variable TIME; exit 2; fi
	@ if [ -z "$(RPS)" ]; then echo >&2 please set requests per second via variable RPS; exit 2; fi
	@ go run cmd/main.go -t $(TIME) -r $(RPS)

.PHONY: run-by-digs
## run-by-digs: runs the tester by number of digs
run-by-digs:
	@ if [ -z "$(DIGS)" ]; then echo >&2 please set number of digs via variable DIGS; exit 2; fi
	@ if [ -z "$(RPS)" ]; then echo >&2 please set requests per second via variable RPS; exit 2; fi
	@ go run cmd/main.go -n $(DIGS) -r $(RPS)

# ==============================================================================
# Metrics

.PHONY: obs
## obs: runs both prometheus and grafana
obs:
	@ docker-compose up

.PHONY: obs-stop
## obs-stop: stops both prometheus and grafana
obs-stop:
	@ docker-compose down