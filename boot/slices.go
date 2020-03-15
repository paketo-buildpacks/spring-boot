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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildpacks/libcnb"
)

func ConventionSlices(root string, classes string, libs string) ([]libcnb.Slice, error) {
	var slices []libcnb.Slice

	slice := libcnb.Slice{}

	if err := filepath.Walk(libs, files(root, func(path string) {
		if !strings.Contains(path, "SNAPSHOT") {
			slice.Paths = append(slice.Paths, path)
		}
	})); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to walk %s\n%w", libs, err)
	}

	if len(slice.Paths) > 0 {
		slices = append(slices, slice)
	}

	slice = libcnb.Slice{}

	if err := filepath.Walk(libs, files(root, func(path string) {
		if strings.Contains(path, "SNAPSHOT") {
			slice.Paths = append(slice.Paths, path)
		}
	})); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to walk %s\n%w", libs, err)
	}

	if len(slice.Paths) > 0 {
		slices = append(slices, slice)
	}

	slice = libcnb.Slice{}

	for _, f := range []string{filepath.Join("META-INF", "resources"), "resources", "static", "public"} {
		file := filepath.Join(root, f)
		if err := filepath.Walk(file, files(root, func(path string) {
			slice.Paths = append(slice.Paths, path)
		})); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("unable to walk %s\n%w", file, err)
		}
	}

	if len(slice.Paths) > 0 {
		slices = append(slices, slice)
	}

	slice = libcnb.Slice{}

	if err := filepath.Walk(classes, files(root, func(path string) {
		slice.Paths = append(slice.Paths, path)
	})); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to walk %s\n%w", classes, err)
	}

	if len(slice.Paths) > 0 {
		slices = append(slices, slice)
	}

	return slices, nil
}

func IndexSlices(root string, layers ...string) ([]libcnb.Slice, error) {
	var slices []libcnb.Slice

	for _, layer := range layers {
		slice := libcnb.Slice{}

		if err := filepath.Walk(layer, files(root, func(path string) {
			slice.Paths = append(slice.Paths, path)
		})); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("unable to walk %s\n%w", layer, err)
		}

		if len(slice.Paths) > 0 {
			slices = append(slices, slice)
		}
	}

	return slices, nil
}

func files(root string, f func(path string)) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("unable to generate relative path from %s to %s", root, path)
		}

		f(rel)
		return nil
	}
}
