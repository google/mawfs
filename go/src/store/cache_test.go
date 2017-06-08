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
	for cur := cache.newest; cur != nil; cur = cur.GetPrev() {
		fmt.Printf("elem is %d", cur.(*TestCacheObj).val)
	}
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
