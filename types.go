package gcache

import "time"

const (
	//NoExpiration mean It willl not be deleted by timeout
	NoExpiration time.Duration = -1
	//DefaultExpirationMarker identify is need set default expiration or not
	DefaultExpirationMarker time.Duration = 0
	//DefaultExpiration default expiration time
	DefaultExpiration time.Duration = time.Duration(5 * time.Minute)
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
	//Purge cache cleanup but it still alive
	Purge()
	//Dead should stop cashing and clean
	Dead()
	//Return stats of cache
	Statistic() Stats
}

//Item is a wrapper for storage
type Item struct {
	Object     []byte
	Expiration int64
}
