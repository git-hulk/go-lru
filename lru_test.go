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

package lru

import (
	"fmt"
	"testing"
	"time"
)

type simpleStruct struct {
	int
	string
}

type complexStruct struct {
	int
	simpleStruct
}

var getTests = []struct {
	name       string
	keyToAdd   interface{}
	keyToGet   interface{}
	expectedOk bool
}{
	{"string_hit", "myKey", "myKey", true},
	{"string_miss", "myKey", "nonsense", false},
	{"simple_struct_hit", simpleStruct{1, "two"}, simpleStruct{1, "two"}, true},
	{"simeple_struct_miss", simpleStruct{1, "two"}, simpleStruct{0, "noway"}, false},
	{"complex_struct_hit", complexStruct{1, simpleStruct{2, "three"}},
		complexStruct{1, simpleStruct{2, "three"}}, true},
}

func TestGet(t *testing.T) {
	for _, tt := range getTests {
		lru := NewCache(0)
		lru.Set(tt.keyToAdd, 1234)
		val, ok := lru.Get(tt.keyToGet)
		if ok != tt.expectedOk {
			t.Fatalf("%s: cache hit = %v; want %v", tt.name, ok, !ok)
		} else if ok && val != 1234 {
			t.Fatalf("TestGet failed, %s expected 1234, got %v", tt.name, val)
		}
	}
}

func TestExpire(t *testing.T) {
	lru := NewCache(100)
	expireKey := "keyWithExpire"
	expireWithoutKey := "keyWithoutExpire"
	lru.Set(expireKey, 1234, 1)
	lru.Set(expireWithoutKey, 1234)
	val, ok := lru.Get(expireKey)
	if !ok || val != 1234 || lru.Len() != 2 {
		t.Fatal("TestExpire get val error")
	}
	time.Sleep(1100 * time.Millisecond)
	val, ok = lru.Get(expireKey)
	if ok || val == 1234 {
		t.Fatal("TestExpire get expire error")
	}
	val, ok = lru.Get(expireWithoutKey)
	if !ok || val != 1234 || lru.Len() != 1 {
		t.Fatal("TestExpire get val error")
	}
}

func TestRemove(t *testing.T) {
	lru := NewCache(0)
	lru.Set("myKey", 1234)
	if val, ok := lru.Get("myKey"); !ok {
		t.Fatal("TestRemove returned no match")
	} else if val != 1234 {
		t.Fatalf("TestRemove failed, expected %d, got %v", 1234, val)
	}

	lru.Remove("myKey")
	if _, ok := lru.Get("myKey"); ok {
		t.Fatal("TestRemove returned a removed entry")
	}
}

func TestTTL(t *testing.T) {
	expireKey := "expireKey"
	expireTime := 2
	lru := NewCache(100)
	lru.Set(expireKey, 1234, expireTime)
	ttl := lru.TTL(expireKey)
	fmt.Println(ttl)
	if ttl < 1 || ttl > 2 {
		t.Fatal("TestTtl got ttl < 0")
	}
	lru.Set("noExpireKey", 1234)
	if lru.TTL("noExpireKey") != -1 {
		t.Fatal("TestTtl not expire time key ttl != -1")
	}
	if lru.TTL("notExistKey") != -2 {
		t.Fatal("TestTtl not exist key ttl != -2")
	}
}

func TestKeys(t *testing.T) {
	lru := NewCache(100)
	lru.Set("a", 1)
	lru.Set("b", 1, 1)
	keys := lru.Keys()
	if len(keys) != 2 {
		t.Fatalf("TestKeys failed, expected %d, got %d", 2, len(keys))
	}
	time.Sleep(1100 * time.Millisecond)
	keys = lru.Keys()
	if len(keys) != 1 {
		t.Fatalf("TestKeys failed, expected %d, got %d", 1, len(keys))
	}
}

func TestFlushAll(t *testing.T) {
	lru := NewCache(100)
	lru.Set("a", 1)
	lru.Set("b", 1)
	lru.Set("c", 1)
	lru.FlushAll()
	lru.Set("a", 1)
	if lru.Len() != 1 {
		t.Fatalf("TestFlushAll failed, expected %d, got %d", 1, lru.Len())
	}
}
