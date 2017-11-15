package gcache

type ConfigCacheInterface interface {
	GetDefaultExpiration() int64
	GetSizeLimit() int64
	GetIsKeepUsefull() bool
}

type ConfigShardCacheInterface interface {
	ConfigCacheInterface
	GetShardCount() int64
	GetCacheType() ConfigMessage_CacheTypes
}
