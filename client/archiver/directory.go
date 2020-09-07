// Copyright 2015 The LUCI Authors.
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
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tetrafolium/luci-go/client/internal/common"
	"github.com/tetrafolium/luci-go/common/errors"
)

// WalkItem represents a file encountered in the (symlink-following) walk of a directory.
type walkItem struct {
	// Absolute path to the item.
	fullPath string

	// Relativie path to the item within the directory.
	relPath string

	// FileInfo of the item.
	info os.FileInfo

	// Whether the item is symlinked and has a source within the directory.
	inTreeSymlink bool
}

// computeRelDestOfInTreeSymlink handles the specific case where an in-tree symlink
// points to an absolute path. In this case, we need to convert that destination to
// a relative path.
//
// Precondition: |evaledDir| is a prefix of |evaledSymPath| (in-tree).
func computeRelDestOfInTreeSymlink(symlinkRelPath, evaledSymPath, evaledDir string) (string, error) {
	// It could be that the logical absolute path pointed by |symlinkPath| is different from
	// what is returned by os.EvalSymlinks (E.g. macOS's "/tmp" vs  "/private/tmp"). So we use
	// |evaledSymPath| to figure out the relative portion within the directory tree. For example:
	//
	// /tmp/ -> private/tmp/
	// `- base/
	//    |- first/
	//    |  `- foo
	//    `- second/
	//       `- bar -> /tmp/base/first/foo
	//
	// |evaledDir|: "/private/tmp/base"
	// |evaledSymPath|: "/private/tmp/base/first/foo"
	//
	// In this case, filepath.Rel(evaledDir, evaledSymPath) = "first/foo"

	// Ideally we should panic on error, because the precondition enforces it.
	relDstInDir, err := filepath.Rel(evaledDir, evaledSymPath)
	if err != nil {
		return "", errors.Annotate(err, "failed to compute relDstInDir: Rel(%s, %s)", evaledDir, evaledSymPath).Err()
	}
	symlinkDir := filepath.Dir(symlinkRelPath)
	relDstToSym, err := filepath.Rel(symlinkDir, relDstInDir)
	if err != nil {
		return "", errors.Annotate(err, "failed to compute relDstToPath: Rel(%s, %s)", symlinkDir, relDstInDir).Err()
	}
	return relDstToSym, nil
}

