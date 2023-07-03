package stats

import (
	"sync/atomic"
	"time"
)

// statistics contains information to be displayed at screen.
// its properties can't be acessed directly on purpose;
// since they'll be updated concurrently,
// we want to make it thread-safe.
type Statistics struct {
	requestsPerSecond       int
	totalDnsRequests        int64
	totalFailedDnsRequests  int64
	totalAvailableServers   int32
	totalUnavailableServers int32
	elapsedTime             time.Duration
}

// NewStatistics creates a new Statistics
func New() *Statistics {
	return &Statistics{}
}

func (s *Statistics) SetRequestsPerSecond(requestsPerSecond int) {
	s.requestsPerSecond = requestsPerSecond
}

func (s *Statistics) RequestsPerSecond() int {
	return s.requestsPerSecond
}

func (s *Statistics) IncrTotalDnsRequests() {
	atomic.AddInt64(&s.totalDnsRequests, 1)
}

func (s *Statistics) TotalDnsRequests() int64 {
	return s.totalDnsRequests
}

func (s *Statistics) IncrTotalFailedDnsRequests() {
	atomic.AddInt64(&s.totalFailedDnsRequests, 1)
}

func (s *Statistics) TotalFailedDnsRequests() int64 {
	return s.totalFailedDnsRequests
}

func (s *Statistics) SetTotalAvailableServers(total int) {
	s.totalAvailableServers = int32(total)
}

func (s *Statistics) IncrTotalAvailableServers() {
	atomic.AddInt32(&s.totalAvailableServers, 1)
}

func (s *Statistics) DecrTotalAvailableServers() {
	atomic.AddInt32(&s.totalAvailableServers, -1)
}

func (s *Statistics) TotalAvailableServers() int {
	return int(s.totalAvailableServers)
}

func (s *Statistics) IncrTotalUnavailableServers() {
	atomic.AddInt32(&s.totalUnavailableServers, 1)
}

func (s *Statistics) DecrTotalUnavailableServers() {
	atomic.AddInt32(&s.totalUnavailableServers, -1)
}

func (s *Statistics) TotalUnavailableServers() int {
	return int(s.totalUnavailableServers)
}

func (s *Statistics) UpdateElapsedTime(elapsedTime time.Duration) {
	s.elapsedTime = elapsedTime
}

func (s *Statistics) ElapsedTime() time.Duration {
	return s.elapsedTime
}
