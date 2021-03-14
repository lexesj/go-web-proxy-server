package cache

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lexesjan/go-web-proxy-server/pkg/http"
	"github.com/lexesjan/go-web-proxy-server/pkg/log"
)

// Cache represents the proxy cache
type Cache struct {
	cacheMap *sync.Map
}

// NewCache returns a new Cache
func NewCache() (cache *Cache) {
	cache = &Cache{cacheMap: &sync.Map{}}

	return cache
}

// Entry represents a cache entry
type Entry struct {
	Response             string
	Stale                bool
	UncachedResponseTime time.Duration
	UncachedBandwidth    int64
}

func (cache *Cache) CacheResponse(
	reqURL string,
	resp *http.Response,
	duration time.Duration,
) (err error) {
	contains := func(arr []string, str string) bool {
		for _, elem := range arr {
			if elem == str {
				return true
			}
		}
		return false
	}

	newCacheEntry := &Entry{
		Response:             resp.String(),
		Stale:                false,
		UncachedResponseTime: duration,
		UncachedBandwidth:    int64(len(resp.String())),
	}

	cacheControl := resp.Headers.CacheControl()
	uncacheable := contains(cacheControl, "no-store") || resp.StatusCode == 304
	// Can't be cached.
	if uncacheable {
		return nil
	}

	cache.cacheMap.Store(reqURL, newCacheEntry)
	err = newCacheEntry.ResetTimer(reqURL, cacheControl)
	if err != nil {
		return err
	}

	return nil
}

// ResetTimer resets the timer which marks a cache entry stale
func (entry *Entry) ResetTimer(
	reqURL string,
	cacheControl []string,
) (err error) {
	maxAge := 0
	for _, elem := range cacheControl {
		if strings.HasPrefix(elem, "max-age") {
			tokens := strings.Split(elem, "=")
			maxAge, err = strconv.Atoi(tokens[1])
			if err != nil {
				return err
			}
		}
	}

	// Mark expired cache as stale
	time.AfterFunc(time.Duration(maxAge)*time.Second, func() {
		entry.Stale = true
		log.ProxyCacheStale(reqURL)
	})

	return nil
}

// Get returns the cache Entry in the map. The ok result indicates whether the
// value was found in the map
func (cache *Cache) Get(key string) (value *Entry, ok bool) {
	cachedEntryInterface, ok := cache.cacheMap.Load(key)
	value = &Entry{}
	if ok {
		value = cachedEntryInterface.(*Entry)
	}

	return value, ok
}
