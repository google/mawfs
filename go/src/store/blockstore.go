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

// Basic blockstore functionality.  Code for storing and encrypting chunks and
// nodes and history.

package blockstore

import (
	"bytes"
	"crypto/sha256"
	"github.com/golang/protobuf/proto"
	"github.com/jacobsa/crypto/siv"
	"io"
	pb "mawfs"
	"os"
)

const BlockSize = 65536

const (
    MODE_DIR = 1
    MODE_EXE = 2
)

type Chunk struct {
	contents []byte
	digest   []byte
}

func NewChunk(contents, digest []byte) *Chunk {
	return &Chunk{contents: contents, digest: digest}
}

type Cipher interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

type SivCipher struct {
	// The encryption/decryption key, must be 32 bytes.
	key []byte
}

func (cipher *SivCipher) Encrypt(plaintext []byte) ([]byte, error) {
	return siv.Encrypt(nil, cipher.key, plaintext, nil)
}

func (cipher *SivCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	return siv.Decrypt(cipher.key, ciphertext, nil)
}

type FSInfo struct {
	cipher Cipher
}

// Cretaes a new FSInfo object from the given password.  The password can be
// ordinary UIT8 text and of any length, the actual key will be generated from
// its SHA256 sum.
func NewFSInfo(password string) *FSInfo {
	key := sha256.Sum256([]byte(password))
	return &FSInfo{&SivCipher{key[:]}}
}

// Returns plaintext encrypted with the filesystem's cipher.
func (f *FSInfo) Encrypt(plaintext []byte) ([]byte, error) {
	return f.cipher.Encrypt(plaintext)
}

// Returns ciphertext decrypted with the filesystem's cipher.
func (f *FSInfo) Decrypt(ciphertext []byte) ([]byte, error) {
	return f.cipher.Decrypt(ciphertext)
}

func (f *FSInfo) ReadChunk(src io.Reader) (*Chunk, error) {
	// Read in the entire ciphertext.
	ciphertextBuf := bytes.NewBuffer([]byte{})
	ciphertextBuf.ReadFrom(src)

	// Get the digest.
	digest := sha256.Sum256(ciphertextBuf.Bytes())

	// Decrypt.
	plaintext, err := f.Decrypt(ciphertextBuf.Bytes())
	if err != nil {
		return nil, err
	}

	return &Chunk{contents: plaintext, digest: digest[:]}, nil
}

// Writes the chunk to the writer, returns the digest of the encrypted data.
func (f *FSInfo) WriteChunk(dst io.Writer, data []byte) ([]byte, error) {
	encrypted, err := f.Encrypt(data)
	if err != nil {
		return nil, err
	}
	_, err = dst.Write(encrypted)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(encrypted)
	return digest[:], nil
}

type ChangeEntry struct {
	digest []byte
	change pb.Change
}

type JournalIter interface {
	Elem() (*ChangeEntry, error)
	Next() error
	IsValid() bool
}

type UnknownName struct {
	s string
}

func (err UnknownName) Error() string {
	return err.s
}

type NodeStore interface {
	// Store a Node, return its digest.
	StoreNode(node *pb.Node) ([]byte, error)

	// Compute and return the digest for the node.  This is just like
	// storeNode() only it doesn't store the node, it just creates its digest.
	MakeDigest(data []byte) ([]byte, error)

	// Get the node at the given digest, nil if it's not currently stored.
	LoadNode(digest []byte) (*pb.Node, error)

	// Stores a Commit, returns its digest.
	StoreCommit(commit *pb.Commit) ([]byte, error)

	// Retrieves a commit object from its digest, an error if it's not
	// currently stored.
	LoadCommit(digest []byte) (*pb.Commit, error)

	//    # Get the root node (null if there is no root).
	//    @abstract Node getRoot();
	//
	// Get the root digest (null if there is no root).
	LoadRootDigest() ([]byte, error)

	// Store the digest of the root node.
	StoreRootDigest(digest []byte) error

	// Returns the digest of the head commit of a given branch.   Returns
	// null and a UnknownName error if the branch is not defined.
	GetHead(branch string) ([]byte, error)

	// Sets the digest of the head commit for the branch.
	SetHead(branch string, digest []byte) error

	// Write a change to the journal for the branch.  Returns the digest of
	// the change.
	WriteToJournal(branch string, change *pb.Change) ([]byte, error)

	// Delete the journal for a branch.
	DeleteJournal(branch string) error

	// Return an iterator over the journal
	MakeJournalIter(branch string) (JournalIter, error)

	//    ## Returns the size of the journal, or rather, the size of all of the
	//    ## changes in it.
	//    @abstract uint getJournalSize(String branch);
}

