package gcache

import (
	"sync"
	"time"
)

type Rwlockcache struct {
	defaultExpiration time.Duration
	l                 sync.RWMutex
	m                 map[string]*Item
	janitor           *janitor
	stats             Stats
}

// Delete all expired items from the cache.
func (c *Rwlockcache) deleteExpired() {
	//now := time.Now().UnixNano()

}

// Delete all items from the cache.
func (c *Rwlockcache) Purge() {
	c.l.Lock()
	for k := range c.m {
		delete(c.m, k)
	}
	c.stats.ItemsCount = 0
	c.l.Lock()
}

func (c *Rwlockcache) Statistic() Stats {
	return c.stats
}
