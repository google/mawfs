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
	"testing"
)

// Implements File.
type bufferFile struct {
	bytes.Buffer
}

func (*bufferFile) Close() error {
	return nil
}

// Implements FileSys.
type FakeFileSys struct {
	contents map[string]*bufferFile
}

func NewFakeFileSys() *FakeFileSys {
	return &FakeFileSys{make(map[string]*bufferFile)}
}

func (fs *FakeFileSys) Create(name string) (File, error) {
	result := &bufferFile{}
	fs.contents[name] = result
	return result, nil
}

func (fs *FakeFileSys) Open(name string) (File, error) {
	return fs.contents[name], nil
}

func (fs *FakeFileSys) Exists(name string) bool {
	_, ok := fs.contents[name]
	return ok
}

func (fs *FakeFileSys) Append(name string) (File, error) {
	if _, exists := fs.contents[name]; !exists {
		fs.contents[name] = &bufferFile{}
	}
	return fs.contents[name], nil
}

func (fs *FakeFileSys) Mkdir(name string) error {
	return nil
}

func (fs *FakeFileSys) Remove(name string) error {
	delete(fs.contents, "journals/"+name)
	return nil
}

func Assertf(t *testing.T, cond bool, message string, v ...interface{}) {
	if !cond {
		t.Errorf(message, v...)
	}
}

func Assert(t *testing.T, cond bool) {
    if !cond {
        t.Fail();
    }
}