// Wraps a file in an interface.
type File interface {
	io.Closer
	io.Reader
	io.Writer
}

// Wraps a filesystem in an interface to improve testability.
type FileSys interface {
	Create(name string) (File, error)
	Open(name string) (File, error)
	Append(name string) (File, error)
	Exists(name string) bool
	Mkdir(name string) error
	Remove(name string) error
}

// Implements File.
type BackingDir struct {
	root string
}

func checkBackingDirIfaces() {
	var _ FileSys = &BackingDir{}
}

func (bd BackingDir) Create(name string) (File, error) {
	return os.Create(bd.root + name)
}

func (bd BackingDir) Open(name string) (File, error) {
	return os.Open(bd.root + name)
}

func (bd BackingDir) Append(name string) (File, error) {
	return os.OpenFile(name,
		os.O_WRONLY|os.O_APPEND|os.O_SYNC|os.O_CREATE,
		0)
}

func (bd BackingDir) Exists(name string) bool {
	if _, err := os.Stat(bd.root + name); os.IsNotExist(err) {
		return false
	} else if err == nil {
		return true
	} else {
		panic(err.Error())
	}
}

func (bd BackingDir) Mkdir(name string) error {
	return os.Mkdir(name, 0700)
}

func (bd BackingDir) Remove(name string) error {
	return os.Remove(name)
}

// NodeStore implementation that writes to a backing filesystem directory.
type ChunkStore struct {
	fsInfo  *FSInfo
	backing FileSys
}

func NewChunkStore(fsInfo *FSInfo, backing FileSys) *ChunkStore {
	return &ChunkStore{fsInfo, backing}
}

// Stores chunk data among the objects (filenames are the alt-encoded
// digests), returns the digest.
func (cs *ChunkStore) store(obj proto.Message) ([]byte, error) {
	rep, err := proto.Marshal(obj)
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	digest, err := cs.fsInfo.WriteChunk(&buf, rep)
	if err != nil {
		return nil, err
	}

	dst, err := cs.backing.Create(altEncode(digest))
	if err != nil {
		return nil, err
	}

	dst.Write(buf.Bytes())
	return digest, nil
}

func (cs *ChunkStore) load(digest []byte) (*Chunk, error) {
	src, err := cs.backing.Open(altEncode(digest))
	if err != nil {
		return nil, err
	}
	defer src.Close()

	return cs.fsInfo.ReadChunk(src)
}

func (cs *ChunkStore) StoreNode(node *pb.Node) ([]byte, error) {
	return cs.store(node)
}

