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

package downloader

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/tetrafolium/luci-go/common/data/caching/cache"
	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/isolated"
	"github.com/tetrafolium/luci-go/common/isolatedclient"
	"github.com/tetrafolium/luci-go/common/isolatedclient/isolatedfake"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNormalizePathSeparator(t *testing.T) {
	t.Parallel()

	Convey("Check path normalization", t, func() {
		Convey("posix path", func() {
			So(normalizePathSeparator("a/b"), ShouldEqual, filepath.Join("a", "b"))
		})

		Convey("windows path", func() {
			So(normalizePathSeparator(`a\b`), ShouldEqual, filepath.Join("a", "b"))
		})
	})
}

func TestDownloaderFetchIsolated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	data1 := []byte("hello world!")
	data2 := []byte("wat")
	tardata := genTar(t)

	namespace := isolatedclient.DefaultNamespace
	h := isolated.GetHash(namespace)
	server := isolatedfake.New()
	data1hash := server.Inject(namespace, data1)
	data2hash := server.Inject(namespace, data2)
	tardatahash := server.Inject(namespace, tardata)
	tardataname := fmt.Sprintf("%s.tar", tardatahash)

	onePath := filepath.Join("foo", "one.txt")
	twoPath := filepath.Join("foo", "two.txt")
	posixPath := "posix/path"
	winPath := `win\path`
	isolated1 := isolated.New(h)
	isolated1.Files = map[string]isolated.File{
		onePath:     isolated.BasicFile(data1hash, 0664, int64(len(data1))),
		twoPath:     isolated.BasicFile(data2hash, 0764, int64(len(data2))),
		posixPath:   isolated.BasicFile(data1hash, 0664, int64(len(data1))),
		winPath:     isolated.BasicFile(data2hash, 0664, int64(len(data2))),
		tardataname: isolated.TarFile(tardatahash, int64(len(tardata))),
	}
	isolated1bytes, _ := json.Marshal(&isolated1)
	isolated1hash := server.Inject(namespace, isolated1bytes)

	lolPath := filepath.Join("bar", "lol.txt")
	oloPath := filepath.Join("foo", "boz", "olo.txt")
	isolated2 := isolated.New(h)
	isolated2.Files = map[string]isolated.File{
		lolPath: isolated.BasicFile(data1hash, 0664, int64(len(data1))),
		oloPath: isolated.BasicFile(data2hash, 0664, int64(len(data2))),
	}
	isolatedFiles := stringset.NewFromSlice([]string{
		tardataname,
		onePath,
		twoPath,
		normalizePathSeparator(posixPath),
		normalizePathSeparator(winPath),
		lolPath,
		oloPath,
		// In tardata
		"file1",
		"file2",
		filepath.Join("tar", "posix", "path"),
		filepath.Join("tar", "win", "path"),
	}...)
	blahPath := "blah.txt"

	// Symlinks not supported on Windows.
	if runtime.GOOS != "windows" {
		isolated2.Files[blahPath] = isolated.SymLink(oloPath)
		isolatedFiles.Add(blahPath)
	}
	isolated2.Includes = isolated.HexDigests{isolated1hash}
	isolated2bytes, _ := json.Marshal(&isolated2)
	isolated2hash := server.Inject(namespace, isolated2bytes)

	ts := httptest.NewServer(server)
	defer ts.Close()
	client := isolatedclient.NewClient(ts.URL, isolatedclient.WithNamespace(namespace))

	Convey(`A downloader should be able to download the isolated.`, t, func() {
		tmpDir, err := ioutil.TempDir("", "isolated")
		So(err, ShouldBeNil)
		defer func() {
			So(os.RemoveAll(tmpDir), ShouldBeNil)
		}()

		mu := sync.Mutex{}
		var files []string
		d := New(ctx, client, isolated2hash, tmpDir, &Options{
			FileCallback: func(name string, _ *isolated.File) {
				mu.Lock()
				files = append(files, name)
				mu.Unlock()
			},
		})
		So(d.Wait(), ShouldBeNil)
		So(stringset.NewFromSlice(files...), ShouldResemble, isolatedFiles)

		b, err := ioutil.ReadFile(filepath.Join(tmpDir, onePath))
		So(err, ShouldBeNil)
		So(b, ShouldResemble, data1)

		fi, err := os.Stat(filepath.Join(tmpDir, onePath))
		So(err, ShouldBeNil)
		// to ignore effect of umask, only check executable bit.
		So(fi.Mode()&0111, ShouldEqual, 0)

		b, err = ioutil.ReadFile(filepath.Join(tmpDir, twoPath))
		So(err, ShouldBeNil)
		So(b, ShouldResemble, data2)

		fi, err = os.Stat(filepath.Join(tmpDir, twoPath))
		So(err, ShouldBeNil)
		// to ignore effect of umask, only check executable bit.
		So(fi.Mode()&0011, ShouldEqual, 0)

		b, err = ioutil.ReadFile(filepath.Join(tmpDir, lolPath))
		So(err, ShouldBeNil)
		So(b, ShouldResemble, data1)

		b, err = ioutil.ReadFile(filepath.Join(tmpDir, oloPath))
		So(err, ShouldBeNil)
		So(b, ShouldResemble, data2)

		// Check files in tar archive
		_, err = os.Stat(filepath.Join(tmpDir, "file1"))
		So(err, ShouldBeNil)

		_, err = os.Stat(filepath.Join(tmpDir, "file2"))
		So(err, ShouldBeNil)

		_, err = os.Stat(filepath.Join(tmpDir, "tar", "posix", "path"))
		So(err, ShouldBeNil)

		_, err = os.Stat(filepath.Join(tmpDir, "tar", "win", "path"))
		So(err, ShouldBeNil)

		_, err = ioutil.ReadFile(filepath.Join(tmpDir, tardataname))
		So(os.IsNotExist(err), ShouldBeTrue)

		if runtime.GOOS != "windows" {
			l, err := os.Readlink(filepath.Join(tmpDir, blahPath))
			So(err, ShouldBeNil)
			So(l, ShouldResemble, oloPath)
		}
	})
}

