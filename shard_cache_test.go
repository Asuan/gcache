package gcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func defaultShardConfig() ConfigShardCacheInterface {
	r := ConfigMessage{
		SizeLimit:         20000,
		DefaultExpiration: -1,
		IsKeepUsefull:     true,
		CacheType:         ConfigMessage_RWL,
		ShardCount:        10,
	}
	return &r
}

func getCacheGorBase() *ShardCache {
	gen := func(config ConfigCacheInterface) Cacher {
		return NewRwCache(config)
	}
	c := NewShardCache(defaultShardConfig(), gen, calcSUM)
	return c
}

func TestShardCache_Purge(t *testing.T) {
	as := assert.New(t)
	c := getCacheGorBase()
	c.SetOrUpdate("first", []byte(`zaza`), DefaultExpirationMarker)
	as.Equal(int64(1), c.Statistic().ItemsCount)
	as.Equal(int64(1), c.Statistic().SetOrReplaceCount)
	c.Purge()
	as.Equal(int64(0), c.Statistic().ItemsCount)
	as.Equal(int64(1), c.Statistic().DeleteCount)
	c.Dead() //Cleanup
}

func TestShardCache_Get(t *testing.T) {
	as := assert.New(t)
	c := getCacheGorBase()
	c.SetOrUpdate("first", []byte(`zaza`), DefaultExpirationMarker)
	c.SetOrUpdate("second", []byte(`azaz`), DefaultExpirationMarker)

	as.Nil(c.Get("irst"))
	as.Equal(int64(2), c.Statistic().ItemsCount)
	as.Equal([]byte(`zaza`), c.Get("first"))
	as.Equal([]byte(`azaz`), c.Get("second"))

	as.Equal(int64(2), c.Statistic().GetSuccessNumber)
	as.Equal(int64(1), c.Statistic().GetErrorNumber)
	as.Equal(int64(2), c.Statistic().ItemsCount)

	c.Dead() //Cleanup
}

func TestShardCache_SetOrUpdate(t *testing.T) {
	as := assert.New(t)
	c := getCacheGorBase()
	c.SetOrUpdate("first", []byte(`zaza`), (DefaultExpirationMarker))
	c.SetOrUpdate("second", []byte(`azaz`), (DefaultExpirationMarker))
	as.Nil(c.Get("irst"))
	time.Sleep(100)
	as.Equal(int64(2), c.Statistic().ItemsCount)
	as.Equal(int64(2), c.Statistic().SetOrReplaceCount)
	as.Equal(int64(1), c.Statistic().GetErrorNumber)
	as.Equal([]byte(`zaza`), c.Get("first"))
	as.Equal([]byte(`azaz`), c.Get("second"))

	c.SetOrUpdate("second", []byte(`zara`), (DefaultExpirationMarker))
	as.Equal(int64(2), c.Statistic().ItemsCount)
	as.Equal([]byte(`zara`), c.Get("second"))
	time.Sleep(100)

	as.Equal(int64(3), c.Statistic().SetOrReplaceCount)
	as.Equal(int64(3), c.Statistic().GetSuccessNumber)
	as.Equal(int64(1), c.Statistic().GetErrorNumber)
	as.Equal(int64(2), c.Statistic().ItemsCount)

	c.Dead() //Cleanup
}