func (cs *ChunkStore) LoadNode(digest []byte) (*pb.Node, error) {
	chunk, err := cs.load(digest)
	if err != nil {
		return nil, err
	}

	node := &pb.Node{}
	err = proto.Unmarshal(chunk.contents, node)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (cs *ChunkStore) MakeDigest(data []byte) ([]byte, error) {
	buf := bytes.Buffer{}
	digest, err := cs.fsInfo.WriteChunk(&buf, data)
	return digest, err
}

func (cs *ChunkStore) StoreCommit(commit *pb.Commit) ([]byte, error) {
	return cs.store(commit)
}

func (cs *ChunkStore) LoadCommit(digest []byte) (*pb.Commit, error) {
	chunk, err := cs.load(digest)
	if err != nil {
		return nil, err
	}

	commit := &pb.Commit{}
	err = proto.Unmarshal(chunk.contents, commit)
	if err != nil {
		return nil, err
	}

	return commit, nil
}

func (cs *ChunkStore) LoadRootDigest() ([]byte, error) {
	src, err := cs.backing.Open("refs/root")
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	buf.ReadFrom(src)
	return altDecode(buf.String())
}

func (cs *ChunkStore) StoreRootDigest(digest []byte) error {
	encoded := altEncode(digest)
	dst, err := cs.backing.Create("refs/root")
	if err != nil {
		return err
	}

	dst.Write([]byte(encoded))
	return nil
}

func (cs *ChunkStore) GetHead(branch string) ([]byte, error) {
	if !cs.backing.Exists("refs/" + branch) {
		return nil, UnknownName{"Unknown name: " + branch}
	}
	src, err := cs.backing.Open("refs/" + branch)
	if err != nil {
		return nil, err
	}
	buf := bytes.Buffer{}
	buf.ReadFrom(src)
	return altDecode(buf.String())
}

func (cs *ChunkStore) SetHead(branch string, digest []byte) error {
	dst, err := cs.backing.Create("refs/" + branch)
	if err != nil {
		return err
	}

	_, err = dst.Write([]byte(altEncode(digest)))
	dst.Close()
	return err
}

func (cs *ChunkStore) WriteToJournal(branch string, change *pb.Change) (
	[]byte, error) {

	// Create the journals directory first.
	if !cs.backing.Exists("journals") {
		err := cs.backing.Mkdir("journals")
		if err != nil {
			return nil, err
		}
	}

	rep, err := proto.Marshal(change)
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	digest, err := cs.fsInfo.WriteChunk(&buf, rep)
	if err != nil {
		return nil, err
	}

	envelope := proto.Buffer{}
	err = envelope.EncodeRawBytes(buf.Bytes())
	if err != nil {
		return nil, err
	}

	dst, err := cs.backing.Append("journals/" + branch)
	if err != nil {
		return nil, err
	}

	_, err = dst.Write(envelope.Bytes())
	return digest, err
}

func (cs *ChunkStore) DeleteJournal(branch string) error {
	return cs.backing.Remove(branch)
}

// Implements JournalIter.
type csJournalIter struct {
	src    File
	cs     *ChunkStore
	change *ChangeEntry
}

func (i *csJournalIter) readChange() (*ChangeEntry, error) {
	// Read the first 8 bytes which will be long enough for the chunk size.  It
	// will also necessarily be shorter than the size + chunk, because every
	// encrypted Change record has a 16 byte SIV header.
	buf := bytes.NewBuffer(make([]byte, 8))
	_, err := i.src.Read(buf.Bytes())
	if err != nil {
		return nil, err
	}

	// Try to decode a varint.
	size, n := proto.DecodeVarint(buf.Bytes())
	if n == 0 {
		return nil,
			&DecodingError{"Journal change is too large to be realistic."}
	}

	// Skip the bytes for the varint we just processed.
	buf.Next(n)

	// Read in the rest of the protobuf.
	remaining := size - (8 - uint64(n))
	remainingBuf := make([]byte, remaining)
	n, err = i.src.Read(remainingBuf)
	if uint64(n) != remaining {
		return nil, &DecodingError{"Incomplete change record."}
	} else if err != nil {
		return nil, err
	}
	buf.Write(remainingBuf)

	// Read a chunk out of it.
	chunk, err := i.cs.fsInfo.ReadChunk(buf)
	if err != nil {
		return nil, err
	}

	// Unmarshal the chunk.
	var entry ChangeEntry
	err = proto.Unmarshal(chunk.contents, &entry.change)
	if err != nil {
		return nil, err
	}
	entry.digest = chunk.digest
	return &entry, nil
}

func (cs *csJournalIter) IsValid() bool {
	return cs.change != nil
}

func newCsJournalIter(src File, cs *ChunkStore) (*csJournalIter, error) {
	result := csJournalIter{src: src, cs: cs}
	var err error
	result.change, err = result.readChange()
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &result, nil
}

// There was an error decoding an object from storage.
//
// Implements error.
type DecodingError struct {
	msg string
}

func (err *DecodingError) Error() string {
	return err.msg
}

func (i *csJournalIter) Elem() (*ChangeEntry, error) {
	return i.change, nil
}

func (cs *csJournalIter) Next() error {
	var err error
	cs.change, err = cs.readChange()
	return err
}

func (cs *ChunkStore) MakeJournalIter(branch string) (JournalIter, error) {
	src, err := cs.backing.Open("journals/" + branch)
	if err != nil {
		return nil, err
	}

	return newCsJournalIter(src, cs)
}
