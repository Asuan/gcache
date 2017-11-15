package gcache

import (
	"sync/atomic"
	"time"
)

type ShardCache struct {
	Cacher
	shards     []Cacher
	shardCount uint64
	hashCalc   HashCalculator
}

type ShardGenerator func(ConfigCacheInterface) Cacher

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

func (c *ShardCache) GetShard(str string) Cacher {
	key := c.hashCalc(str)
	return c.shards[key%c.shardCount]
}

func (c *ShardCache) Get(name string) []byte {
	return c.GetShard(name).Get(name)
}

func (c *ShardCache) SetOrUpdate(name string, value []byte, expriation time.Duration) {
	c.GetShard(name).SetOrUpdate(name, value, expriation)
}

func (c *ShardCache) Purge() {
	for i := range c.shards {
		c.shards[i].Purge()
	}
}

func (c *ShardCache) Dead() {
	for i := range c.shards {
		c.shards[i].Dead()
	}
	c.shards = c.shards[:0]
}

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
