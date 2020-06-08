package mysql

import "sync"

type monitorData struct {
	mu                *sync.RWMutex
	Connections       int64
	MaxConnections    int64
	Concurrences      int64
	MaxConcurrences   int64
	QPS               float64
	TPS               float64
	CacheHitRate      float64
	Input             float64
	Output            float64
	SlowQueriesCount  int64
	blockQueriesCount int64
	DeadLocksCount    int64
	Queries           int64
	BytesReceived     int64
	BytesSent         int64
	TC                int64
	Uptime            int64
	DiskUsage         float64
	DiskTotal         float64
	BlockQueries      []*blockPidRow
	SlowQueries       []*slowQueryRow
	ConnectionsY      []int64
	ConnectionsX      []string
	ConcurrencesY     []int64
	ConcurrencesX     []string
	CacheHitRateY     []float64
	CacheHitRateX     []string
	DiskUsageRateY    []float64
	DiskUsageRateX    []string
	QPSY              []float64
	QPSX              []string
	TPSY              []float64
	TPSX              []string
	InputY            []float64
	InputX            []string
	OutputY           []float64
	OutputX           []string
}

func newMonitorData() *monitorData {

	m := &monitorData{
		mu: new(sync.RWMutex),
	}

	return m
}

func (p *monitorData) append(list *[]interface{}, elem interface{}) {

	if len(*list) >= 60 {
		*list = (*list)[1:59]
	}

	*list = append(*list, elem)
}
