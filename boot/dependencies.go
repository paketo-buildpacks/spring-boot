/*
 * Copyright 2018-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package boot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
)

var maven = regexp.MustCompile(".+/(.*)-([\\d].*)\\.jar")

type Dependency struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	SHA256  string `toml:"sha256"`
}

func NewDependency(path string) (Dependency, error) {
	d := Dependency{
		Name:    filepath.Base(path),
		Version: "unknown",
	}

	m := maven.FindStringSubmatch(path)
	if m != nil {
		d.Name = m[1]
		d.Version = m[2]
	}

	in, err := os.Open(path)
	if err != nil {
		return Dependency{}, fmt.Errorf("unable to open %s\n%w", path, err)
	}
	defer in.Close()

	s := sha256.New()

	_, err = io.Copy(s, in)
	if err != nil {
		return Dependency{}, fmt.Errorf("unable to calculate sha256 for %s\n%w", path, err)
	}

	d.SHA256 = hex.EncodeToString(s.Sum(nil))

	return d, nil
}

type result struct {
	err   error
	value Dependency
}

func Dependencies(directories ...string) ([]Dependency, error) {
	ch := make(chan result)
	var wg sync.WaitGroup

	for _, dir := range directories {
		if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			wg.Add(1)
			go func() {
				defer wg.Done()

				r := result{}
				r.value, r.err = NewDependency(path)
				ch <- r
			}()

			return nil
		}); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("unable to walk %s\n%w", dir, err)
		}
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var d []Dependency
	for r := range ch {
		if r.err != nil {
			return nil, r.err
		}

		d = append(d, r.value)
	}
	sort.Slice(d, func(i, j int) bool {
		return d[i].Name < d[j].Name
	})

	return d, nil
}
