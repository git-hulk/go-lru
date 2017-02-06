# go-lru

go-lru is an MIT-licensed Go LRU cache bases on GroupCache, with expire time supported

## Example

Set key with expire time

```
cache := NewCache(100) // max entries in cache is 100
cache.Set("a", 1234, 1) // key "a" would be expired after 1 second
```

Set key without expire time

```
cache := NewCache(100)
cache.Set("a", 1234) // set key "a" without expire time
```

## API doc

API documentation is available via  [https://godoc.org/github.com/git-hulk/go-lru](https://godoc.org/github.com/git-hulk/go-lru)
