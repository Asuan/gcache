package cache

import (
	"fmt"

	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// For use with functions that take an expiration time.
	NoExpiration time.Duration = -1
	// For use with functions that take an expiration time. Equivalent to
	// passing in the same expiration duration as was given to New() or
	// NewFrom() when the cache was created (e.g. 5 minutes.)
	DefaultExpiration time.Duration = 0
)

type lockMap struct {
	l sync.RWMutex
	m map[string]Item
}

type Item struct {
	Object     []byte
	Size       int32
	Expiration int64
}

type Cache struct {
	*cache
	// If this is confusing, see the comment at the bottom of New()
}

type cache struct {
	calcHash          HashCalculator
	defaultExpiration time.Duration
	shards            []*lockMap
	shardCount        uint64
	janitor           *janitor
	Statistic         stats
}

type stats struct {
	ItemsCount, GetCount, ErrorGetCount, SetCount, ReplaceCount, DeleteCount, AddCount, DeleteExpired, Size int32
}

func (c *cache) newShardMap() {
	count := uint64(10)

	c.shards = make([]*lockMap, count)
	c.shardCount = count

	for i := range c.shards {
		c.shards[i] = &lockMap{m: make(map[string]Item)}
	}
}

// Returns true if the item has expired.
func (item Item) expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

func (c *cache) GetShard(str string) *lockMap {
	key := c.calcHash(str)
	return c.shards[key%c.shardCount]
}

// Add an item to the cache, replacing any existing item. If the duration is 0
// (DefaultExpiration), the cache's default expiration time is used. If it is -1
// (NoExpiration), the item never expires.
func (c *cache) Set(k string, x []byte, d time.Duration) {
	atomic.AddInt32(&c.Statistic.SetCount, 1)
	atomic.AddInt32(&c.Statistic.ItemsCount, 1)
	atomic.AddInt32(&c.Statistic.Size, int32(len(x)))
	c.set(k, x, d)
}

func (c *cache) set(k string, x []byte, d time.Duration) {
	// "Inlining" of set
	var e int64
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}

	shard := c.GetShard(k)
	size := int32(len(x))
	shard.l.Lock()
	shard.m[k] = Item{
		Object:     x,
		Size:       size,
		Expiration: e,
	}
	shard.l.Unlock()
}

// Add an item to the cache only if an item doesn't already exist for the given
// key, or if the existing item has expired. Returns an error otherwise.
func (c *cache) Add(k string, x []byte, d time.Duration) error {
	_, found := c.Get(k)
	if found {
		return fmt.Errorf("Item %s already exists", k)
	}
	atomic.AddInt32(&c.Statistic.AddCount, 1)
	atomic.AddInt32(&c.Statistic.ItemsCount, 1)
	atomic.AddInt32(&c.Statistic.Size, int32(len(x)))
	c.set(k, x, d)
	return nil
}

// Set a new value for the cache key only if it already exists, and the existing
// item hasn't expired. Returns an error otherwise.
func (c *cache) Replace(k string, x []byte, d time.Duration) error {
	o, found := c.Get(k)
	if !found {
		return fmt.Errorf("Item %s doesn't exist", k)
	}
	size := int32(len(o.([]byte)))
	atomic.AddInt32(&c.Statistic.ReplaceCount, 1)
	atomic.AddInt32(&c.Statistic.Size, int32(len(x))-size)
	c.set(k, x, d)
	return nil
}

// Get an item from the cache. Returns the item or nil, and a bool indicating
// whether the key was found.
func (c *cache) Get(k string) (interface{}, bool) {
	// "Inlining" of get and expired
	shard := c.GetShard(k)
	shard.l.RLock()
	item, found := shard.m[k]
	shard.l.RUnlock()

	if !found {
		atomic.AddInt32(&c.Statistic.ErrorGetCount, 1)
		return nil, false
	}
	atomic.AddInt32(&c.Statistic.GetCount, 1)
	return item.Object, true
}

func (c *cache) Delete(k string) (interface{}, bool) {
	shard := c.GetShard(k)
	v, f := shard.m[k]

	if f {
		atomic.AddInt32(&c.Statistic.ItemsCount, -1)
		atomic.AddInt32(&c.Statistic.Size, -v.Size)
		atomic.AddInt32(&c.Statistic.DeleteCount, 1)
		delete(shard.m, k)
		return v.Object, true
	}
	return nil, false
}

// Delete all expired items from the cache.
func (c *cache) DeleteExpired() {
	now := time.Now().UnixNano()
	for i := range c.shards {
		sh := c.shards[i]
		sh.l.Lock()
		for k, v := range sh.m {
			if v.Expiration > 0 && now > v.Expiration {
				atomic.AddInt32(&c.Statistic.DeleteExpired, 1)
				atomic.AddInt32(&c.Statistic.ItemsCount, -1)
				atomic.AddInt32(&c.Statistic.ItemsCount, -v.Size)
				delete(sh.m, k)
			}
		}
		sh.l.Unlock()
	}

}

// Returns the number of items in the cache. This may include items that have
// expired, but have not yet been cleaned up. Equivalent to len(c.Items()).
func (c *cache) ItemCount() int32 {
	return c.Statistic.ItemsCount
}

// Delete all items from the cache.
func (c *cache) Flush() {
	c.newShardMap() //TODO init with params
	c.Statistic.ItemsCount = 0
	c.Statistic.Size = 0
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *janitor) Run(c *cache) {
	j.stop = make(chan bool)
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(c *Cache) {
	c.janitor.stop <- true
}

func runJanitor(c *cache, ci time.Duration) {
	j := &janitor{
		Interval: ci,
	}
	c.janitor = j
	go j.Run(c)
}

func newCache(de time.Duration) *cache {
	if de == 0 {
		de = -1
	}
	c := &cache{
		defaultExpiration: de,
	}
	c.calcHash = calcSUM
	c.newShardMap()
	return c
}

func newCacheWithJanitor(de time.Duration, ci time.Duration) *Cache {
	c := newCache(de)
	// This trick ensures that the janitor goroutine (which--granted it
	// was enabled--is running DeleteExpired on c forever) does not keep
	// the returned C object from being garbage collected. When it is
	// garbage collected, the finalizer stops the janitor goroutine, after
	// which c can be collected.
	C := &Cache{c}

	if ci > 0 {
		runJanitor(c, ci)
		runtime.SetFinalizer(C, stopJanitor)
	}
	return C
}

// Return a new cache with a given default expiration duration and cleanup
// interval. If the expiration duration is less than one (or NoExpiration),
// the items in the cache never expire (by default), and must be deleted
// manually. If the cleanup interval is less than one, expired items are not
// deleted from the cache before calling c.DeleteExpired().
func New(defaultExpiration, cleanupInterval time.Duration) *Cache {
	return newCacheWithJanitor(defaultExpiration, cleanupInterval)
}
