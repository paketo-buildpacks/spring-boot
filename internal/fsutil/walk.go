package fsutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

func Walk(root string, walkFn filepath.WalkFunc) error {
	type node struct {
		next  *node
		value string
	}
	var (
		head = &node{value: root}
		tail = head
	)
	var err error
	for ; head != nil; head = head.next {
		p := head.value
		var fi fs.FileInfo
		fi, err = os.Lstat(p)
		if err != nil || !fi.IsDir() {
			err = walkFn(p, fi, err)
			if err != nil {
				break
			}
			continue
		}
		var names []string
		names, err = readDirNames(p)
		err = walkFn(p, fi, err)
		if err != nil {
			if errors.Is(err, filepath.SkipDir) {
				continue
			}
			break
		}
		for _, name := range names {
			tail.next = &node{value: filepath.Join(p, name)}
			tail = tail.next
		}
	}
	if errors.Is(err, filepath.SkipAll) {
		return nil
	}
	return err
}

func readDirNames(dirname string) ([]string, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}
