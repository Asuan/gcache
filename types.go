package gcache

import "time"

const (
	// For use with functions that take an expiration time.
	NoExpiration time.Duration = -1
	// For use with functions that take an expiration time. Equivalent to
	// passing in the same expiration duration as was given to New() or
	// NewFrom() when the cache was created (e.g. 5 minutes.)
	DefaultExpirationMarker time.Duration = 0
	DefaultExpiration       time.Duration = time.Duration(5 * time.Minute)
)

//Stats is a statistic holder for cache
type Stats struct {
	ItemsCount,
	GetSuccessNumber,
	GetErrorNumber,
	SetOrReplaceCount,
	DeleteCount,
	DeleteExpired,
	SizeLimit int64
}

//Cacher interface for a storage
type Cacher interface {
	//Get func for getting item
	Get(name string) []byte
	//Set or update item
	SetOrUpdate(name string, value []byte, exp time.Duration)
	//Purge cache
	Purge()
	//Dead should stop cahing and clean
	Dead()
	//Return stats of cache
	Statistic() Stats
}

//Item is a wrapper for storage
type Item struct {
	Object     []byte
	Expiration int64
}
