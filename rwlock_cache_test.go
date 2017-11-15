package gcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func defaultConfig() ConfigCacheInterface {
	r := ConfigMessage{
		SizeLimit:         20000,
		DefaultExpiration: -1,
		IsKeepUsefull:     true,
		CacheType:         ConfigMessage_RWL,
	}
	return &r
}

func TestRwlockcache_Purge(t *testing.T) {
	as := assert.New(t)
	c := NewRwCache(defaultConfig())
	c.SetOrUpdate("first", []byte(`zaza`), DefaultExpirationMarker)
	as.Equal(1, len(c.m))
	c.Purge()

	as.Equal(0, len(c.m))
	as.Equal(int64(1), c.Statistic().DeleteCount)
	c.Dead() //Cleanup
}

func TestRwlockcache_Get(t *testing.T) {
	as := assert.New(t)
	c := NewRwCache(defaultConfig())
	c.SetOrUpdate("first", []byte(`zaza`), DefaultExpirationMarker)
	c.SetOrUpdate("second", []byte(`azaz`), DefaultExpirationMarker)

	as.Nil(c.Get("irst"))
	as.Equal(2, len(c.m))
	as.Equal([]byte(`zaza`), c.Get("first"))
	as.Equal([]byte(`azaz`), c.Get("second"))
	as.Equal(2, len(c.m))
	as.Equal(int64(2), c.Statistic().GetSuccessNumber)
	as.Equal(int64(1), c.Statistic().GetErrorNumber)
	as.Equal(int64(2), c.Statistic().ItemsCount)

	c.Dead() //Cleanup
}

func TestRwlockcache_SetOrUpdate(t *testing.T) {
	as := assert.New(t)
	c := NewRwCache(defaultConfig())
	c.SetOrUpdate("first", []byte(`zaza`), (DefaultExpirationMarker))
	c.SetOrUpdate("second", []byte(`azaz`), (DefaultExpirationMarker))
	as.Nil(c.Get("irst"))
	as.Equal(2, len(c.m))
	as.Equal([]byte(`zaza`), c.Get("first"))
	as.Equal([]byte(`azaz`), c.Get("second"))

	c.SetOrUpdate("second", []byte(`zara`), (DefaultExpirationMarker))
	as.Equal(2, len(c.m))

	as.Equal([]byte(`zara`), c.Get("second"))

	as.Equal(int64(3), c.Statistic().SetOrReplaceCount)
	as.Equal(int64(3), c.Statistic().GetSuccessNumber)
	as.Equal(int64(1), c.Statistic().GetErrorNumber)
	as.Equal(int64(2), c.Statistic().ItemsCount)

	c.Dead() //Cleanup
}