// walk enumerates a directory tree synchronously and sends the items to
// channel c.
func walk(root string, fsView common.FilesystemView, c chan<- *walkItem) {
	// TODO(maruel): Walk() sorts the file names list, which is not needed here
	// and slows things down. Options:
	// #1 Use os.File.Readdir() directly. It's in the stdlib and works fine, but
	//    it's not the most efficient implementation. On posix it does a lstat()
	//    call, on Windows it does a Win32FileAttributeData.
	// #2 Use raw syscalls.
	//   - On POSIX, use syscall.ReadDirent(). See src/os/dir_unix.go.
	//   - On Windows, use syscall.FindFirstFile(), syscall.FindNextFile(),
	//     syscall.FindClose() directly. See src/os/file_windows.go. For odd
	//     reasons, Windows does not have a batched version to reduce the number
	//     of kernel calls. It's as if they didn't care about performance.
	//
	// In practice, #2 may not be needed, the performance of #1 may be good
	// enough relative to the other performance costs. This needs to be perf
	// tested at 100k+ files scale on Windows and OSX.
	//
	// TODO(maruel): Cache directory enumeration. In particular cases (Chromium),
	// the same directory may be enumerated multiple times. Caching the content
	// may be worth. This needs to be perf tested.
	var walkWithLinks func(string, common.FilesystemView) filepath.WalkFunc

	walkWithLinks = func(dir string, view common.FilesystemView) filepath.WalkFunc {
		return func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Annotate(err, "walk(%q)", path).Err()
			}
			relPath, err := view.RelativePath(path)
			if err != nil {
				return errors.Annotate(err, "walk(%q)", path).Err()
			}
			if relPath == "" { // empty string indicates skip.
				return common.WalkFuncSkipFile(info)
			}

			// Follow symlinks in the tree, treating links that point outside the as if they
			// were ordinary files or directories.
			//
			// The rule about symlinks outside the build tree is for the benefit of use-cases
			// in which not all objects are available in a canonical directory (e.g., when one
			// wishes to isolate build directory artifacts along with dynamically created files).
			mode := info.Mode()
			if mode&os.ModeSymlink == os.ModeSymlink {
				l, err := filepath.EvalSymlinks(path)
				if err != nil {
					return errors.Annotate(err, "EvalSymlinks(%s)", path).Err()
				}
				info, err = os.Stat(l)
				if err != nil {
					return errors.Annotate(err, "Stat(%s)", l).Err()
				}
				// This is super annoying here especially on macOS, as when using
				// TMPDIR, dir could have /var/folders/ while l has
				// /private/var/folders/. Handle this case specifically.
				realDir, err := filepath.EvalSymlinks(dir)
				if err != nil {
					return errors.Annotate(err, "EvalSymlinks(%s)", dir).Err()
				}
				if strings.HasPrefix(l, realDir) {
					// Readlink preserves relative paths of links; necessary to not break in tree symlinks.
					l2, err := os.Readlink(path)
					if err != nil {
						return errors.Annotate(err, "Readlink(%s)", path).Err()
					}

					if !filepath.IsAbs(l2) {
						l = l2
					} else {
						l, err = computeRelDestOfInTreeSymlink(relPath, l, realDir)
						if err != nil {
							return errors.Annotate(err, "failed to computeRelDestOfInTreeSymlink(%s, %s, %s)", relPath, l, realDir).Err()
						}
					}
					c <- &walkItem{fullPath: l, relPath: relPath, info: info, inTreeSymlink: true}
					return nil
				}
				// Found a symlink that pointed out of tree.
				if info.IsDir() {
					linkedView := view.NewSymlinkedView(relPath, l)
					return filepath.Walk(l, walkWithLinks(l, linkedView))
				}
				c <- &walkItem{fullPath: l, relPath: relPath, info: info}
				return nil
			}

			if !info.IsDir() {
				c <- &walkItem{fullPath: path, relPath: relPath, info: info}
			}
			return nil
		}
	}

	if err := filepath.Walk(root, walkWithLinks(root, fsView)); err != nil {
		// No point continuing if an error occurred during walk.
		log.Fatalf("Unable to walk %q: %v", root, err)
	}
}

// PushedDirectoryItem represents a file within the directory being pushed.
type PushedDirectoryItem struct {
	// Absolute path to the item.
	FullPath string

	// Relativie path to the item within the directory.
	RelPath string

	// FileInfo of the item.
	Info os.FileInfo
}

// PushDirectory walks a directory at root and pushes the files in it. relDir
// is a relative directory to offset relative paths against the root directory.
// For symlink that points to another file under the same root (called *in-tree*),
// the symlink is preserved. Otherwise, the symlink is followed (*out-of-tree*),
// and is pushed as an ordinary file.
//
// It does not return until all the pushed items are hashed, which is done
// asynchornously.
//
// The returned files are in a map from its relative path to PushedDirectoryItem.
// The returned in-tree symlinks are in a map from its relative path to the
// relative path it points to.
func PushDirectory(a *Archiver, root, relDir string) (error, map[*PendingItem]PushedDirectoryItem, map[string]string) {
	fileItems := make(map[*PendingItem]PushedDirectoryItem)
	symlinkItems := make(map[string]string)

	fsView, err := common.NewFilesystemView(root, "")
	if err != nil {
		return err, nil, nil
	}

	c := make(chan *walkItem)
	go func() {
		walk(root, fsView, c)
		close(c)
	}()

	for item := range c {
		if relDir != "" {
			item.relPath = filepath.Join(relDir, item.relPath)
		}
		if item.inTreeSymlink {
			symlinkItems[item.relPath] = item.fullPath
		} else {
			fileItems[a.PushFile(item.relPath, item.fullPath, item.info.Size())] = PushedDirectoryItem{item.fullPath, item.relPath, item.info}
		}
	}
	log.Printf("PushDirectoryNoIsolated(%s) files=%d symlinks=%d", root, len(fileItems), len(symlinkItems))

	// Hashing, cache lookups and upload is done asynchronously.
	for pending := range fileItems {
		pending.WaitForHashed()
		if err = pending.Error(); err != nil {
			return err, nil, nil
		}
	}
	return nil, fileItems, symlinkItems
}
