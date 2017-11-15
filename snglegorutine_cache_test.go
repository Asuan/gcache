package gcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGorCache_Purge(t *testing.T) {
	as := assert.New(t)
	c := NewGorCache(defaultConfig())
	c.SetOrUpdate("first", []byte(`zaza`), DefaultExpirationMarker)
	c.Get("aa")
	as.Equal(int64(1), c.Statistic().ItemsCount)
	c.Purge()

	as.Equal(int64(0), c.Statistic().ItemsCount)
	as.Equal(int64(1), c.Statistic().DeleteCount)
	c.Dead() //Cleanup
}

func TestGorCache_Get(t *testing.T) {
	as := assert.New(t)
	c := NewGorCache(defaultConfig())
	c.SetOrUpdate("first", []byte(`zaza`), DefaultExpirationMarker)
	c.SetOrUpdate("second", []byte(`azaz`), DefaultExpirationMarker)

	as.Nil(c.Get("irst"))
	as.Equal(int64(2), c.Statistic().ItemsCount)
	as.Equal([]byte(`zaza`), c.Get("first"))
	as.Equal([]byte(`azaz`), c.Get("second"))
	as.Equal(int64(2), c.Statistic().ItemsCount)
	as.Equal(int64(2), c.Statistic().GetSuccessNumber)
	as.Equal(int64(1), c.Statistic().GetErrorNumber)
	as.Equal(int64(2), c.Statistic().ItemsCount)

	c.Dead() //Cleanup
}

func TestGorCache_SetOrUpdate(t *testing.T) {
	as := assert.New(t)
	c := NewGorCache(defaultConfig())
	c.SetOrUpdate("first", []byte(`zaza`), DefaultExpirationMarker)
	c.SetOrUpdate("second", []byte(`azaz`), DefaultExpirationMarker)
	as.Nil(c.Get("irst"))
	as.Equal(int64(2), c.Statistic().ItemsCount)
	as.Equal([]byte(`zaza`), c.Get("first"))
	as.Equal([]byte(`azaz`), c.Get("second"))

	c.SetOrUpdate("second", []byte(`zara`), DefaultExpirationMarker)
	as.Equal(int64(2), c.Statistic().ItemsCount)

	as.Equal([]byte(`zara`), c.Get("second"))

	as.Equal(int64(3), c.Statistic().SetOrReplaceCount)
	as.Equal(int64(3), c.Statistic().GetSuccessNumber)
	as.Equal(int64(1), c.Statistic().GetErrorNumber)
	as.Equal(int64(2), c.Statistic().ItemsCount)

	c.Dead() //Cleanup
}

//TODO check expiration time
//TODO check timeouts and isKeep...
