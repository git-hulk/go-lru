/*
Copyright 2013 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package lru implements an LRU cache.
package lru

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

// Cache is an LRU cache. It is not safe for concurrent access.
type Cache struct {
	// MaxEntries is the maximum number of cache entries before
	// an item is evicted. Zero means no limit.
	MaxEntries int

	// OnEvicted optionally specificies a callback function to be
	// executed when an entry is purged from the cache.
	OnEvicted func(key Key, value interface{})

	ll    *list.List
	cache map[interface{}]*list.Element
	lock  sync.RWMutex
}

// A Key may be any value that is comparable. See http://golang.org/ref/spec#Comparison_operators
type Key interface{}

type entry struct {
	key    Key
	value  interface{}
	expire int64 // unit ns
}

// NewCache creates a new Cache.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
func NewCache(maxEntries int) *Cache {
	c := &Cache{
		MaxEntries: maxEntries,
		ll:         list.New(),
		cache:      make(map[interface{}]*list.Element),
	}
	go c.expiredBackground()
	return c
}

// Check whether entry is expired or not
func (e *entry) isExpired() bool {
	if e.expire <= 0 { // entry without expire
		return false
	}

	now := time.Now().UnixNano()
	if e.expire >= now {
		return false
	}
	return true
}

func (c *Cache) expiredBackground() {
	hz := 10
	minIterations := 50
	defer func() {
		if err := recover(); err != nil {
			// Do nothing
		}
	}()

	ticker := time.NewTicker(time.Second / time.Duration(hz))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// In the worst case we process all the keys in 10 seconds
			// In normal conditions (a reasonable number of keys) we process
			// all the keys in a shorter time.
			numEntries := c.ll.Len()
			iterations := numEntries / (hz * 10)
			if iterations < minIterations {
				iterations = minIterations
				if numEntries < 50 {
					iterations = numEntries
				}
			}
			c.lock.Lock()
			for i := 0; i < iterations && c.ll.Len() > 0; i++ {
				ele := c.ll.Back()
				if ele.Value.(*entry).isExpired() {
					c.removeElement(ele)
				} else {
					c.ll.MoveToFront(ele)
				}
			}
			c.lock.Unlock()
		}
	}
}

// Set a value to the cache.
// Key and value is required, and expire time is option,
// Set("a", 1) or Set("a", 1, 1) is ok.
func (c *Cache) Set(key Key, value interface{}, args ...interface{}) (bool, error) {
	var expire int64
	var ttl int64
	if c.cache == nil {
		return false, errors.New("cache is not initialized")
	}

	expire = -1
	if args != nil && len(args) >= 1 {
		ttl = 0
		switch t := args[0].(type) {
		case int:
			ttl = int64(t)
		case int8:
			ttl = int64(t)
		case int16:
			ttl = int64(t)
		case int32:
			ttl = int64(t)
		case int64:
			ttl = t
		default:
			return false, errors.New("expire time should be int, int8, int16, int32, int64")
		}
		if ttl <= 0 {
			return false, errors.New("expire time should be > 0")
		}
		expire = time.Now().UnixNano() + ttl*1e9
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if ee, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ee)
		ee.Value.(*entry).value = value
		ee.Value.(*entry).expire = expire
		return true, nil
	}
	ele := c.ll.PushFront(&entry{key, value, expire})
	c.cache[key] = ele
	if c.MaxEntries != 0 && c.ll.Len() > c.MaxEntries {
		c.removeOldest()
	}
	return true, nil
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key Key) (value interface{}, ok bool) {
	if c.cache == nil {
		return nil, false
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if ele, hit := c.cache[key]; hit {
		if ele.Value.(*entry).isExpired() {
			// delete expired elem
			c.removeElement(ele)
			return nil, false
		}
		c.ll.MoveToFront(ele)
		return ele.Value.(*entry).value, true
	}
	return nil, false
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key Key) {
	if c.cache == nil {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if ele, hit := c.cache[key]; hit {
		c.removeElement(ele)
	}
}

// removeOldest removes the oldest item from the cache.
func (c *Cache) removeOldest() {
	if c.cache == nil {
		return
	}
	ele := c.ll.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

func (c *Cache) removeElement(e *list.Element) {
	c.ll.Remove(e)
	kv := e.Value.(*entry)
	delete(c.cache, kv.key)
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	if c.cache == nil {
		return 0
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.ll.Len()
}

// TTL return the ttl for key.
// Return -2 when key is not found,
// return -1 when key without expire time,
// return > 0 when key with expire time.
func (c *Cache) TTL(key Key) int64 {
	if c.cache == nil {
		return -2
	}

	c.lock.RLock()
	defer c.lock.RUnlock()
	if ele, hit := c.cache[key]; hit {
		expire := ele.Value.(*entry).expire
		if expire <= 0 {
			return -1
		}
		ttl := expire - time.Now().UnixNano()
		if ttl > 0 {
			return ttl / 1e9
		}
		return 0
	}
	return -2
}
