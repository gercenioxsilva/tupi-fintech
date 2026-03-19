package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type counter struct {
	mu     sync.Mutex
	values map[string]float64
}

func newCounter() *counter          { return &counter{values: map[string]float64{}} }
func (c *counter) Inc(label string) { c.Add(label, 1) }
func (c *counter) Add(label string, value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[label] += value
}
func (c *counter) snapshot() map[string]float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]float64, len(c.values))
	for k, v := range c.values {
		out[k] = v
	}
	return out
}

type histogram struct {
	mu     sync.Mutex
	sums   map[string]float64
	counts map[string]uint64
}

func newHistogram() *histogram {
	return &histogram{sums: map[string]float64{}, counts: map[string]uint64{}}
}
func (h *histogram) Observe(label string, d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sums[label] += d.Seconds()
	h.counts[label]++
}

type Metrics struct {
	TransactionsTotal    *counter
	TransactionDuration  *histogram
	AuthorizationResults *counter
}

func NewMetrics() *Metrics {
	return &Metrics{TransactionsTotal: newCounter(), TransactionDuration: newHistogram(), AuthorizationResults: newCounter()}
}

func (m *Metrics) RenderPrometheus() string {
	var b strings.Builder
	writeCounter := func(name, help, label string, values map[string]float64) {
		b.WriteString(fmt.Sprintf("# HELP %s %s\n", name, help))
		b.WriteString(fmt.Sprintf("# TYPE %s counter\n", name))
		keys := make([]string, 0, len(values))
		for k := range values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("%s{%s=%q} %v\n", name, label, k, values[k]))
		}
	}
	writeCounter("emv_transactions_total", "Total de transações EMV processadas.", "status", m.TransactionsTotal.snapshot())
	writeCounter("emv_authorization_total", "Resultado das autorizações mock.", "decision", m.AuthorizationResults.snapshot())
	b.WriteString("# HELP emv_transaction_duration_seconds Latência acumulada e contagem das transações EMV.\n")
	b.WriteString("# TYPE emv_transaction_duration_seconds summary\n")
	m.TransactionDuration.mu.Lock()
	keys := make([]string, 0, len(m.TransactionDuration.sums))
	for k := range m.TransactionDuration.sums {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("emv_transaction_duration_seconds_sum{status=%q} %v\n", k, m.TransactionDuration.sums[k]))
		b.WriteString(fmt.Sprintf("emv_transaction_duration_seconds_count{status=%q} %d\n", k, m.TransactionDuration.counts[k]))
	}
	m.TransactionDuration.mu.Unlock()
	return b.String()
}
