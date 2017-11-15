package gcache

import (
	"sync"
	"sync/atomic"
	"time"
)

//Rwlockcache classic cache based on map + ReadWrite lock
type Rwlockcache struct {
	defaultExpiration int64
	l                 sync.RWMutex
	m                 map[string]*Item
	janitor           *janitor
	stats             Stats
	getterFunc        rwGetter
}

type rwGetter func(*Rwlockcache, string) []byte

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

//NewRwCache create new Rwlockcache based on cache config
func NewRwCache(config ConfigCacheInterface) *Rwlockcache {
	defaultExpiration := config.GetDefaultExpiration()
	if defaultExpiration <= 0 {
		defaultExpiration = int64(DefaultExpiration)
	}
	cache := &Rwlockcache{
		defaultExpiration: defaultExpiration,
		m:                 make(map[string]*Item),
		janitor: &janitor{
			Interval: time.Duration(defaultExpiration * 5),
			stop:     make(chan bool),
		},
		stats: Stats{
			SizeLimit: config.GetSizeLimit(),
		},
	}
	if config.GetIsKeepUsefull() {
		cache.getterFunc = func(cache *Rwlockcache, name string) []byte {
			cache.l.Lock()
			if itm, ok := cache.m[name]; ok {
				atomic.AddInt64(&cache.stats.GetSuccessNumber, 1)
				itm.Expiration = int64(defaultExpiration) + itm.Expiration
				v := itm.Object
				cache.l.Unlock()
				return v
			}
			atomic.AddInt64(&cache.stats.GetErrorNumber, 1)
			cache.l.Unlock()
			return nil
		}
	} else {
		cache.getterFunc = func(cache *Rwlockcache, name string) []byte {
			cache.l.RLock()
			if itm, ok := cache.m[name]; ok {
				v := itm.Object
				cache.l.RUnlock()
				return v
			}
			cache.l.RUnlock()
			return nil
		}

	}

	go cache.janitor.Run(cache)

	return cache
}

// Delete all expired items from the cache.
func (c *Rwlockcache) deleteExpired() {
	now := time.Now().UnixNano()
	c.l.Lock()
	for k, v := range c.m {
		if v.expired(now) {
			delete(c.m, k)
		}
	}
	c.l.Unlock()
}

//Get return item by name or nil
func (c *Rwlockcache) Get(name string) []byte {
	return c.getterFunc(c, name)
}

//SetOrUpdate set or update item in cache
func (c *Rwlockcache) SetOrUpdate(name string, value []byte, exp time.Duration) {
	atomic.AddInt64(&c.stats.SetOrReplaceCount, 1)
	c.l.Lock()
	itm := &Item{
		Expiration: int64(exp),
		Object:     value,
	}
	c.m[name] = itm
	c.l.Unlock()
}

//Purge delete all items from the cache
func (c *Rwlockcache) Purge() {
	c.l.Lock()
	atomic.AddInt64(&c.stats.DeleteCount, int64(len(c.m)))
	for k := range c.m {
		delete(c.m, k)
	}
	c.l.Unlock()
}

//Dead stop all internal func and clear cache
func (c *Rwlockcache) Dead() {
	c.janitor.stop <- true
	c.Purge()
}

//Statistic return all cache statistic
func (c *Rwlockcache) Statistic() Stats {
	c.l.RLock()
	c.stats.ItemsCount = int64(len(c.m))
	c.l.RUnlock()
	return c.stats
}

//Run async janitor
func (j *janitor) Run(c *Rwlockcache) {
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}
