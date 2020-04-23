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
	"github.com/paketo-buildpacks/libjvm"
	"github.com/paketo-buildpacks/spring-boot/boot"
	"github.com/sclevine/spec"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx   libcnb.BuildContext
		build boot.Build
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = ioutil.TempDir("", "build-application")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "META-INF"), 0755)).To(Succeed())
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
	})

	it("does nothing without Spring-Boot-Version", func() {
		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result).To(BeZero())
	})

	it("contributes spring-boot plan entry", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
`), 0644)).To(Succeed())

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Plan.Entries).To(ContainElement(libcnb.BuildpackPlanEntry{
			Name:    "spring-boot",
			Version: "1.1.1",
		}))
	})

	context("dependencies plan entry", func() {

		it("contributes from Spring-Boot-Lib", func() {
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Lib: ALTERNATE/lib
`), 0644)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "ALTERNATE", "lib"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "ALTERNATE", "lib", "test-file-2.2.2.jar"),
				[]byte{}, 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan.Entries).To(ContainElement(libcnb.BuildpackPlanEntry{
				Name: "dependencies",
				Metadata: map[string]interface{}{
					"dependencies": []libjvm.MavenJAR{
						{
							Name:    "test-file",
							Version: "2.2.2",
							SHA256:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
						},
					},
				},
			}))
		})

		it("contributes from default", func() {
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
`), 0644)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "BOOT-INF", "lib"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "BOOT-INF", "lib", "test-file-2.2.2.jar"),
				[]byte{}, 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan.Entries).To(ContainElement(libcnb.BuildpackPlanEntry{
				Name: "dependencies",
				Metadata: map[string]interface{}{
					"dependencies": []libjvm.MavenJAR{
						{
							Name:    "test-file",
							Version: "2.2.2",
							SHA256:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
						},
					},
				},
			}))
		})
	})

	it("contributes slices from layers index", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Layers-Index: layers.idx
`), 0644)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "layers.idx"), []byte(`
- "alpha":
  - "alpha-1"
  - "alpha-2"
- "bravo":
  - "bravo-1"
  - "bravo-2"
`), 0644)).To(Succeed())

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Slices).To(ContainElements(
			libcnb.Slice{Paths: []string{"alpha-1", "alpha-2"}},
			libcnb.Slice{Paths: []string{"bravo-1", "bravo-2"}},
		))
	})
}
