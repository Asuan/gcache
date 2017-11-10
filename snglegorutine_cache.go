package gcache

import (
	"sync/atomic"
	"time"
)

type GorCache struct {
	m                 map[string]*Item
	geterFunc         Getter
	stats             Stats
	defaultExpiration int64

	setChan chan namedItem
	getChan chan *getterItem
}

type Getter func(*GorCache, string) []byte

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
		responce: make(chan []byte),
	}
	c.getChan <- getter
	return <-getter.responce
}

func (c *GorCache) Purge() {
	//TODO
}

func (c *GorCache) Dead() {
	//TODO
}

func (c *GorCache) Statistic() Stats {
	return c.stats //Return copy of stats
}

func (c *GorCache) SetOrUpdate(name string, value []byte, expiration int64) {
	myExpiration := expiration
	if myExpiration == 0 {
		myExpiration = c.defaultExpiration
	}
	c.setChan <- namedItem{
		name: name,
		item: Item{
			Object:     value,
			Expiration: myExpiration,
		},
	}
}

func NewGorCache(sizeLimit int64, defaultExpiration time.Duration, isKeepUsefull bool) *GorCache {
	cacheMap := make(map[string]*Item)

	cache := &GorCache{
		m:                 cacheMap,
		defaultExpiration: int64(defaultExpiration),
		stats: Stats{
			SizeLimit: sizeLimit,
		},
	}
	if isKeepUsefull {
		cache.geterFunc = func(c *GorCache, name string) []byte {
			if item, ok := c.m[name]; ok {
				item.Expiration = time.Now().Unix() //reset timer it looks usefull item
				return item.Object
			}
			return []byte{}
		}
	} else {
		cache.geterFunc = func(c *GorCache, name string) []byte {
			if item, ok := c.m[name]; ok {
				return item.Object
			}
			return []byte{}
		}
	}

	worker := func(cache *GorCache) {
		tiker := time.NewTicker(defaultExpiration)
		stats := &cache.stats
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
			}

		}
		tiker.Stop()
	}
	go worker(cache)
	return cache
}
