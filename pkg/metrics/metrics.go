package metrics

import (
	"fmt"
	"net/url"
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

func (metrics *Metrics) addTimeSaved(
	reqURL string,
	timeSaved time.Duration,
) (err error) {
	url, err := url.Parse(reqURL)
	if err != nil {
		return err
	}

	hostname := url.Hostname()
	hostnameTimeSavedInterface, _ := metrics.timeSaved.LoadOrStore(
		hostname,
		&sync.Map{},
	)
	hostnameTimeSaved := hostnameTimeSavedInterface.(*sync.Map)
	fullPathTimeSavedInterface, _ := hostnameTimeSaved.LoadOrStore(
		reqURL,
		time.Duration(0),
	)
	fullPathTimeSaved := fullPathTimeSavedInterface.(time.Duration)
	hostnameTimeSaved.Store(reqURL, fullPathTimeSaved+timeSaved)

	return nil
}

func (metrics *Metrics) addBandwidthSaved(
	reqURL string,
	bandwidthSaved int64,
) (err error) {
	url, err := url.Parse(reqURL)
	if err != nil {
		return err
	}
	hostname := url.Hostname()
	hostnameBandwitdhSavedInterface, _ := metrics.bandwidthSaved.LoadOrStore(
		hostname,
		&sync.Map{},
	)
	hostnameBandwidthSaved := hostnameBandwitdhSavedInterface.(*sync.Map)
	fullPathBandwidthSavedInterface, _ := hostnameBandwidthSaved.LoadOrStore(
		reqURL,
		int64(0),
	)
	fullPathBandwidthSaved := fullPathBandwidthSavedInterface.(int64)
	hostnameBandwidthSaved.Store(reqURL, fullPathBandwidthSaved+bandwidthSaved)

	return nil
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
	metrics.timeSaved.Range(func(hostname, fullPathTimeSavedInterface interface{}) bool {
		fullPathTimeSaved := fullPathTimeSavedInterface.(*sync.Map)
		totalTimeSaved := time.Duration(0)
		fullPathTimeSaved.Range(func(fullPathName, timeSavedInterface interface{}) bool {
			timeSaved := timeSavedInterface.(time.Duration)
			totalTimeSaved += timeSaved
			return true
		})
		fmt.Fprintf(&builder, "%s - %q: %s\n", prefix, hostname, totalTimeSaved)
		prefix = "        "
		fullPathTimeSaved.Range(func(fullPathName, timeSaved interface{}) bool {
			fmt.Fprintf(&builder, "%s - %q: %s\n", prefix, fullPathName, timeSaved)
			return true
		})
		prefix = "   "
		return true
	})
	fmt.Fprintf(&builder, "%sbandwidth saved:\n", prefix)
	metrics.bandwidthSaved.Range(func(hostname, fullPathBandwidthSavedInterface interface{}) bool {
		fullPathBandwidthSaved := fullPathBandwidthSavedInterface.(*sync.Map)
		totalBandwidthSaved := int64(0)
		fullPathBandwidthSaved.Range(func(fullPathName, bandwidthSavedInterface interface{}) bool {
			bandwidthSaved := bandwidthSavedInterface.(int64)
			totalBandwidthSaved += bandwidthSaved
			return true
		})
		fmt.Fprintf(&builder, "%s - %q: %v bytes\n", prefix, hostname, totalBandwidthSaved)
		prefix = "        "
		fullPathBandwidthSaved.Range(func(fullPathName, bandwidthSaved interface{}) bool {
			fmt.Fprintf(&builder, "%s - %q: %v bytes\n", prefix, fullPathName, bandwidthSaved)
			return true
		})
		prefix = "   "
		return true
	})

	return strings.TrimRight(builder.String(), "\n")
}
