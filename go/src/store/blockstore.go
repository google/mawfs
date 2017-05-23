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
	"io"
	//    "crypto/aes"
	"crypto/sha256"
	"github.com/jacobsa/crypto/siv"
	//"fmt"
	pb "mawfs"
	"github.com/golang/protobuf/proto"
)

const BlockSize = 65536

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
}

type NodeStore interface {
    // Store a Node, return its digest.
    StoreNode(node pb.Node) (string, error)

    // Compute and return the digest for the node.  This is just like
    // storeNode() only it doesn't store the node, it just creates its digest.
    makeDigest(node pb.Node) string

    // Get the node at the given digest, null if it's not currently stored.
    getNode(digest []byte) (pb.Node, error)

    // Stores a Commit, returns its digest.
    storeCommit(commit pb.Commit) string;

    // Retrieves a commit object from its digest, an error if it's not
    // currently stored.
    getCommit(digest []byte) (pb.Commit, error)

//    # Get the root node (null if there is no root).
//    @abstract Node getRoot();
//
//    # Get the root digest (null if there is no root).
//    @abstract String getRootDigest();
//
//    ## Store the digest of the root node.
//    @abstract void storeRoot(String digest);
//
//    ## Returns the digest of the head commit of a given branch.   Returns
//    ## null if the branch is not defined.
//    @abstract String getHead(String branch);
//
//    ## Sets the digest of the head commit for the branch.
//    @abstract void setHead(String branch, String digest);
//
//    ## Write a change to the journal for the branch.  Returns the digest of
//    ## the change.
//    @abstract String writeToJournal(String branch, Change change);
//
//    ## Delete the journal for a branch.
//    @abstract void deleteJournal(String branch);
//
//    ## Return an iterator over the journal
//    @abstract JournalIter makeJournalIter(String branch);
//
//    ## Returns the size of the journal, or rather, the size of all of the
//    ## changes in it.
//    @abstract uint getJournalSize(String branch);
}

// Wraps a filesystem in an interface to improve testability.
type FileSys interface {
    Create(name string) (io.Writer, error)
    Open(name string) (io.Reader, error)
}

// NodeStore implementation that writes to a backing filesystem directory.
type ChunkStore struct {
    fsInfo *FSInfo
    backing FileSys
}

func NewChunkStore(fsInfo *FSInfo, backing FileSys) (*ChunkStore) {
    return &ChunkStore{fsInfo, backing}
}

// Stores chunk data, returns the digest.
func (cs *ChunkStore) store(data []byte) ([]byte, error) {
    buf := bytes.Buffer{}
    digest, err := cs.fsInfo.WriteChunk(&buf, data)
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

func (cs *ChunkStore) StoreNode(node *pb.Node) ([]byte, error) {
    rep, err := proto.Marshal(node)
    if err != nil {
        return nil, err
    }
	return cs.store(rep)
}

func (cs *ChunkStore) LoadNode(digest []byte) (*pb.Node, error) {
    src, err := cs.backing.Open(altEncode(digest))
    if err != nil {
        return nil, err
    }

    chunk, err := cs.fsInfo.ReadChunk(src)
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
