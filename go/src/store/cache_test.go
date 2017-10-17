// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package blockstore

import (
	"fmt"
	pb "mawfs"
	"testing"
)

var _ = fmt.Print

func newTestCache() (*Cache, NodeStore) {
	store := NewChunkStore(NewFSInfo("bad-password"), NewFakeFileSys())
	cache := NewCache(store)
	return cache, store
}

type TestCacheObj struct {
	ObjImpl
	val int
}

func TestNewCache(t *testing.T) {
	cache, store := newTestCache()
	Assertf(t, cache.store == store, "cache.store == store")
}

func TestLru(t *testing.T) {
	cache, _ := newTestCache()
	cache.addObj(&TestCacheObj{val: 1})
	cache.addObj(&TestCacheObj{val: 2})
	cache.addObj(&TestCacheObj{val: 3})

	Assertf(t, cache.oldest.(*TestCacheObj).val == 1, "cache.oldest.val == 1")
	Assertf(t, cache.newest.(*TestCacheObj).val == 3, "cache.oldest.val == 3")
	Assertf(t, cache.oldest.GetNext().(*TestCacheObj).val == 2,
		"cache.oldest.next.val == 2")
	Assertf(t, cache.newest.GetPrev().(*TestCacheObj).val == 2,
		"cache.newest.prev.val == 2")
}

func TestChanges(t *testing.T) {
	cache, store := newTestCache()
	head := NewHead(cache, "master", nil)
	var one int32 = 1
	err := head.addChange(&pb.Change{Type: &one})
	Assertf(t, err == nil, "addChange returns error: %s", err)
	iter, err := store.MakeJournalIter("master")
	Assertf(t, err == nil, "MakeJournalIter returns error: %s", err)
	elem, err := iter.Elem()
	Assertf(t, err == nil, "iter ELem() returns error: %s", err)
	Assertf(t, elem.change.GetType() == 1,
		"failed to load change stored through Head")
}

func newEntry(cache *Cache, name string) *cachedEntry {
    node := &pb.Node{}
    entry := &pb.Entry{Name: &name}
    return newCachedEntry(entry,
                          NewCachedNode(cache, []byte("fake digest"), node), nil)
}

func Checkf(cond bool, format string, a ...interface{}) bool {
    if !cond {
        fmt.Printf(format, a...)
    }
    return cond
}

func checkIndex(ca *childArray, key string, expectedIndex int,
                expectedFound bool) bool {
    index, found := ca.findIndex(key)
    var foundText string = ""
    if !found {
        foundText = "not"
    }
    var pass bool = true
    if found != expectedFound {
        fmt.Printf("%s was unexpectedly %s found", key, foundText)
        pass = false
    }
    if index != expectedIndex {
        fmt.Printf("find %s != %d (got index of %d)", key, expectedIndex, index)
        pass = false
    }
    return pass
}

func TestFind(t *testing.T) {
    cache, _ := newTestCache()
    childArray := newChildArray(make([]*pb.Entry, 0))
    childArray.append(newEntry(cache, "bar"))
    childArray.append(newEntry(cache, "baz"))
    childArray.append(newEntry(cache, "blinky"))
    childArray.append(newEntry(cache, "curly"))
    childArray.append(newEntry(cache, "foo"))
    childArray.append(newEntry(cache, "inky"))
    childArray.append(newEntry(cache, "larry"))
    childArray.append(newEntry(cache, "moe"))
    childArray.append(newEntry(cache, "stinky"))

    Assert(t, checkIndex(childArray, "baz", 1, true))
    Assert(t, checkIndex(childArray, "blinky", 2, true))
    Assert(t, checkIndex(childArray, "curly", 3, true))
    Assert(t, checkIndex(childArray, "foo", 4, true))
    Assert(t, checkIndex(childArray, "inky", 5, true))
    Assert(t, checkIndex(childArray, "larry", 6, true))
    Assert(t, checkIndex(childArray, "moe", 7, true))
    Assert(t, checkIndex(childArray, "stinky", 8, true))

    Assert(t, checkIndex(childArray, "stank", 8, false))
    Assert(t, checkIndex(childArray, "aardvark", 0, false))

    // Try this with an empty array.
    Assert(t, checkIndex(newChildArray(make([]*pb.Entry, 0)), "howdy", 0,
                         false))
}

func newMemStoreCache() *Cache {
    store := NewMemStore(NewFSInfo("bad-password"))
    cache := NewCache(store)
    return cache
}

func TestLoad(t *testing.T) {
    cache := newMemStoreCache()
    head, err := cache.GetHead("master")
if !(err == nil) { t.Errorf("Assertion failed: err == nil: %s", err); return; }
if !(head != nil) { t.Errorf("Assertion failed: head != nil"); return; }

    // Try getting the root node.
    root, err := head.GetRoot()
    fmt.Print(err)
    Assert(t, err == nil)
    Assert(t, root != nil)
}
