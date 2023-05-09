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
	"io/ioutil"
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

	it.Before(func() {
		var err error

		ctx.Application.Path, err = ioutil.TempDir("", "build-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = ioutil.TempDir("", "build-layers")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "META-INF"), 0755)).To(Succeed())

		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "spring-cloud-bindings",
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

	it("skips org.springframework.boot.spring-configuration-metadata.json label when DataFlow is not present", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "spring-configuration-metadata.json"),
			[]byte(`{ "groups": [ { "name": "alpha" } ] }`), 0644))

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Labels).ToNot(ContainElement(libcnb.Label{
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
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "dataflow-configuration-metadata.properties"),
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

	it("contributes dependencies bom entry for API <= 0.6", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "BOOT-INF", "lib"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "BOOT-INF", "lib", "test-file-2.2.2.jar"),
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
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
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
		Expect(result.Layers[0].Name()).To(Equal("helper"))
		Expect(result.Layers[0].(libpak.HelperLayerContributor).Names).To(Equal([]string{"spring-cloud-bindings"}))
		Expect(result.Layers[1].Name()).To(Equal("spring-cloud-bindings"))
		Expect(result.Layers[2].Name()).To(Equal("web-application-type"))

		Expect(result.BOM.Entries).To(HaveLen(3))
		Expect(result.BOM.Entries[0].Name).To(Equal("dependencies"))
		Expect(result.BOM.Entries[1].Name).To(Equal("helper"))
		Expect(result.BOM.Entries[1].Launch).To(BeTrue())
		Expect(result.BOM.Entries[1].Build).To(BeFalse())
		Expect(result.BOM.Entries[2].Name).To(Equal("spring-cloud-bindings"))
		Expect(result.BOM.Entries[2].Launch).To(BeTrue())
		Expect(result.BOM.Entries[2].Build).To(BeFalse())
	})

	it("contributes to the result for API <= 0.6", func() {
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

		ctx.Buildpack.API = "0.6"

		result, err := build.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(3))
		Expect(result.Layers[0].Name()).To(Equal("helper"))
		Expect(result.Layers[0].(libpak.HelperLayerContributor).Names).To(Equal([]string{"spring-cloud-bindings"}))
		Expect(result.Layers[1].Name()).To(Equal("spring-cloud-bindings"))
		Expect(result.Layers[2].Name()).To(Equal("web-application-type"))

		Expect(result.BOM.Entries).To(HaveLen(3))
		Expect(result.BOM.Entries[0].Name).To(Equal("dependencies"))
		Expect(result.BOM.Entries[1].Name).To(Equal("helper"))
		Expect(result.BOM.Entries[1].Launch).To(BeTrue())
		Expect(result.BOM.Entries[1].Build).To(BeFalse())
		Expect(result.BOM.Entries[2].Name).To(Equal("spring-cloud-bindings"))
		Expect(result.BOM.Entries[2].Launch).To(BeTrue())
		Expect(result.BOM.Entries[2].Build).To(BeFalse())
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

	context("when building a native image", func() {
		it.Before(func() {
			ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{
				Name:     "spring-boot",
				Metadata: map[string]interface{}{"native-image": true},
			})
		})

		it("sets the CLASSPATH for the native image build", func() {
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
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
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())

			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Slices).To(HaveLen(0))
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
			Expect(os.Setenv("BP_SPRING_CLOUD_BINDINGS_DISABLED", "true")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv(("BP_SPRING_CLOUD_BINDINGS_DISABLED"))).To(Succeed())
		})

		it("contributes to the result for API 0.7+", func() {
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
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
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
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

		var Copy = func(name string) {
			in, err := os.Open(filepath.Join("testdata", "spring-cloud-bindings", name))
			Expect(err).NotTo(HaveOccurred())
			defer in.Close()

			out, err := os.OpenFile(filepath.Join(ctx.Application.Path, "BOOT-INF/lib", name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			Expect(err).NotTo(HaveOccurred())
			defer out.Close()

			_, err = io.Copy(out, in)
			Expect(err).NotTo(HaveOccurred())
		}

		it("contributes to the result for API 0.7+", func() {
			os.MkdirAll(filepath.Join(ctx.Application.Path, "BOOT-INF/lib"), 0755)
			Copy("spring-cloud-bindings-1.2.3.jar")

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
			Copy("spring-cloud-bindings-1.2.3.jar")

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
}
