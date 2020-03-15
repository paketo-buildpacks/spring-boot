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

package boot_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/spring-boot/boot"
	"github.com/sclevine/spec"
)

func testSlices(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path string
	)

	it.Before(func() {
		var err error

		path, err = ioutil.TempDir("", "slices")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("generates duck slices", func() {
		Expect(os.MkdirAll(filepath.Join(path, "libs"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "libs", "test-file"), []byte{}, 0644)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "libs", "test-file-SNAPSHOT"), []byte{}, 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(path, "META-INF", "resources"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "resources", "test-file"), []byte{}, 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(path, "resources"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "resources", "test-file"), []byte{}, 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(path, "static"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "static", "test-file"), []byte{}, 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(path, "public"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "public", "test-file"), []byte{}, 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(path, "classes"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "classes", "test-file"), []byte{}, 0644)).To(Succeed())

		Expect(boot.ConventionSlices(path,
			filepath.Join(path, "classes"),
			filepath.Join(path, "libs"))).To(Equal([]libcnb.Slice{
			{Paths: []string{filepath.Join("libs", "test-file")}},
			{Paths: []string{filepath.Join("libs", "test-file-SNAPSHOT")}},
			{Paths: []string{
				filepath.Join("META-INF", "resources", "test-file"),
				filepath.Join("resources", "test-file"),
				filepath.Join("static", "test-file"),
				filepath.Join("public", "test-file"),
			}},
			{Paths: []string{filepath.Join("classes", "test-file")}},
		}))
	})

	it("generates index slices", func() {
		Expect(os.MkdirAll(filepath.Join(path, "layer-1"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "layer-1", "test-file"), []byte{}, 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(path, "layer-2"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "layer-2", "test-file"), []byte{}, 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(path, "META-INF", "resources"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "resources", "test-file"), []byte{}, 0644)).To(Succeed())

		Expect(boot.IndexSlices(path,
			filepath.Join(path, "layer-1"),
			filepath.Join(path, "layer-2"),
			filepath.Join(path, "layer-3"))).To(Equal([]libcnb.Slice{
			{Paths: []string{filepath.Join("layer-1", "test-file")}},
			{Paths: []string{filepath.Join("layer-2", "test-file")}},
		}))
	})

}
