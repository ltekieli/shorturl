package cache

type Cache interface {
	FetchByLong(link string) (string, bool)
	FetchByShort(link string) (string, bool)
	Update(long string, short string)
	Ping() error
}
