package gcache

import (
	"sync/atomic"
	"time"
)

//ShardCache implamantation of distributed cache, shards of the current can be any default cache
// with implemented interface Cacher
type ShardCache struct {
	Cacher
	shards     []Cacher
	shardCount uint64
	hashCalc   HashCalculator
}

//ShardGenerator functino for generating cache
type ShardGenerator func(ConfigCacheInterface) Cacher

//NewShardCache create new shard cache with specified paramters
// generator is a func for creating sahrd
//hashCalc is a func for distributing items
func NewShardCache(config ConfigShardCacheInterface, generator ShardGenerator, hashCalc HashCalculator) *ShardCache {
	count := config.GetShardCount()
	c := &ShardCache{
		shardCount: uint64(count),
		shards:     make([]Cacher, count, count),
		hashCalc:   hashCalc,
	}

	for i := range c.shards {
		c.shards[i] = generator(config)
	}
	return c
}

func (c *ShardCache) getShard(str string) Cacher {
	key := c.hashCalc(str)
	return c.shards[key%c.shardCount]
}

//Get return item by name or nil
func (c *ShardCache) Get(name string) []byte {
	return c.getShard(name).Get(name)
}

//SetOrUpdate set or update item in cache
func (c *ShardCache) SetOrUpdate(name string, value []byte, expriation time.Duration) {
	c.getShard(name).SetOrUpdate(name, value, expriation)
}

//Purge delete all items from the cache
func (c *ShardCache) Purge() {
	for i := range c.shards {
		c.shards[i].Purge()
	}
}

//Dead stop all internal func and clear cache
func (c *ShardCache) Dead() {
	for i := range c.shards {
		c.shards[i].Dead()
	}
	c.shards = c.shards[:0]
}

//Statistic return all cache statistic
func (c *ShardCache) Statistic() Stats {
	s := Stats{}
	for i := range c.shards {
		shardStat := c.shards[i].Statistic()
		atomic.AddInt64(&s.DeleteCount, shardStat.DeleteCount)
		atomic.AddInt64(&s.DeleteExpired, shardStat.DeleteExpired)
		atomic.AddInt64(&s.GetErrorNumber, shardStat.GetErrorNumber)
		atomic.AddInt64(&s.GetSuccessNumber, shardStat.GetSuccessNumber)
		atomic.AddInt64(&s.ItemsCount, shardStat.ItemsCount)
		atomic.AddInt64(&s.SetOrReplaceCount, shardStat.SetOrReplaceCount)
		atomic.AddInt64(&s.SizeLimit, shardStat.SizeLimit)
	}
	return s
}
