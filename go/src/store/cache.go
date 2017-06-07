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
)

var _ = fmt.Print

const (
	Meg                   = 1024 * 1024
	DefaultMaxContentSize = Meg
	DefaultMaxChildren    = 256
	DefaultMaxJournalSize = 16 * Meg
	DefaultGcThreshold    = 128 * Meg
	DefaultGcBottom       = 16 * Meg
)

// Base class for cached objects.
type Obj interface {
	GetNext() Obj
	SetNext(next Obj)
	GetPrev() Obj
	SetPrev(prev Obj)
}

type ObjImpl struct {
	next, prev Obj
}

func (o *ObjImpl) GetNext() Obj {
	return o.next
}

func (o *ObjImpl) SetNext(next Obj) {
	o.next = next
}

func (o *ObjImpl) GetPrev() Obj {
	return o.prev
}

func (o *ObjImpl) SetPrev(prev Obj) {
	o.prev = prev
}

// This structure should probably get broken out into a few different things.
// The cache proper should actually just be a cache and should manage
// memory-resident instantiations of nodes from the node store.  There are
// also a bunch of branch-specific concepts that should maybe go into a
// "Branch" class - the baseline commit, last change digest and branch name.
//
// There are some constants that have an effect on the serialized
// representation.  These are mostly interesting for testing.
//
// There are constants for tuning runtime behavior, like gcThreshold, which
// clearly belongs in the cache proper.
type Cache struct {
	store NodeStore

	// The digest of the last commit.  Changes in the journal are relative to
	// this.
	baselineCommit []byte

	// The digest of the last change.
	lastChange []byte

	// The name of the branch.  "master" is the default branch.
	branch string

	maxContentSize uint
	maxChildren    uint
	maxJournalSize uint

	// Cache size where we start doing GC.
	gcThreshold uint

	// Cache size where we stop doing GC.
	gcBottom uint

	// Last and first elements of the doubly-linked list of GC objects.
	// Note that newest is the last element (by direction of "Next") and
	// oldest is the first.
	newest, oldest Obj

	//    oper init(NodeStore store, String branch, String baselineCommit) :
	//        store = store,
	//        branch = branch,
	//        baselineCommit = baselineCommit {
	//    }
	//
	//    @final void addChange(Change change) {
	//        if (lastChange) {
	//            change.lastChange = lastChange;
	//        } else {
	//            change.commit = baselineCommit;
	//        }
	//        lastChange = store.writeToJournal(branch, change);
	//    }
	//
	//    ## Returns true if the caller should commit.
	//    @final bool shouldCommit() {
	//        return store.getJournalSize(branch) >= maxJournalSize;
	//    }
	//
	//    @final String storeNode(Node node) {
	//        return store.storeNode(node);
	//    }
	//
	//    @final Node getNode(String digest) {
	//        return store.getNode(digest);
	//    }
	//
	//    @final void clearJournal() {
	//        store.deleteJournal(branch);
	//    }
	//
	//    @final JournalIter makeJournalIter() {
	//        return store.makeJournalIter(branch);
	//    }
	//
	//    ## Records the digest of a new commit.
	//    @final void recordCommit(String commit) {
	//        lastChange = null;
	//        baselineCommit = commit;
	//    }
	//
	//    ## Adds a new object as the most recently used.
	//    @final void addObj(Obj obj) {
	//        @assert(!obj.next && !obj.prev);
	//        __oldest.append(obj);
	//    }
	//
	//    ## Release the object from the LRU queue.
	//    @final void releaseObj(Obj obj) {
	//        __oldest.remove(obj);
	//    }
	//
	//    ## Bring an object to the end of the least recently used queue.
	//    @final void touch(Obj obj) {
	//        if (!(__oldest.tail is obj)) {
	//            __oldest.remove(obj);
	//            __oldest.append(obj);
	//        }
	//    }
	//
	//    ## Run garbage collection.  'amount' is the number of bytes that we want
	//    ## to release.
	//    void garbageCollect(uintz amount) {
	//        Obj cur = __oldest.head;
	//        uintz amountPruned;
	//        while (cur && amountPruned < amount) {
	//            if (cur.disposable()) {
	//                amountPruned += cur.getRSize();
	//                tmp := cur;
	//                cur = cur.next;
	//                tmp.release();
	//            } else {
	//                cur = cur.next;
	//            }
	//        }
	//    }
}

func NewCache(store NodeStore, branch string, baselineCommit []byte) *Cache {
	cache := &Cache{
		store:          store,
		branch:         branch,
		baselineCommit: baselineCommit,
	}
	cache.store = store
	cache.maxContentSize = DefaultMaxContentSize
	cache.maxChildren = DefaultMaxChildren
	cache.maxJournalSize = DefaultMaxJournalSize

	cache.gcThreshold = DefaultGcThreshold
	cache.gcBottom = DefaultGcBottom

	return cache
}

func (c *Cache) addObj(obj Obj) {
	if obj.GetNext() != nil || obj.GetPrev() != nil {
		panic("Adding object that's already in the LRU chain.")
	}
	if c.oldest == nil {
		c.oldest = obj
	} else {
		c.newest.SetNext(obj)
		fmt.Printf("setting prev of %s to %s\n", obj, c.newest)
		obj.SetPrev(c.newest)
	}
	c.newest = obj
}
