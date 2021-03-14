package metrics

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lexesjan/go-web-proxy-server/pkg/cache"
)

// Metrics represents the metrics stored for the proxy
type Metrics struct {
	timeSaved      *sync.Map
	bandwidthSaved *sync.Map
}

// NewMetrics returns a new Metrics struct
func NewMetrics() (metrics *Metrics) {
	metrics = &Metrics{
		timeSaved:      &sync.Map{},
		bandwidthSaved: &sync.Map{},
	}

	return metrics
}

func (metrics *Metrics) addTimeSaved(reqURL string, timeSaved time.Duration) {
	currentTimeSavedInterface, _ := metrics.timeSaved.LoadOrStore(
		reqURL,
		time.Duration(0),
	)
	currentTimeSaved := currentTimeSavedInterface.(time.Duration)
	metrics.timeSaved.Store(reqURL, currentTimeSaved+timeSaved)
}

func (metrics *Metrics) addBandwidthSaved(
	reqURL string,
	bandwidthSaved int64,
) {
	currentBandwidthSavedInterface, _ := metrics.bandwidthSaved.LoadOrStore(
		reqURL,
		int64(0),
	)
	currentBandwidthSaved := currentBandwidthSavedInterface.(int64)
	metrics.bandwidthSaved.Store(reqURL, currentBandwidthSaved+bandwidthSaved)
}

// AddMetrics calculates the time and bandwidth saved and adds them to the total
// time and bandwidth saved
func (metrics *Metrics) AddMetrics(
	reqURL string,
	cacheEntry *cache.Entry,
	timeTaken time.Duration,
	bandwidthUsed int64,
) {
	timeSaved := cacheEntry.UncachedResponseTime - timeTaken
	bandwidthSaved := cacheEntry.UncachedBandwidth - bandwidthUsed
	metrics.addTimeSaved(reqURL, timeSaved)
	metrics.addBandwidthSaved(reqURL, bandwidthSaved)
}

func (metrics *Metrics) String() string {
	var builder strings.Builder

	prefix := ""
	fmt.Fprintf(&builder, "%smetrics:\n", prefix)
	prefix = "   "
	fmt.Fprintf(&builder, "%stime saved:\n", prefix)
	metrics.timeSaved.Range(func(key, value interface{}) bool {
		fmt.Fprintf(&builder, "%s - %q: %s\n", prefix, key, value)
		return true
	})
	fmt.Fprintf(&builder, "%sbandwidth saved:\n", prefix)
	metrics.bandwidthSaved.Range(func(key, value interface{}) bool {
		fmt.Fprintf(&builder, "%s - %q: %v bytes\n", prefix, key, value)
		return true
	})

	return strings.TrimRight(builder.String(), "\n")
}
