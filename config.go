package gcache

//ConfigCacheInterface interface for Default caches like rwCache or sgCache of syncCache
type ConfigCacheInterface interface {
	GetDefaultExpiration() int64
	GetSizeLimit() int64
	GetIsKeepUsefull() bool
}

//ConfigShardCacheInterface extended interface for shard cache
type ConfigShardCacheInterface interface {
	ConfigCacheInterface
	GetShardCount() int64
	GetCacheType() ConfigMessage_CacheTypes
}
