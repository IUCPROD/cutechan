// Package cache provides an in-memory LRU cache for reducing the duplicate
// workload of database requests and post HTML and JSON generation
package cache

import (
	"container/list"
	"sync"
	"time"
)

// Time for the cache to expire and need counter comparison
const expiryTime = time.Second

var (
	cache     = make(map[Key]*list.Element, 10)
	ll        = list.New()
	totalUsed int
	mu        sync.Mutex

	// Size sets the maximum size of cache before evicting unread data in MB
	Size int
)

// Represents some cached object.
type Key struct {
	Lang    string
	Board   string
	ID      uint64
	LastN   int
	Page    int
	Catalog bool
}

// Single cache entry
type store struct {
	// Controls general access to the contents of the struct, except for size
	sync.RWMutex
	key           Key
	updateCounter uint64
	lastChecked   time.Time
	data          interface{}
	html, json    []byte

	// Separate mutex, because accessed both from get requests and cache
	// eviction calls
	size   int
	sizeMu sync.Mutex
}

// Retrieve a store from the cache or create a new one
func getStore(k Key) (s *store) {
	mu.Lock()
	defer mu.Unlock()

	el := cache[k]
	if el == nil {
		s = &store{key: k}
		cache[k] = ll.PushFront(s)
	} else {
		ll.MoveToFront(el)
		s = el.Value.(*store)
	}
	return s
}

// Clear the cache. Only used for testing.
func Clear() {
	mu.Lock()
	defer mu.Unlock()

	ll = list.New()
	cache = make(map[Key]*list.Element, 10)
}

// Update the total used memory counter and evict, if over limit
func updateUsedSize(delta int) {
	mu.Lock()
	defer mu.Unlock()

	totalUsed += delta

	for totalUsed > Size<<20 {
		last := ll.Back()
		if last == nil {
			return
		}
		s := ll.Remove(last).(*store)
		delete(cache, s.key)

		s.sizeMu.Lock()
		totalUsed -= s.size
		s.sizeMu.Unlock()
	}
}

// Return, if the data can still be considered fresh, without querying the DB
func (s *store) isFresh() bool {
	return time.Now().Sub(s.lastChecked) < expiryTime
}

// Stores the new values of s. Calculates and stores the new size. Passes the
// delta to the central cache to fire eviction checks.
func (s *store) update(data interface{}, json, html []byte, f FrontEnd) {
	var newSize int
	if f.Size == nil {
		newSize = computeSize(data, json, html)
	} else {
		newSize = f.Size(data, json, html)
	}

	s.data = data
	s.json = json
	s.html = html

	s.sizeMu.Lock()
	delta := newSize - s.size
	s.size = newSize
	s.sizeMu.Unlock()

	// In a separate goroutine, to ensure there is never any lock intersection
	go updateUsedSize(delta)
}

// Calculating the actual memory footprint of the stored post data is expensive.
// Assume it is as big as the JSON. Most probably it's far less than that.
func computeSize(data interface{}, json, html []byte) int {
	newSize := len(json) + len(html)
	if data != nil {
		newSize += len(json)
	}
	return newSize
}
