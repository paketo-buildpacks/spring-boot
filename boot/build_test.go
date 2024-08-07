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
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libjvm"
	"github.com/paketo-buildpacks/libpak"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/spring-boot/v5/boot"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx   libcnb.BuildContext
		build boot.Build
	)

	var Copy = func(srcPath string, name string, dstPath string) {
		in, err := os.Open(filepath.Join("testdata", srcPath, name))
		Expect(err).NotTo(HaveOccurred())
		defer in.Close()

		out, err := os.OpenFile(filepath.Join(ctx.Application.Path, dstPath, name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		Expect(err).NotTo(HaveOccurred())
		defer out.Close()

		_, err = io.Copy(out, in)
		Expect(err).NotTo(HaveOccurred())
	}

	it.Before(func() {
		var err error

		ctx.Application.Path, err = os.MkdirTemp("", "build-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = os.MkdirTemp("", "build-layers")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "META-INF"), 0755)).To(Succeed())

		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "spring-cloud-bindings",
					"purl":    "pkg:generic/springframework/spring-cloud-bindings@1.1.0",
					"version": "1.1.0",
					"stacks":  []interface{}{"test-stack-id"},
				},
			},
		}
		ctx.StackID = "test-stack-id"
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("does nothing without Spring-Boot-Version", func() {
		result, err := build.Build(ctx)
		Expect(err).To(BeNil())

		Expect(result).To(BeZero())
	})

	it("contributes org.springframework.boot.version label", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Labels).To(ContainElement(libcnb.Label{Key: "org.springframework.boot.version", Value: "1.1.1"}))
	})

	it("skips org.springframework.boot.spring-configuration-metadata.json label when DataFlow is not present", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "spring-configuration-metadata.json"),
			[]byte(`{ "groups": [ { "name": "alpha" } ] }`), 0644))

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Labels).ToNot(ContainElement(libcnb.Label{
			Key:   "org.springframework.boot.spring-configuration-metadata.json",
			Value: `{"groups":[{"name":"alpha"}]}`,
		}))
	})

	it("contributes org.springframework.cloud.dataflow.spring-configuration-metadata.json label", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "spring-configuration-metadata.json"),
			[]byte(`{ "groups": [ { "name": "alpha", "sourceType": "alpha" } ] }`), 0644))
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "dataflow-configuration-metadata.properties"),
			[]byte("configuration-properties.classes=alpha"), 0644))

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Labels).To(ContainElement(libcnb.Label{
			Key:   "org.springframework.cloud.dataflow.spring-configuration-metadata.json",
			Value: `{"groups":[{"name":"alpha","sourceType":"alpha"}]}`,
		}))
	})

	it("contributes org.opencontainers.image.title label", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
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
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
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

	it("contributes dependencies bom entry for API <= 0.6", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "BOOT-INF", "lib"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "BOOT-INF", "lib", "test-file-2.2.2.jar"),
			[]byte{}, 0644)).To(Succeed())
		ctx.Buildpack.API = "0.6"

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.BOM.Entries).To(ContainElement(libcnb.BOMEntry{
			Name: "dependencies",
			Metadata: map[string]interface{}{
				"layer": "application",
				"dependencies": []libjvm.MavenJAR{
					{
						Name:    "test-file",
						Version: "2.2.2",
						SHA256:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					},
				},
			},
			Build:  false,
			Launch: true,
		}))
	})

	it("contributes to the result for API 0.7+", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		ctx.Buildpack.API = "0.7"
		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "spring-cloud-bindings",
					"version": "1.1.0",
					"stacks":  []interface{}{"test-stack-id"},
					"cpes":    []string{"cpe:2.3:a:vmware:spring_cloud_bindings:1.8.0:*:*:*:*:*:*:*"},
					"purl":    "pkg:generic/springframework/spring-cloud-bindings@1.8.0",
				},
			},
		}

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(3))
		Expect(result.Layers[2].Name()).To(Equal("helper"))
		Expect(result.Layers[2].(libpak.HelperLayerContributor).Names).To(Equal([]string{"spring-cloud-bindings"}))
		Expect(result.Layers[0].Name()).To(Equal("spring-cloud-bindings"))
		Expect(result.Layers[1].Name()).To(Equal("web-application-type"))

		Expect(result.BOM.Entries).To(HaveLen(3))
		Expect(result.BOM.Entries[1].Name).To(Equal("dependencies"))
		Expect(result.BOM.Entries[2].Name).To(Equal("helper"))
		Expect(result.BOM.Entries[1].Launch).To(BeTrue())
		Expect(result.BOM.Entries[1].Build).To(BeFalse())
		Expect(result.BOM.Entries[0].Name).To(Equal("spring-cloud-bindings"))
		Expect(result.BOM.Entries[2].Launch).To(BeTrue())
		Expect(result.BOM.Entries[2].Build).To(BeFalse())
	})

	it("contributes to the result for API <= 0.6", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

		ctx.Buildpack.API = "0.6"

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(3))
		Expect(result.Layers[2].Name()).To(Equal("helper"))
		Expect(result.Layers[2].(libpak.HelperLayerContributor).Names).To(Equal([]string{"spring-cloud-bindings"}))
		Expect(result.Layers[0].Name()).To(Equal("spring-cloud-bindings"))
		Expect(result.Layers[1].Name()).To(Equal("web-application-type"))

		Expect(result.BOM.Entries).To(HaveLen(3))
		Expect(result.BOM.Entries[1].Name).To(Equal("dependencies"))
		Expect(result.BOM.Entries[2].Name).To(Equal("helper"))
		Expect(result.BOM.Entries[2].Launch).To(BeTrue())
		Expect(result.BOM.Entries[2].Build).To(BeFalse())
		Expect(result.BOM.Entries[0].Name).To(Equal("spring-cloud-bindings"))
		Expect(result.BOM.Entries[0].Launch).To(BeTrue())
		Expect(result.BOM.Entries[0].Build).To(BeTrue())
	})

	it("contributes slices from layers index", func() {
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
Spring-Boot-Layers-Index: layers.idx
`), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "layers.idx"), []byte(`
- "alpha":
  - "testdata/alpha/alpha-1"
  - "testdata/alpha/alpha-2"
- "bravo":
  - "bravo-1"
  - "bravo-2"
`), 0644)).To(Succeed())

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Slices).To(ContainElements(
			libcnb.Slice{Paths: []string{"testdata/alpha/alpha-1", "testdata/alpha/alpha-2"}},
			libcnb.Slice{Paths: []string{"bravo-1", "bravo-2"}},
		))
	})

	context("when building a native image", func() {
		it.Before(func() {
			ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{
				Name:     "spring-boot",
				Metadata: map[string]interface{}{"native-image": true},
			})
		})

		it("sets the CLASSPATH for the native image build", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			Expect(result.Layers[0].Name()).To(Equal("Class Path"))
		})

		it("adds no slices to the result", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Slices).To(HaveLen(0))
		})

		it("does not change META-INF/services if FileSystemProvider exists but Spring Boot < 3.2", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

			Expect(os.Mkdir(filepath.Join(ctx.Application.Path, "META-INF", "services"), 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "services", "java.nio.file.spi.FileSystemProvider"),
				[]byte(`org.springframework.boot.loader.nio.file.NestedFileSystemProvider`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Layers).To(HaveLen(1))

			fileBytes, err := os.ReadFile(filepath.Join(ctx.Application.Path, "META-INF", "services", "java.nio.file.spi.FileSystemProvider"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(fileBytes)).To(Equal("org.springframework.boot.loader.nio.file.NestedFileSystemProvider"))

		})

		it("does not blow up if META-INF/services does not exist and Spring Boot >= 3.2", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.2.0
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Layers).To(HaveLen(1))
		})

		it("does not blow up if META-INF/services/java.nio.file.spi.FileSystemProvider does not exist and Spring Boot >= 3.2", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.2.0
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(ctx.Application.Path, "META-INF", "services"), 0755)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Layers).To(HaveLen(1))
		})

		it("changes META-INF/services removing FileSystemProvider if Spring Boot >= 3.2 and only line is NestedFileSystemProvider", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.2.0
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

			Expect(os.Mkdir(filepath.Join(ctx.Application.Path, "META-INF", "services"), 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "services", "java.nio.file.spi.FileSystemProvider"),
				[]byte(`org.springframework.boot.loader.nio.file.NestedFileSystemProvider`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Layers).To(HaveLen(1))

			_, err = os.Stat(filepath.Join(ctx.Application.Path, "META-INF", "services", "java.nio.file.spi.FileSystemProvider"))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		it("changes META-INF/services removing line NestedFileSystemProvider from FileSystemProvider if Spring Boot >= 3.2", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.2.0
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

			Expect(os.Mkdir(filepath.Join(ctx.Application.Path, "META-INF", "services"), 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "services", "java.nio.file.spi.FileSystemProvider"),
				[]byte(`org.springframework.boot.loader.nio.file.NestedFileSystemProvider
jdk.nio.zipfs.ZipFileSystemProvider`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Layers).To(HaveLen(1))

			fileBytes, err := os.ReadFile(filepath.Join(ctx.Application.Path, "META-INF", "services", "java.nio.file.spi.FileSystemProvider"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(fileBytes)).To(Equal("jdk.nio.zipfs.ZipFileSystemProvider"))
		})

	})

	context("when a native-processed BuildPlanEntry is found with a native-image sub entry", func() {
		it.Before(func() {
			ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{
				Name:     "native-processed",
				Metadata: map[string]interface{}{"native-image": true},
			})
		})

		it("contributes a native image build", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			Expect(result.Layers[0].Name()).To(Equal("Class Path"))
			Expect(result.Slices).To(HaveLen(0))
		})

	})

	context("set BP_SPRING_CLOUD_BINDINGS_DISABLED to true", func() {
		it.Before(func() {
			t.Setenv("BP_SPRING_CLOUD_BINDINGS_DISABLED", "true")
		})

		it("contributes to the result for API 0.7+", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
			ctx.Buildpack.API = "0.7"
			ctx.Buildpack.Metadata = map[string]interface{}{
				"dependencies": []map[string]interface{}{
					{
						"id":      "spring-cloud-bindings",
						"version": "1.1.0",
						"stacks":  []interface{}{"test-stack-id"},
						"cpes":    []string{"cpe:2.3:a:vmware:spring_cloud_bindings:1.8.0:*:*:*:*:*:*:*"},
						"purl":    "pkg:generic/springframework/spring-cloud-bindings@1.8.0",
					},
				},
			}

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			Expect(result.Layers[0].Name()).To(Equal("web-application-type"))

			Expect(result.BOM.Entries).To(HaveLen(1))
			Expect(result.BOM.Entries[0].Name).To(Equal("dependencies"))
		})

		it("contributes to the result for API <= 0.6", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

			ctx.Buildpack.API = "0.6"

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			Expect(result.Layers[0].Name()).To(Equal("web-application-type"))

			Expect(result.BOM.Entries).To(HaveLen(1))
			Expect(result.BOM.Entries[0].Name).To(Equal("dependencies"))
		})
	})

	context("have an existing spring-cloud-bindings jar among spring libs", func() {

		mavenJars := make([]libjvm.MavenJAR, 0, 2)
		mavenJars = append(mavenJars, libjvm.MavenJAR{
			Name:    "spring-boot",
			Version: "3",
			SHA256:  "1",
		})
		mavenJars = append(mavenJars, libjvm.MavenJAR{
			Name:    "junit",
			Version: "5",
			SHA256:  "2",
		})

		it("returns the version of the found spring cloud bindings jar", func() {
			expectedScbJar := libjvm.MavenJAR{
				Name:    "spring-cloud-bindings",
				Version: "1.8.1",
				SHA256:  "79a036f93414230a402d30d75ab2ccec9a953259bdeb3dd31e8fee2056445df3",
			}
			mavenJars = append(mavenJars, expectedScbJar)

			// add another jar to make sure the detection stops correctly after the first found occurence
			mavenJars = append(mavenJars, libjvm.MavenJAR{
				Name:    "tomcat",
				Version: "10",
				SHA256:  "3",
			})
			scbJarFound := boot.FindExistingDependency(mavenJars, "spring-cloud-bindings")
			Expect(scbJarFound == true)
		})

		it("returns no match", func() {
			scbJarFound := boot.FindExistingDependency(mavenJars, "spring-cloud-bindings")
			Expect(scbJarFound == false)
		})

	})

	context("when the Spring Boot lib folder already contains a spring-cloud-bindings jar", func() {

		it("contributes to the result for API 0.7+", func() {
			os.MkdirAll(filepath.Join(ctx.Application.Path, "BOOT-INF/lib"), 0755)
			Copy("spring-cloud-bindings", "spring-cloud-bindings-1.2.3.jar", "BOOT-INF/lib")

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
			ctx.Buildpack.API = "0.7"
			ctx.Buildpack.Metadata = map[string]interface{}{
				"dependencies": []map[string]interface{}{
					{
						"id":      "spring-cloud-bindings",
						"version": "1.1.0",
						"stacks":  []interface{}{"test-stack-id"},
						"cpes":    []string{"cpe:2.3:a:vmware:spring_cloud_bindings:1.8.0:*:*:*:*:*:*:*"},
						"purl":    "pkg:generic/springframework/spring-cloud-bindings@1.8.0",
					},
				},
			}

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			Expect(result.Layers[0].Name()).To(Equal("web-application-type"))

			Expect(result.BOM.Entries).To(HaveLen(1))
			Expect(result.BOM.Entries[0].Name).To(Equal("dependencies"))
		})

		it("contributes to the result for API <= 0.6", func() {
			os.MkdirAll(filepath.Join(ctx.Application.Path, "BOOT-INF/lib"), 0755)
			Copy("spring-cloud-bindings", "spring-cloud-bindings-1.2.3.jar", "BOOT-INF/lib")

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

			ctx.Buildpack.API = "0.6"

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			Expect(result.Layers[0].Name()).To(Equal("web-application-type"))

			Expect(result.BOM.Entries).To(HaveLen(1))
			Expect(result.BOM.Entries[0].Name).To(Equal("dependencies"))
		})
	})

	context("when there are multiple Spring Cloud Binding versions", func() {
		it.Before(func() {
			ctx.Buildpack.Metadata = map[string]interface{}{
				"dependencies": []map[string]interface{}{
					{
						"id":      "spring-cloud-bindings",
						"version": "1.1.0",
						"stacks":  []interface{}{"test-stack-id"},
						"cpes":    []string{"cpe:2.3:a:vmware:spring_cloud_bindings:1.8.0:*:*:*:*:*:*:*"},
						"purl":    "pkg:generic/springframework/spring-cloud-bindings@1.8.0",
					},
					{
						"id":      "spring-cloud-bindings",
						"version": "2.1.0",
						"stacks":  []interface{}{"test-stack-id"},
						"cpes":    []string{"cpe:2.3:a:vmware:spring_cloud_bindings:1.8.0:*:*:*:*:*:*:*"},
						"purl":    "pkg:generic/springframework/spring-cloud-bindings@1.8.0",
					},
				},
			}
		})

		it("installs the correct bindings version based on the Spring Boot version in manifest (2.x)", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
						Spring-Boot-Version: 2.1.1
						Spring-Boot-Classes: BOOT-INF/classes
						Spring-Boot-Lib: BOOT-INF/lib
						`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers[0].(boot.SpringCloudBindings).LayerContributor.Dependency.Version).To(Equal("1.1.0"))

		})

		it("installs the correct bindings version based on the Spring Boot version in manifest (3.x)", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
						Spring-Boot-Version: 3.1.0
						Spring-Boot-Classes: BOOT-INF/classes
						Spring-Boot-Lib: BOOT-INF/lib
						`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers[0].(boot.SpringCloudBindings).LayerContributor.Dependency.Version).To(Equal("2.1.0"))

		})

		it("installs the correct bindings version based on the SCB Configuration Variable", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
						Spring-Boot-Version: 2.1.1
						Spring-Boot-Classes: BOOT-INF/classes
						Spring-Boot-Lib: BOOT-INF/lib
						`), 0644)).To(Succeed())
			t.Setenv("BP_SPRING_CLOUD_BINDINGS_VERSION", "2")
			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers[0].(boot.SpringCloudBindings).LayerContributor.Dependency.Version).To(Equal("2.1.0"))

		})
	})

	context("when BP_JVM_CDS_ENABLED is enabled", func() {

		it.Before(func() {
			t.Setenv("BP_JVM_CDS_ENABLED", "true")
			t.Setenv("BP_SPRING_CLOUD_BINDINGS_DISABLED", "true")
		})

		it.After(func() {
			os.Unsetenv("BP_JVM_CDS_ENABLED")
			os.Unsetenv("BP_SPRING_CLOUD_BINDINGS_DISABLED")
			os.Unsetenv("BP_SPRING_AOT_ENABLED")
			os.Unsetenv("CDS_TRAINING_JAVA_TOOL_OPTIONS")
		})

		ctx.Buildpack.API = "0.6"

		it("contributes CDS layer & helper for Boot 3.3+ apps", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
			Spring-Boot-Version: 3.3.1
			Start-Class: test-class
			Spring-Boot-Classes: BOOT-INF/classes
			Spring-Boot-Lib: BOOT-INF/lib
			`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(3))
			Expect(result.Layers[2].Name()).To(Equal("helper"))
			Expect(result.Layers[2].(libpak.HelperLayerContributor).Names).To(Equal([]string{"performance"}))
		})

		it("contributes CDS layer & helper for Boot 3.3+ apps even when they're jar'ed", func() {

			Copy("cds", "spring-app-3.3-no-dependencies.jar", "")

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(3))
			Expect(result.Layers[2].Name()).To(Equal("helper"))
			Expect(result.Layers[2].(libpak.HelperLayerContributor).Names).To(Equal([]string{"performance"}))
		})

		it("does not contribute CDS layer & helper for Boot < 3.3 apps", func() {
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
			Spring-Boot-Version: 3.2.1
			Start-Class: test-class
			Spring-Boot-Classes: BOOT-INF/classes
			Spring-Boot-Lib: BOOT-INF/lib
			`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			Expect(result.Layers[0].Name()).To(Equal("web-application-type"))
		})

		it("contributes CDS layer & helper for Boot 3.3+ apps with BP_SPRING_AOT_ENABLED and CDS_TRAINING_JAVA_TOOL_OPTIONS not set", func() {
			t.Setenv("BP_SPRING_AOT_ENABLED", "true")
			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
			Spring-Boot-Version: 3.3.1
			Start-Class: test-class
			Spring-Boot-Classes: BOOT-INF/classes
			Spring-Boot-Lib: BOOT-INF/lib
			`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(3))
			Expect(result.Layers[2].Name()).To(Equal("helper"))
			Expect(result.Layers[2].(libpak.HelperLayerContributor).Names).To(Equal([]string{"performance"}))
		})

		it("fails the build because CDS_TRAINING_JAVA_TOOL_OPTIONS was provided with BP_SPRING_AOT_ENABLED", func() {
			t.Setenv("BP_SPRING_AOT_ENABLED", "true")
			t.Setenv("CDS_TRAINING_JAVA_TOOL_OPTIONS", "user-cds-opt")

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
			Spring-Boot-Version: 3.3.1
			Start-Class: test-class
			Spring-Boot-Classes: BOOT-INF/classes
			Spring-Boot-Lib: BOOT-INF/lib
			`), 0644)).To(Succeed())

			Expect(os.Mkdir(filepath.Join(ctx.Application.Path, "META-INF", "native-image"), 0755)).To(Succeed())

			_, err := build.Build(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("build failed because of invalid user configuration"))
		})

		it("contributes CDS layer & helper for Boot 3.3+ apps with CDS_TRAINING_JAVA_TOOL_OPTIONS but BP_SPRING_AOT_ENABLED is disabled", func() {
			t.Setenv("BP_SPRING_AOT_ENABLED", "false")
			t.Setenv("CDS_TRAINING_JAVA_TOOL_OPTIONS", "user-cds-opt")

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
			Spring-Boot-Version: 3.3.1
			Start-Class: test-class
			Spring-Boot-Classes: BOOT-INF/classes
			Spring-Boot-Lib: BOOT-INF/lib
			`), 0644)).To(Succeed())

			Expect(os.Mkdir(filepath.Join(ctx.Application.Path, "META-INF", "native-image"), 0755)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(3))

			springPerformanceLayer := result.Layers[0].(boot.SpringPerformance)
			// specific CDS_TRAINING_JAVA_TOOL_OPTIONS was set, so we set it in the training run
			Expect(springPerformanceLayer.TrainingRunJavaToolOptions).To(Equal("user-cds-opt"))

			Expect(result.Layers[2].Name()).To(Equal("helper"))
			Expect(result.Layers[2].(libpak.HelperLayerContributor).Names).To(Equal([]string{"performance"}))
		})

		it("contributes CDS layer & helper for Boot 3.3+ apps with BP_SPRING_AOT_ENABLED and JAVA_TOOL_OPTIONS set", func() {
			t.Setenv("BP_SPRING_AOT_ENABLED", "true")
			t.Setenv("CDS_TRAINING_JAVA_TOOL_OPTIONS", "default-opt")

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
			Spring-Boot-Version: 3.3.1
			Start-Class: test-class
			Spring-Boot-Classes: BOOT-INF/classes
			Spring-Boot-Lib: BOOT-INF/lib
			`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(3))

			springPerformanceLayer := result.Layers[0].(boot.SpringPerformance)
			// no specific CDS_TRAINING_JAVA_TOOL_OPTIONS was set, but JAVA_TOOL_OPTIONS was, so we set it in the training run as well
			Expect(springPerformanceLayer.TrainingRunJavaToolOptions).To(Equal("default-opt"))

			Expect(result.Layers[2].Name()).To(Equal("helper"))
			Expect(result.Layers[2].(libpak.HelperLayerContributor).Names).To(Equal([]string{"performance"}))
		})

	})

	context("when there is a non-exploded jar passed to the buildpack", func() {

		it.Before(func() {
			t.Setenv("BP_SPRING_CLOUD_BINDINGS_DISABLED", "true")
			os.Remove(filepath.Join(ctx.Application.Path, "META-INF"))
		})

		it("finds and extracts a jar that exists", func() {

			Copy("cds", "spring-app-3.3-no-dependencies.jar", "")

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "other-file"), []byte(`
			stuff
			`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Layers).To(HaveLen(1))
		})

		it("returns silently if no jar is found", func() {

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "other-file"), []byte(`
			stuff
			`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(libcnb.BuildResult{}))
		})

		it("returns silently if only a non-boot jar is found", func() {

			Copy("", "stub-empty.jar", "")

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(libcnb.BuildResult{}))
		})
	})
}
