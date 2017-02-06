# go-lru

go-lru is a lru cache bases on GroupCache, with expire time supported.

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
