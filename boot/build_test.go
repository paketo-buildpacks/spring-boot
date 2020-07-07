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
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/spring-boot/boot"
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

		ctx.Layers.Path, err = ioutil.TempDir("", "build-layers")
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

	it("contributes org.springframework.boot.version label", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Labels).To(ContainElement(libcnb.Label{Key: "org.springframework.boot.version", Value: "1.1.1"}))
	})

	it("contributes org.springframework.boot.spring-configuration-metadata.json label", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "spring-configuration-metadata.json"),
			[]byte(`{ "groups": [ { "name": "alpha" } ] }`), 0644))

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Labels).To(ContainElement(libcnb.Label{
			Key:   "org.springframework.boot.spring-configuration-metadata.json",
			Value: `{"groups":[{"name":"alpha"}]}`,
		}))
	})

	it("contributes org.springframework.cloud.dataflow.spring-configuration-metadata.json label", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "spring-configuration-metadata.json"),
			[]byte(`{ "groups": [ { "name": "alpha", "sourceType": "alpha" } ] }`), 0644))
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "dataflow-configuration-metadata-whitelist.properties"),
			[]byte("configuration-properties.classes=alpha"), 0644))

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Labels).To(ContainElement(libcnb.Label{
			Key:   "org.springframework.cloud.dataflow.spring-configuration-metadata.json",
			Value: `{"groups":[{"name":"alpha","sourceType":"alpha"}]}`,
		}))
	})

	it("contributes org.opencontainers.image.title label", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
Implementation-Title: test-title
`), 0644)).To(Succeed())

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Labels).To(ContainElement(libcnb.Label{
			Key:   "org.opencontainers.image.title",
			Value: "test-title",
		}))
	})

	it("contributes org.opencontainers.image.version label", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
Implementation-Version: 2.2.2
`), 0644)).To(Succeed())

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Labels).To(ContainElement(libcnb.Label{
			Key:   "org.opencontainers.image.version",
			Value: "2.2.2",
		}))
	})

	it("contributes dependencies plan entry", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
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

	it("adds web-application-type to the result", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(1))
	})

	context("BP_BOOT_NATIVE_IMAGE", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_BOOT_NATIVE_IMAGE", "")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_BOOT_NATIVE_IMAGE")).To(Succeed())
		})

		it("contributes native image layer", func() {
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
Spring-Boot-Layers-Index: layers.idx
Start-Class: test-start-class
`), 0644)).To(Succeed())

			ctx.Buildpack.Metadata = map[string]interface{}{
				"dependencies": []map[string]interface{}{
					{
						"id":      "spring-graalvm-native",
						"version": "1.1.1",
						"stacks":  []interface{}{"test-stack-id"},
					},
				},
			}
			ctx.StackID = "test-stack-id"

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			Expect(result.Layers[0].(boot.NativeImage).Arguments).To(BeEmpty())
			Expect(result.Processes).To(ContainElements(
				libcnb.Process{Type: "native-image", Command: filepath.Join(ctx.Application.Path, "test-start-class"), Direct: true},
				libcnb.Process{Type: "task", Command: filepath.Join(ctx.Application.Path, "test-start-class"), Direct: true},
				libcnb.Process{Type: "web", Command: filepath.Join(ctx.Application.Path, "test-start-class"), Direct: true},
			))
		})

		context("BP_BOOT_NATIVE_IMAGE_BUILD_ARGUMENTS", func() {
			it.Before(func() {
				Expect(os.Setenv("BP_BOOT_NATIVE_IMAGE_BUILD_ARGUMENTS", "test-native-image-argument")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("BP_BOOT_NATIVE_IMAGE_BUILD_ARGUMENTS")).To(Succeed())
			})

			it("contributes native image build arguments", func() {
				Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
Spring-Boot-Layers-Index: layers.idx
Start-Class: test-start-class
`), 0644)).To(Succeed())

				ctx.Buildpack.Metadata = map[string]interface{}{
					"dependencies": []map[string]interface{}{
						{
							"id":      "spring-graalvm-native",
							"version": "1.1.1",
							"stacks":  []interface{}{"test-stack-id"},
						},
					},
				}
				ctx.StackID = "test-stack-id"

				result, err := build.Build(ctx)
				Expect(err).NotTo(HaveOccurred())

				Expect(result.Layers[0].(boot.NativeImage).Arguments).To(Equal([]string{"test-native-image-argument"}))
			})

		})
	})

	it("contributes slices from layers index", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
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
