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
	//"strings"  TODO: get latest go, use strings.Compare()
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

// The filesystem's in-memory cache.
type Cache struct {
	store NodeStore

	// Cache size where we start doing GC.
	gcThreshold uint

	// Cache size where we stop doing GC.
	gcBottom uint

	// Last and first elements of the doubly-linked list of GC objects.
	// Note that newest is the last element (by direction of "Next") and
	// oldest is the first.
	newest, oldest Obj

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

func NewCache(store NodeStore) *Cache {
	return &Cache{store: store,
		gcThreshold: DefaultGcThreshold,
		gcBottom:    DefaultGcBottom,
	}
}

func (c *Cache) addObj(obj Obj) {
	if obj.GetNext() != nil || obj.GetPrev() != nil {
		panic("Adding object that's already in the LRU chain.")
	}
	if c.oldest == nil {
		c.oldest = obj
	} else {
		c.newest.SetNext(obj)
		obj.SetPrev(c.newest)
	}
	c.newest = obj
}

// Encapsulates the current head of a branch in the filesystem.
//
// There are some constants in here that have an effect on the serialized
// representation.  These are mostly interesting for testing.
type Head struct {

	// The node cache.
	cache *Cache

	// The underlying node store.
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
}

// Creates a new Head object.
// baselineCommit may be nil if the branch is currently empty.
func NewHead(cache *Cache, branch string, baselineCommit []byte) *Head {
	return &Head{cache, cache.store, baselineCommit,
		nil,
		branch,
		DefaultMaxContentSize,
		DefaultMaxChildren,
		DefaultMaxJournalSize,
	}
}

func (head *Head) addChange(change *pb.Change) error {
	if head.lastChange != nil {
		change.LastChange = head.lastChange
	} else {
		change.Commit = head.baselineCommit
	}
	lastChange, err := head.store.WriteToJournal(head.branch, change)
	if err != nil {
		head.lastChange = lastChange
	}
	return err
}

// Wrapper around Node to manage its presence in the cache.
//
// CachedNode is a node in a sparse tree.  Its children may or may not be
// memory resident.  In general, CachedNodes are demand-loaded and remain in
// memory until a garbage collection.  Even after a garbage collection,
// "dirty" nodes will remain memory resident.
//
// Implements Obj.
type CachedNode struct {

    cache *Cache
    digest []byte
    node *pb.Node

    // Indicates that a node has been changed in memory and in the transaction
    // log but needs to be committed.  A dirty node is assumed to have an
    // invalid digest.  Likewise, a non-dirty node is assumed to have a valid
    // digest.  All nodes should either be loaded from the block store (in
    // which case, they have a valid digest) or created as part of an
    // operation (in which case they should have no digest and be dirty).
    dirty bool

    // The parent node (the directory if this is a directory or top-level
    // file node, an intermediate node for anything else).  Note that this
    // introduces a reference cycle, so you need to call release() on a node
    // to break this cycle (and also to remove the node from the LRU queue in
    // the cache).
    parent *CachedNode

    children []*cachedEntry
}

func (node *CachedNode) GetMode() int {
    return node.GetMode()
}

func NewCachedNode(cache *Cache, digest []byte, node *pb.Node) *CachedNode {
    return &CachedNode{cache: cache, digest: digest, node: node};
}

// Wrapper around Entry which serves the same purpose as CachedNode.
type cachedEntry struct {
    entry *pb.Entry
    cache *Cache
    node, parent *CachedNode
}

func newCachedEntry(entry *pb.Entry, node *CachedNode,
                    parent *CachedNode) *cachedEntry {
    return &cachedEntry{entry: entry, cache: node.cache, parent: parent}
}

// Returns the entry's name or nil if it doesn't have a name.
func (e *cachedEntry) GetName() *string {
    return e.entry.Name
}

// Returns the digest of the node that the entry references, nil if the node
// is not yet committed.
func (e *cachedEntry) GetDigest() []byte {
    return e.entry.GetHash()
}

// A managed array of entries.
type childArray struct {
    rep []*pb.Entry
    cached []*cachedEntry
}

func newChildArray(rep []*pb.Entry) *childArray {
    return &childArray{rep: rep, cached: make([]*cachedEntry, len(rep))}
}

func Compare(a, b string) int {
    if (a == b) {
        return 0
    } else if (a > b) {
        return 1
    }
    return -1
}

func (ca *childArray) findIndexHelper(name string, start, end uint) (uint, bool) {
    if len(ca.cached) == 0 {
        return 0, false
    }

    midpoint := (end - start) / 2 + start
    if midpoint == start {
        comparison := Compare(name, *ca.cached[midpoint].GetName())
        switch {
            case comparison == 0:
                return start, true
            case comparison < 0:
                return start, false
            default:
                return end, false
        }
    }

    switch {
        case name == *ca.cached[midpoint].GetName():
            return midpoint, true
        case name < *ca.cached[midpoint].GetName():
            return ca.findIndexHelper(name, start, midpoint)
        default:
            return ca.findIndexHelper(name, midpoint, end)
    }
}

func (ca *childArray) findIndex(name string) (uint, bool) {
    return ca.findIndexHelper(name, 0, uint(len(ca.rep)))
}

// Appends a new entry onto the childArray.
func (ca *childArray) append(entry *cachedEntry) {
    ca.cached = append(ca.cached, entry)
    ca.rep = append(ca.rep, entry.entry)
}
