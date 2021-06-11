package memcache

import (
    "github.com/bradfitz/gomemcache/memcache"
)

type Memcache struct {
    cache *memcache.Client
}

func (cache *Memcache) FetchByLong(link string) (string, bool) {
    v, err := cache.cache.Get(link)
    if err != nil {
        return "", false
    }
    return string(v.Value), true
}

func (cache *Memcache) FetchByShort(link string) (string, bool) {
    v, err := cache.cache.Get(link)
    if err != nil {
        return "", false
    }
    return string(v.Value), true
}

func (cache *Memcache) Update(long string, short string) {
    cache.cache.Set(&memcache.Item{Key: long, Value: []byte(short)})
    cache.cache.Set(&memcache.Item{Key: short, Value: []byte(long)})
}

func (cache *Memcache) Ping() error {
    return cache.cache.Ping()
}

func New(servers ...string) *Memcache {
    mc := memcache.New(servers...)
    return &Memcache{cache: mc}
}
