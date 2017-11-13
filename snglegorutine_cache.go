package gcache

import (
	"sync/atomic"
	"time"
)

type GorCache struct {
	m                 map[string]*Item
	geterFunc         GetterGorCache
	stats             Stats
	defaultExpiration int64

	setChan   chan namedItem
	getChan   chan *getterItem
	purgeChan chan bool
	deadChan  chan bool
}

type GetterGorCache func(*GorCache, string) []byte

type namedItem struct {
	name string
	item Item
}

type getterItem struct {
	name     string
	responce chan []byte
}

func (c *GorCache) Get(name string) []byte {
	getter := &getterItem{ //TODO make pool for this
		name:     name,
		responce: make(chan []byte, 1),
	}
	c.getChan <- getter
	return <-getter.responce
}

func (c *GorCache) Purge() {
	c.purgeChan <- true
}

func (c *GorCache) Dead() {
	c.deadChan <- true
}

func (c *GorCache) Statistic() Stats {
	return c.stats
}

func (c *GorCache) SetOrUpdate(name string, value []byte, expiration time.Duration) {
	myExpiration := expiration
	var expValue = int64(myExpiration)
	if expiration == DefaultExpirationMarker {
		expValue = time.Now().Add(myExpiration).UnixNano()
	}
	c.setChan <- namedItem{
		name: name,
		item: Item{
			Object:     value,
			Expiration: expValue,
		},
	}
}

func (c *GorCache) purge() {
	for k := range c.m {
		delete(c.m, k)
	}
}

//NewGorCache create new Gorutine cache (no lock but all in one line)
// sizeLimit -- set maximum number of items inside cache
// defaultExpiration -- set expiration for item
// isKeepUsefull -- reset expiration or not for item
func NewGorCache(sizeLimit int64, defaultExpiration time.Duration, isKeepUsefull bool) *GorCache {
	if defaultExpiration <= 0 {
		defaultExpiration = DefaultExpiration
	}
	cache := &GorCache{
		m:                 make(map[string]*Item),
		defaultExpiration: int64(defaultExpiration),
		stats: Stats{
			SizeLimit: sizeLimit,
		},
		setChan:   make(chan namedItem, 100),
		getChan:   make(chan *getterItem, 100),
		purgeChan: make(chan bool),
		deadChan:  make(chan bool),
	}
	if isKeepUsefull {
		cache.geterFunc = func(c *GorCache, name string) []byte {
			if item, ok := c.m[name]; ok {
				item.Expiration = time.Now().Unix() //reset timer it looks usefull item
				return item.Object
			}
			return nil
		}
	} else {
		cache.geterFunc = func(c *GorCache, name string) []byte {
			if item, ok := c.m[name]; ok {
				return item.Object
			}
			return nil
		}
	}

	// working with cache in gorutinge without any lock
	worker := func(cache *GorCache) {
		tiker := time.NewTicker(defaultExpiration)
		stats := &cache.stats
	loop:
		for {
			select {
			case itm := <-cache.setChan:
				if stats.ItemsCount < stats.SizeLimit {
					cache.m[itm.name] = &itm.item
					atomic.AddInt64(&stats.SetOrReplaceCount, 1)
					stats.ItemsCount = int64(len(cache.m))
				}
			case get := <-cache.getChan:
				result := cache.geterFunc(cache, get.name)
				if len(result) != 0 {
					atomic.AddInt64(&stats.GetSuccessNumber, 1)
				} else {
					atomic.AddInt64(&stats.GetErrorNumber, 1)
				}

				get.responce <- result
			case <-tiker.C:
				now := time.Now().Unix()
				for k, v := range cache.m {
					if v.Expiration < now && v.Expiration > 0 {
						delete(cache.m, k)
						atomic.AddInt64(&stats.DeleteExpired, 1)
						atomic.AddInt64(&stats.ItemsCount, -1)
					}
				}
			case <-cache.purgeChan:
				atomic.AddInt64(&stats.DeleteCount, int64(len(cache.m)))
				atomic.StoreInt64(&stats.ItemsCount, int64(0))
				cache.purge()

			case <-cache.deadChan:
				atomic.StoreInt64(&stats.ItemsCount, int64(0))
				cache.purge()
				break loop
			}
		}
		tiker.Stop()
	}

	go worker(cache)
	return cache
}