// genTar returns a valid tar file.
func genTar(t *testing.T) []byte {
	b := bytes.Buffer{}
	tw := tar.NewWriter(&b)
	d := []byte("hello file1")
	if err := tw.WriteHeader(&tar.Header{Name: "file1", Mode: 0644, Typeflag: tar.TypeReg, Size: int64(len(d))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(d); err != nil {
		t.Fatal(err)
	}
	d = []byte(strings.Repeat("hello file2", 100))
	if err := tw.WriteHeader(&tar.Header{Name: "file2", Mode: 0644, Typeflag: tar.TypeReg, Size: int64(len(d))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(d); err != nil {
		t.Fatal(err)
	}
	d = []byte("posixpath")
	if err := tw.WriteHeader(&tar.Header{Name: "tar/posix/path", Mode: 0644, Typeflag: tar.TypeReg, Size: int64(len(d))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(d); err != nil {
		t.Fatal(err)
	}
	d = []byte("winpath")
	if err := tw.WriteHeader(&tar.Header{Name: `tar\win\path`, Mode: 0644, Typeflag: tar.TypeReg, Size: int64(len(d))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(d); err != nil {
		t.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestDownloaderWithCache(t *testing.T) {
	t.Parallel()
	Convey(`TestDownloaderWithCache`, t, func() {
		tmpDir := t.TempDir()
		ctx := context.Background()

		miss := []byte("cache miss")
		hit := []byte("cache hit")
		namespace := isolatedclient.DefaultNamespace
		h := isolated.GetHash(namespace)
		server := isolatedfake.New()
		misshash := server.Inject(namespace, miss)
		hithash := isolated.HashBytes(isolated.GetHash(namespace), hit)

		missPath := filepath.Join("foo", "miss.txt")
		hitPath := filepath.Join("foo", "hit.txt")
		isolated1 := isolated.New(h)

		isolated1.Files = map[string]isolated.File{
			missPath: isolated.BasicFile(misshash, 0664, int64(len(miss))),
			hitPath:  isolated.BasicFile(hithash, 0664, int64(len(hit))),
		}
		isolated1bytes, _ := json.Marshal(&isolated1)
		isolated1hash := server.Inject(namespace, isolated1bytes)

		isolatedFiles := stringset.NewFromSlice([]string{
			missPath,
			hitPath,
		}...)

		ts := httptest.NewServer(server)
		defer ts.Close()
		client := isolatedclient.NewClient(ts.URL, isolatedclient.WithNamespace(namespace))

		mu := sync.Mutex{}
		var files []string
		policy := cache.Policies{
			MaxSize:  1024,
			MaxItems: 1024,
		}

		cacheDir := filepath.Join(tmpDir, "cache")
		So(os.MkdirAll(cacheDir, os.ModePerm), ShouldBeNil)
		cacheObj, err := cache.New(policy, cacheDir, h)
		So(err, ShouldBeNil)

		So(cacheObj.Add(hithash, bytes.NewReader(hit)), ShouldBeNil)

		d := New(ctx, client, isolated1hash, tmpDir, &Options{
			FileCallback: func(name string, _ *isolated.File) {
				mu.Lock()
				files = append(files, name)
				mu.Unlock()
			},
			Cache: cacheObj,
		})
		So(d.Wait(), ShouldBeNil)
		So(stringset.NewFromSlice(files...), ShouldResemble, isolatedFiles)

		So(cacheObj.Touch(misshash), ShouldBeTrue)
	})
}

func TestFetchAndMap(t *testing.T) {
	t.Parallel()

	Convey(`TestFetchAndMap`, t, func() {
		tmpdir := t.TempDir()
		ctx := context.Background()
		isolatedDir := filepath.Join(tmpdir, "isolated")
		So(os.Mkdir(isolatedDir, 0o700), ShouldBeNil)

		data1 := []byte("hello world!")

		namespace := isolatedclient.DefaultNamespace
		h := isolated.GetHash(namespace)
		server := isolatedfake.New()
		data1hash := server.Inject(namespace, data1)

		onePath := filepath.Join("foo", "one.txt")
		onePathFile := isolated.BasicFile(data1hash, 0664, int64(len(data1)))
		isolated1 := isolated.New(h)

		tardata := genTar(t)
		tarhash := server.Inject(namespace, tardata)

		isolated1.Files = map[string]isolated.File{
			onePath:   onePathFile,
			"tar.tar": isolated.TarFile(tarhash, int64(len(tardata))),
		}
		isolated1bytes, _ := json.Marshal(&isolated1)
		isolated1hash := server.Inject(namespace, isolated1bytes)

		ts := httptest.NewServer(server)
		defer ts.Close()
		client := isolatedclient.NewClient(ts.URL, isolatedclient.WithNamespace(namespace))

		policy := cache.Policies{
			MaxSize:  1024 * 1024,
			MaxItems: 1024,
		}
		diskcache, err := cache.New(policy, filepath.Join(tmpdir, "cache"), h)
		So(err, ShouldBeNil)
		defer func() {
			So(diskcache.Close(), ShouldBeNil)
		}()

		isomap, stats, err := FetchAndMap(ctx, isolated1hash, client, diskcache, isolatedDir)
		So(err, ShouldBeNil)
		So(isomap, ShouldResemble, &isolated.Isolated{
			Algo:    "sha-1",
			Version: isolated.IsolatedFormatVersion,
		})

		So(stats.Duration, ShouldBeGreaterThan, 0)
		So(stats.ItemsCold, ShouldResemble, []byte{120, 156, 226, 249, 162, 15, 8, 0, 0, 255, 255, 2, 62, 1, 48})
		So(stats.ItemsHot, ShouldResemble, []byte(nil))

		buf, err := ioutil.ReadFile(filepath.Join(isolatedDir, onePath))
		So(err, ShouldBeNil)
		So(buf, ShouldResemble, data1)

		So(diskcache.Touch(tarhash), ShouldBeTrue)
	})
}
