// Copyright 2017 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package archiver

import (
	"bytes"
	"crypto"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	humanize "github.com/dustin/go-humanize"
	"github.com/tetrafolium/luci-go/common/isolated"
	"github.com/tetrafolium/luci-go/common/isolatedclient"
	"github.com/tetrafolium/luci-go/common/system/filesystem"
)

const (
	// archiveMaxSize is the maximum size of the created archives.
	archiveMaxSize = 10e6
)

// limitedOS contains a subset of the functions from the os package.
type limitedOS interface {
	Readlink(string) (string, error)

	// Open is like os.Open, but returns an io.ReadCloser since
	// that's all we need and it's easier to implement with a fake.
	Open(string) (io.ReadCloser, error)

	openFiler
}

type openFiler interface {
	// OpenFile is like os.OpenFile, but returns an io.WriteCloser since
	// that's all we need and it's easier to implement with a fake.
	OpenFile(string, int, os.FileMode) (io.WriteCloser, error)
}

// standardOS implements limitedOS by delegating to the standard library's os package.
type standardOS struct{}

func (sos standardOS) Readlink(name string) (string, error) {
	return os.Readlink(name)
}

func (sos standardOS) Open(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

func (sos standardOS) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	return os.OpenFile(name, flag, perm)
}

// hashResult stores the result of an isolated.Hash call.
type hashResult struct {
	digest isolated.HexDigest
	err    error
}

// UploadTracker uploads and keeps track of files.
type UploadTracker struct {
	checker  Checker
	uploader Uploader
	isol     *isolated.Isolated

	// A cache of file hashes, to speed up future requests for a hash of the same file.
	fileHashCache *sync.Map

	// Override for testing.
	lOS            limitedOS
	doHashFileImpl func(*UploadTracker, string) (isolated.HexDigest, error)
}

// newUploadTracker constructs an UploadTracker.  It tracks uploaded files in isol.Files.
func newUploadTracker(checker Checker, uploader Uploader, isol *isolated.Isolated, fileHashCache *sync.Map) *UploadTracker {
	isol.Files = make(map[string]isolated.File)
	return &UploadTracker{
		checker:        checker,
		uploader:       uploader,
		isol:           isol,
		fileHashCache:  fileHashCache,
		lOS:            standardOS{},
		doHashFileImpl: doHashFile,
	}
}

// UploadDeps uploads all of the items in parts.
func (ut *UploadTracker) UploadDeps(parts partitionedDeps) error {
	if err := ut.populateSymlinks(parts.links.items); err != nil {
		return err
	}

	if err := ut.tarAndUploadFiles(parts.filesToArchive.items); err != nil {
		return err
	}

	if err := ut.uploadFiles(parts.indivFiles.items); err != nil {
		return err
	}
	return nil
}

// Files returns the files which have been uploaded.
// Note: files may not have completed uploading until the tracker's Checker and
// Uploader have been closed.
func (ut *UploadTracker) Files() map[string]isolated.File {
	return ut.isol.Files
}

// populateSymlinks adds an isolated.File to files for each provided symlink
func (ut *UploadTracker) populateSymlinks(symlinks []*Item) error {
	for _, item := range symlinks {
		l, err := ut.lOS.Readlink(item.Path)
		if err != nil {
			return fmt.Errorf("unable to resolve symlink for %q: %v", item.Path, err)
		}
		ut.isol.Files[item.RelPath] = isolated.SymLink(l)
	}
	return nil
}

// tarAndUploadFiles creates bundles of files, uploads them, and adds each bundle to files.
func (ut *UploadTracker) tarAndUploadFiles(smallFiles []*Item) error {
	bundles := shardItems(smallFiles, archiveMaxSize)
	log.Printf("\t%d TAR archives to be isolated", len(bundles))

	for _, bundle := range bundles {
		// TODO(crbug/854610): Do not create a tarfile when it contains a low
		// number of files, it should upload the file(s) directly instead.
		bundle := bundle
		digest, tarSize, err := bundle.Digest(ut.checker.Hash())
		if err != nil {
			return err
		}

		log.Printf("Created tar archive %q (%s)", digest, humanize.Bytes(uint64(tarSize)))
		log.Printf("\tcontains %d files (total %s)", len(bundle.items), humanize.Bytes(uint64(bundle.itemSize)))
		// Mint an item for this tar.
		item := &Item{
			Path:    fmt.Sprintf(".%s.tar", digest),
			RelPath: fmt.Sprintf(".%s.tar", digest),
			Size:    tarSize,
			Digest:  digest,
		}
		ut.isol.Files[item.RelPath] = isolated.TarFile(item.Digest, item.Size)

		ut.checker.AddItem(item, false, func(item *Item, ps *isolatedclient.PushState) {
			if ps == nil {
				return
			}

			// We are about to upload item. If we fail, the entire isolate operation will fail.
			// If we succeed, then item will be on the server by the time that the isolate operation
			// completes. So it is safe for subsequent checks to assume that the item exists.
			ut.checker.PresumeExists(item)

			log.Printf("QUEUED %q for upload", item.RelPath)
			ut.uploader.Upload(item.RelPath, bundle.Contents, ps, func() {
				log.Printf("UPLOADED %q", item.RelPath)
			})
		})
	}
	return nil
}

