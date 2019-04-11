package metrics

import (
	"fmt"
	"io"
	"sync/atomic"
)

// NewCounter registers and returns new counter with the given name.
//
// name must be valid Prometheus-compatible metric with possible lables.
// For instance,
//
//     * foo
//     * foo{bar="baz"}
//     * foo{bar="baz",aaa="b"}
//
// The returned counter is safe to use from concurrent goroutines.
func NewCounter(name string) *Counter {
	c := &Counter{}
	registerMetric(name, c)
	return c
}

// Counter is a counter.
//
// It may be used as a gauge if Dec and Set are called.
type Counter struct {
	n uint64
}

// Inc increments c.
func (c *Counter) Inc() {
	atomic.AddUint64(&c.n, 1)
}

// Dec decrements c.
func (c *Counter) Dec() {
	atomic.AddUint64(&c.n, ^uint64(0))
}

// Add adds n to c.
func (c *Counter) Add(n int) {
	atomic.AddUint64(&c.n, uint64(n))
}

// Get returns the current value for c.
func (c *Counter) Get() uint64 {
	return atomic.LoadUint64(&c.n)
}

// Set sets c value to n.
func (c *Counter) Set(n uint64) {
	atomic.StoreUint64(&c.n, n)
}

// marshalTo marshals c with the given prefix to w.
func (c *Counter) marshalTo(prefix string, w io.Writer) {
	v := c.Get()
	fmt.Fprintf(w, "%s %d\n", prefix, v)
}

// GetOrCreateCounter returns registered counter with the given name
// or creates new counter if the registry doesn't contain counter with
// the given name.
//
// name must be valid Prometheus-compatible metric with possible lables.
// For instance,
//
//     * foo
//     * foo{bar="baz"}
//     * foo{bar="baz",aaa="b"}
//
// The returned counter is safe to use from concurrent goroutines.
//
// Performance tip: prefer NewCounter instead of GetOrCreateCounter.
func GetOrCreateCounter(name string) *Counter {
	metricsMapLock.Lock()
	nm := metricsMap[name]
	metricsMapLock.Unlock()
	if nm == nil {
		// Slow path - create and register missing counter.
		if err := validateMetric(name); err != nil {
			panic(fmt.Errorf("BUG: invalid metric name %q: %s", name, err))
		}
		nmNew := &namedMetric{
			name:   name,
			metric: &Counter{},
		}
		metricsMapLock.Lock()
		nm = metricsMap[name]
		if nm == nil {
			nm = nmNew
			metricsMap[name] = nm
			metricsList = append(metricsList, nm)
		}
		metricsMapLock.Unlock()
	}

	c, ok := nm.metric.(*Counter)
	if !ok {
		panic(fmt.Errorf("BUG: metric %q isn't a Counter. It is %T", name, nm.metric))
	}
	return c
}
