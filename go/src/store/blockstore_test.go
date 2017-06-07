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
	"bytes"
	"crypto/sha256"
	pb "mawfs"
	"reflect"
	"testing"
)

func TestNewChunk(t *testing.T) {
	contents := []byte("contents")
	digest := []byte("digest")
	chunk := NewChunk(contents, digest)
	if bytes.Compare(chunk.contents, contents) != 0 {
		t.Error("bad contents: ", chunk.contents)
		t.Fail()
	}

	if bytes.Compare(chunk.digest, digest) != 0 {
		t.Error("bad digest: ", chunk.digest)
		t.Fail()
	}
}

const testContentsStr = "this is the contents of the file rendered here " +
	"for your unit testing pleasure!"

func TestReadChunk(t *testing.T) {
	fsinfo := NewFSInfo("bad-password")
	testContents := []byte(testContentsStr)
	encrypted, err := fsinfo.Encrypt(testContents)
	if err != nil {
		t.Error("unable to encrypt")
		t.Fail()
	}
	chunk, err := fsinfo.ReadChunk(bytes.NewBuffer(encrypted))
	if err != nil {
		t.Errorf("Failed to read chunk: %s", err)
		t.Fail()
	}
	if bytes.Compare(chunk.contents, testContents) != 0 {
		t.Error("Did not get plaintext contents back from chunk")
		t.Fail()
	}

	digest := sha256.Sum256(encrypted)
	if bytes.Compare(digest[:], chunk.digest) != 0 {
		t.Error("Read chunk had incorrect checksum.")
		t.Fail()
	}
}

func TestWriteChunk(t *testing.T) {
	fsinfo := NewFSInfo("bad-password")
	buf := bytes.NewBuffer([]byte{})
	testContents := []byte(testContentsStr)
	digest, err := fsinfo.WriteChunk(buf, testContents)
	expectedDigest := sha256.Sum256(buf.Bytes())
	if bytes.Compare(digest, expectedDigest[:]) != 0 {
		t.Error("digest wasn't what we expected.")
		t.Fail()
	}

	chunk, err := fsinfo.ReadChunk(buf)
	if err != nil {
		t.Errorf("Error reading chunk: %s", err)
		t.Fail()
	}
	if bytes.Compare(chunk.digest, digest) != 0 {
		t.Error("Chunk digest doesn't match encrypted digest")
		t.Fail()
	}
	if bytes.Compare(chunk.contents, testContents) != 0 {
		t.Error("Chunk plaintext doesn't match original plaintext")
		t.Fail()
	}
}

func TestStoreNode(t *testing.T) {
	var checksum int32 = 12345
	contents := "Here is some contents"
	node := &pb.Node{Checksum: &checksum, Contents: &contents}

	cs := NewChunkStore(NewFSInfo("bad-password"), NewFakeFileSys())
	digest, err := cs.StoreNode(node)
	if err != nil {
		t.Error("StoreNode failed: ", err)
		t.Fail()
		return
	}

	newNode, err := cs.LoadNode(digest)
	if !reflect.DeepEqual(node, newNode) {
		t.Error("Node contents was not preseerved")
		t.Fail()
	}
}

func TestMakeDigest(t *testing.T) {
	buf := &bytes.Buffer{}
	fsinfo := NewFSInfo("bad-password")
	cs := NewChunkStore(fsinfo, NewFakeFileSys())
	data := []byte("This is some test data")
	digest, err := fsinfo.WriteChunk(buf, data)
	if err != nil {
		t.Error(": ", err)
		t.Fail()
		return
	}

	newDigest, err := cs.MakeDigest(data)
	if err != nil {
		t.Error(": ", err)
		t.Fail()
		return
	}

	if bytes.Compare(newDigest, digest) != 0 {
		t.Error("MakeDigest doesn't produce the dame digest as WriteChunk")
		t.Fail()
	}
}