// uploadFiles uploads each file and adds it to files.
func (ut *UploadTracker) uploadFiles(files []*Item) error {
	// Handle the large individually-uploaded files.
	for _, item := range files {
		d, err := ut.hashFile(item.Path)
		if err != nil {
			return err
		}
		item.Digest = d
		ut.isol.Files[item.RelPath] = isolated.BasicFile(item.Digest, int(item.Mode), item.Size)
		ut.checker.AddItem(item, false, func(item *Item, ps *isolatedclient.PushState) {
			if ps == nil {
				return
			}

			// We are about to upload item. If we fail, the entire isolate operation will fail.
			// If we succeed, then item will be on the server by the time that the isolate operation
			// completes. So it is safe for subsequent checks to assume that the item exists.
			ut.checker.PresumeExists(item)

			log.Printf("QUEUED %q for upload", item.RelPath)
			ut.uploader.UploadFile(item, ps, func() {
				log.Printf("UPLOADED %q", item.RelPath)
			})
		})
	}
	return nil
}

// IsolatedSummary contains an isolate name and its digest.
type IsolatedSummary struct {
	// Name is the base name an isolated file with any extension stripped
	Name   string
	Digest isolated.HexDigest
}

// Finalize creates and uploads the isolate JSON at the isolatePath, and closes the checker and uploader.
// It returns the isolate name and digest.
// Finalize should only be called after UploadDeps.
func (ut *UploadTracker) Finalize(isolatedPath string) (IsolatedSummary, error) {
	isolFile, err := newIsolatedFile(ut.isol, isolatedPath)
	if err != nil {
		return IsolatedSummary{}, err
	}

	// Check and upload isolate JSON.
	ut.checker.AddItem(isolFile.item(ut.checker.Hash()), true, func(item *Item, ps *isolatedclient.PushState) {
		if ps == nil {
			return
		}
		log.Printf("QUEUED %q for upload", item.RelPath)
		ut.uploader.UploadBytes(item.RelPath, isolFile.contents(), ps, func() {
			log.Printf("UPLOADED %q", item.RelPath)
		})
	})

	// Write the isolated file...
	if isolFile.path != "" {
		if err := isolFile.writeJSONFile(ut.lOS); err != nil {
			return IsolatedSummary{}, err
		}
	}

	return IsolatedSummary{
		Name:   isolFile.name(),
		Digest: isolFile.item(ut.checker.Hash()).Digest,
	}, nil
}

// hashFile returns the hash of the contents of path, memoizing its results.
func (ut *UploadTracker) hashFile(path string) (isolated.HexDigest, error) {
	result, ok := ut.fileHashCache.Load(path)

	if ok {
		castedResult := result.(hashResult)
		return castedResult.digest, castedResult.err
	}

	digest, err := ut.doHashFileImpl(ut, path)
	ut.fileHashCache.Store(path, hashResult{digest, err})
	return digest, err
}

// doHashFile returns the hash of the contents of path. This should not be
// called directly; call hashFile instead.
func doHashFile(ut *UploadTracker, path string) (isolated.HexDigest, error) {
	f, err := ut.lOS.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return isolated.Hash(ut.checker.Hash(), f)
}

// isolatedFile is an isolated file which is stored in memory.
// It can produce a corresponding Item, for upload to the server,
// and also write its contents to the filesystem.
type isolatedFile struct {
	json []byte
	path string
}

func newIsolatedFile(isol *isolated.Isolated, path string) (*isolatedFile, error) {
	j, err := json.Marshal(isol)
	if err != nil {
		return &isolatedFile{}, err
	}
	return &isolatedFile{json: j, path: path}, nil
}

// item creates an *Item to represent the isolated JSON file.
func (ij *isolatedFile) item(h crypto.Hash) *Item {
	return &Item{
		Path:    ij.path,
		RelPath: filepath.Base(ij.path),
		Digest:  isolated.HashBytes(h, ij.json),
		Size:    int64(len(ij.json)),
	}
}

func (ij *isolatedFile) contents() []byte {
	return ij.json
}

// writeJSONFile writes the file contents to the filesystem.
func (ij *isolatedFile) writeJSONFile(opener openFiler) error {
	f, err := opener.OpenFile(ij.path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, bytes.NewBuffer(ij.json))
	err2 := f.Close()
	if err != nil {
		return err
	}
	return err2
}

// name returns the base name of the isolated file, extension stripped.
func (ij *isolatedFile) name() string {
	return filesystem.GetFilenameNoExt(ij.path)
}
