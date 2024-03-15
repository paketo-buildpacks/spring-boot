package fsutil_test

import (
	"errors"
	"fmt"
	"github.com/paketo-buildpacks/spring-boot/v5/internal/fsutil"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

type fileInfo struct {
	Path string
	Type fs.FileMode
}

const (
	filename      = "a.txt"
	contentA1A2   = "a1/a2/" + filename
	contentC1A2A3 = "c1/a2/a3/" + filename
)

var (
	errSentinel = errors.New("sentinel")
)

func testWalk(t *testing.T, context spec.G, it spec.S) {
	var (
		root   string
		Expect = NewWithT(t).Expect
	)

	it.Before(func() {
		root = createTestDir(t)
	})

	context("compare to stdlib Walk", func() {

		var createWalkFn func(out *[]fileInfo) filepath.WalkFunc = func(out *[]fileInfo) filepath.WalkFunc {
			return func(path string, fi fs.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				var typ fs.FileMode
				if fi != nil {
					typ = fi.Mode().Type()
				}
				*out = append(*out, fileInfo{
					Path: path,
					Type: typ,
				})
				return nil
			}
		}

		it("basic", func() {
			var ourFiles, theirFiles []fileInfo

			errOurs := fsutil.Walk(root, createWalkFn(&ourFiles))
			Expect(errOurs).NotTo(HaveOccurred())

			errTheirs := filepath.Walk(root, createWalkFn(&theirFiles))
			Expect(errTheirs).NotTo(HaveOccurred())

			sortBFSOrder(theirFiles)

			Expect(ourFiles).To(Equal(theirFiles))
		})

		it("non-existent root folder", func() {
			var ourFiles, theirFiles []fileInfo
			root = filepath.Join(t.TempDir(), "nonexistent")

			errOurs := fsutil.Walk(root, createWalkFn(&ourFiles))
			Expect(errOurs).NotTo(HaveOccurred())

			errTheirs := filepath.Walk(root, createWalkFn(&theirFiles))
			Expect(errTheirs).NotTo(HaveOccurred())

			sortBFSOrder(theirFiles)

			Expect(ourFiles).To(Equal(theirFiles))
		})

		it("propagate error", func() {
			var ourFiles, theirFiles []fileInfo
			createWalkFn = func(out *[]fileInfo) filepath.WalkFunc {
				return func(path string, fi fs.FileInfo, err error) error {
					return errSentinel
				}
			}

			errOurs := fsutil.Walk(root, createWalkFn(&ourFiles))
			Expect(errOurs).To(MatchError(errSentinel))

			errTheirs := filepath.Walk(root, createWalkFn(&theirFiles))
			Expect(errTheirs).To(MatchError(errSentinel))

			sortBFSOrder(theirFiles)

			Expect(ourFiles).To(Equal(theirFiles))
		})

	})

	it("finds a file", func() {
		var content string
		err := fsutil.Walk(root, func(path string, fi fs.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if fi.Mode().Type() == 0 {
				_, name := filepath.Split(path)
				if name == filename {
					bs, e := os.ReadFile(path)
					if e != nil {
						return e
					}
					content = string(bs)
					return errSentinel
				}
			}
			return nil
		})

		Expect(err).To(MatchError(errSentinel))
		Expect(content).To(Equal(contentA1A2))
	})

	it("finds a file (skipping a dir)", func() {
		var content string
		err := fsutil.Walk(root, func(path string, fi fs.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			rp, err := filepath.Rel(root, path)
			Expect(err).NotTo(HaveOccurred())
			if rp == "a1" { // skip the first dir containing a.txt
				return filepath.SkipDir
			}
			if fi.Mode().Type() == 0 {
				_, name := filepath.Split(path)
				if name == filename {
					bs, e := os.ReadFile(path)
					if e != nil {
						return e
					}
					content = string(bs)
					return errSentinel
				}
			}
			return nil
		})

		Expect(err).To(MatchError(errSentinel))
		Expect(content).To(Equal(contentC1A2A3))
	})
}

func sortBFSOrder(files []fileInfo) {
	sort.SliceStable(files, func(i, j int) bool {
		iLen := len(strings.Split(files[i].Path, string(filepath.Separator)))
		jLen := len(strings.Split(files[j].Path, string(filepath.Separator)))
		return iLen < jLen
	})

}

// Create this file structure for the tests to run (obtained using `tree`)
// ├── a.lnk -> a1
// ├── a1
// │   ├── a2
// │   │   ├── a.txt
// │   │   ├── a3
// │   │   ├── b3
// │   │   └── c3
// │   ├── b2
// │   │   ├── a3
// │   │   ├── b3
// │   │   └── c3
// │   └── c2
// │       ├── a3
// │       ├── b3
// │       └── c3
// ├── b1
// │   ├── a2
// │   │   ├── a3
// │   │   ├── b3
// │   │   └── c3
// │   ├── b2
// │   │   ├── a3
// │   │   ├── b3
// │   │   └── c3
// │   └── c2
// │       ├── a3
// │       ├── b3
// │       └── c3
// └── c1
//
//	├── a2
//	│   ├── a3
//	│   │   └── a.txt
//	│   ├── b3
//	│   └── c3
//	├── b2
//	│   ├── a3
//	│   ├── b3
//	│   └── c3
//	└── c2
//	    ├── a3
//	    ├── b3
//	    └── c3
func createTestDir(t *testing.T) string {
	root := t.TempDir()
	var err error

	var createChildrenDirs func(dir string, lvl int)
	createChildrenDirs = func(dir string, lvl int) {
		if lvl >= 4 {
			return
		}
		for _, n := range []string{"a", "b", "c"} {
			d := filepath.Join(dir, fmt.Sprintf("%s%d", n, lvl))
			err := os.Mkdir(d, 0755)
			if err != nil {
				t.Fatal(err)
			}
			createChildrenDirs(d, lvl+1)
		}
	}
	createChildrenDirs(root, 1)

	err = os.Chmod(filepath.Join(root, "b1", "a2"), 0000)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err = os.Chmod(filepath.Join(root, "b1", "a2"), 0755)
		if err != nil {
			t.Fatal(err)
		}
	})

	err = os.Symlink("a1", filepath.Join(root, "a.lnk"))
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(root, "a1", "a2", filename), []byte(contentA1A2), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(root, "c1", "a2", "a3", filename), []byte(contentC1A2A3), 0644)
	if err != nil {
		t.Fatal(err)
	}
	return root
}
