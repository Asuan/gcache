package gcache

import "time"

type Stats struct {
	ItemsCount,
	GetSuccessNumber,
	GetErrorNumber,
	SetOrReplaceCount,
	DeleteCount,
	DeleteExpired,
	SizeLimit int64
}

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

type Item struct {
	Object     []byte
	Expiration int64
}
