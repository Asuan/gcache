package gcache

import (
	"sync/atomic"
	"time"
)

//GorCache is a lock less single gorutine cache (use chan for comunication)
//It is slow but so fun with async communications
type GorCache struct {
	m                 map[string]*Item
	geterFunc         GetterGorCache
	stats             Stats
	defaultExpiration int64

	setChan   chan namedItem
	getChan   chan *getterItem
	purgeChan chan bool
	deadChan  chan bool
	statsChan chan *statItem
}

//GetterGorCache is a func for different get functionality depend on IsKeepUsefull option.
type GetterGorCache func(*GorCache, string) []byte

type namedItem struct {
	name string
	item *Item
}

type getterItem struct {
	name     string
	responce chan []byte
}

type statItem struct {
	responce chan Stats
}

//Get func is implementation of getting value of cache
func (c *GorCache) Get(name string) []byte {
	getter := &getterItem{ //TODO make pool for this
		name:     name,
		responce: make(chan []byte, 1),
	}
	c.getChan <- getter
	return <-getter.responce
}

//Purge func cleanup all items in cache
func (c *GorCache) Purge() {
	c.purgeChan <- true
}

//Dead call deleting of all data na grace stopping cache
func (c *GorCache) Dead() {
	c.deadChan <- true
}

//Statistic return statatistic of current cache
func (c *GorCache) Statistic() Stats {
	getter := &statItem{ //TODO make pool for this
		responce: make(chan Stats, 1),
	}
	c.statsChan <- getter
	return <-getter.responce
}

//SetOrUpdate set or update cache item
func (c *GorCache) SetOrUpdate(name string, value []byte, expiration time.Duration) {
	myExpiration := expiration
	var expValue = int64(myExpiration)
	if expiration == DefaultExpirationMarker {
		expValue = time.Now().Add(myExpiration).UnixNano()
	}
	c.setChan <- namedItem{
		name: name,
		item: &Item{
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
func NewGorCache(config ConfigCacheInterface) *GorCache {
	defaultExpiration := config.GetDefaultExpiration()
	if defaultExpiration <= 0 {
		defaultExpiration = int64(DefaultExpiration)
	}
	cache := &GorCache{
		m:                 make(map[string]*Item),
		defaultExpiration: int64(defaultExpiration),
		stats: Stats{
			SizeLimit: config.GetSizeLimit(),
		},
		setChan:   make(chan namedItem, 100),
		getChan:   make(chan *getterItem, 100),
		purgeChan: make(chan bool),
		deadChan:  make(chan bool),
		statsChan: make(chan *statItem),
	}
	if config.GetIsKeepUsefull() {
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
		tiker := time.NewTicker(time.Duration(defaultExpiration))
		stats := &cache.stats
	loop:
		for {
			select {
			case itm := <-cache.setChan:
				if stats.ItemsCount < stats.SizeLimit {
					cache.m[itm.name] = itm.item
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
			case sts := <-cache.statsChan:
				sts.responce <- cache.stats
			}
		}
		tiker.Stop()
	}

	go worker(cache)
	return cache
}