func TestCommits(t *testing.T) {
	root := []byte("123456")
	cs := NewChunkStore(NewFSInfo("bad-password"), NewFakeFileSys())
	commit := &pb.Commit{Parent: [][]byte{root}, Root: root}
	digest, err := cs.StoreCommit(commit)
	if err != nil {
		t.Error("StoreCommit: ", err)
		t.Fail()
		return
	}

	newCommit, err := cs.LoadCommit(digest)
	if err != nil {
		t.Error("LoadCommit: ", err)
		t.Fail()
		return
	}

	if !reflect.DeepEqual(commit, newCommit) {
		t.Error("Resurrected commit doesn't match")
		t.Fail()
		return
	}
}

func TestChunkStoreConformance(t *testing.T) {
	var ns NodeStore = NewChunkStore(NewFSInfo("bad-password"), NewFakeFileSys())
	ns.StoreCommit(&pb.Commit{Root: []byte("12345")})
}

func TestStoreRetrieveRootDigest(t *testing.T) {
	var ns NodeStore = NewChunkStore(NewFSInfo("bad-password"), NewFakeFileSys())
	node := &pb.Node{}
	digest, err := ns.StoreNode(node)
	err = ns.StoreRootDigest(digest)
	if err != nil {
		t.Error("StoreRootDigest failed: ", err)
		t.Fail()
	}

	newDigest, err := ns.LoadRootDigest()
	if err != nil {
		t.Error("LoadRootDigest: ", err)
		t.Fail()
		return
	}

	if !bytes.Equal(digest, newDigest) {
		t.Error("Digests don't match")
		t.Fail()
	}
}

func TestSetGetHead(t *testing.T) {
	var ns NodeStore = NewChunkStore(NewFSInfo("bad-password"), NewFakeFileSys())
	digest := sha256.Sum256([]byte(testContentsStr))
	err := ns.SetHead("branch", digest[:])
	if err != nil {
		t.Error("SetHead: ", err)
		t.Fail()
		return
	}

	head, err := ns.GetHead("branch")
	if err != nil {
		t.Error("GetHead: ", err)
		t.Fail()
		return
	}

	if !bytes.Equal(head, digest[:]) {
		t.Error("Digests don't match")
		t.Fail()
	}
}

func TestJournal(t *testing.T) {
	fs := NewFakeFileSys()
	var ns NodeStore = NewChunkStore(NewFSInfo("bad-password"), fs)

	var one int32 = 1
	var two int32 = 2
	digests := [][]byte{}
	digest, err := ns.WriteToJournal("branch1", &pb.Change{Type: &one})
	if err != nil {
		t.Error("WriteToJournal[1]: ", err)
		t.Fail()
		return
	}
	digests = append(digests, digest)

	digest, err = ns.WriteToJournal("branch1", &pb.Change{Type: &two})
	if err != nil {
		t.Error("WriteToJournal[2]: ", err)
		t.Fail()
		return
	}
	digests = append(digests, digest)

	iter, err := ns.MakeJournalIter("branch1")
	if err != nil {
		t.Error(": ", err)
		t.Fail()
		return
	}

	for i, val := range []int32{1, 2} {
		entry, err := iter.Elem()
		if err != nil {
			t.Error(": ", err)
			t.Fail()
			return
		}

		if !bytes.Equal(entry.digest, digests[i]) {
			t.Error("digest %d doesn't match expected value")
			t.Fail()
		}

		if entry.change.GetType() != val {
			t.Errorf("Change %d has a type of %d", i, entry.change.GetType())
			t.Fail()
		}
		iter.Next()
	}
	entry, err := iter.Elem()
	if err != nil {
		t.Error(": ", err)
		t.Fail()
		return
	}

	if entry != nil {
		t.Error("iter.Elem() != nil after last change.")
		t.Fail()
	}

	if !fs.Exists("journals/branch1") {
		t.Error("journals/branch1 does not exist")
		t.Fail()
	}
	ns.DeleteJournal("branch1")
	if fs.Exists("journals/branch1") {
		t.Error("journals/branch1 exists after removal")
		t.Fail()
	}
}
