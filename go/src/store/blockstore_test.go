package blockstore

import (
	"bytes"
	"crypto/sha256"
	"io"
	"testing"
	pb "mawfs"
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

type FakeFileSys struct {
    contents map[string]*bytes.Buffer
}

func NewFakeFileSys() (*FakeFileSys) {
    return &FakeFileSys{make(map[string]*bytes.Buffer)}
}

func (fs *FakeFileSys) Create(name string) (io.Writer, error) {
    result := &bytes.Buffer{}
    fs.contents[name] = result
    return result, nil
}

func (fs *FakeFileSys) Open(name string) (io.Reader, error) {
    return fs.contents[name], nil
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
    if newNode.GetChecksum() != node.GetChecksum() ||
    	  newNode.GetContents() != node.GetContents() {
        t.Error("Node contents was not preseerved")
        t.Fail()
    }
}
