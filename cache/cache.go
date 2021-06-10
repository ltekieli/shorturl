package cache

type Cache interface {
	FetchByLong(link string) (string, bool)
	FetchByShort(link string) (string, bool)
	Update(long string, short string)
}

type InMemoryCache struct {
	MappingLongToShort map[string]string
	MappingShortToLong map[string]string
}

func (cache *InMemoryCache) FetchByLong(link string) (string, bool) {
    v, found := cache.MappingLongToShort[link]
    return v, found
}

func (cache *InMemoryCache) FetchByShort(link string) (string, bool) {
    v, found := cache.MappingShortToLong[link]
    return v, found
}

func (cache *InMemoryCache) Update(long string, short string) {
	cache.MappingLongToShort[long] = short
	cache.MappingShortToLong[short] = long
}

func New() *InMemoryCache {
	return &InMemoryCache{make(map[string]string), make(map[string]string)}
}
