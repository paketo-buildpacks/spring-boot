/*
 * Copyright 2018-2021 the original author or authors.
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
	"strings"
	"testing"

	"github.com/buildpacks/libcnb"
	"github.com/magiconair/properties"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/spring-boot/v5/boot"
)

func testNativeImage(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		appDir      string
		contributor boot.NativeImageClasspath
		manifest    *properties.Properties
		layer       libcnb.Layer
		layerDir    string
	)

	it.Before(func() {
		var err error
		appDir, err = ioutil.TempDir("", "native-image-application")
		Expect(err).NotTo(HaveOccurred())

		layerDir, err = ioutil.TempDir("", "classpath-layer")
		Expect(err).NotTo(HaveOccurred())
		layers := &libcnb.Layers{Path: layerDir}
		layer, err = layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		manifest = properties.NewProperties()

		_, _, err = manifest.Set("Start-Class", "test-start-class")
		Expect(err).NotTo(HaveOccurred())
		_, _, err = manifest.Set("Spring-Boot-Classes", "BOOT-INF/classes/")
		Expect(err).NotTo(HaveOccurred())
		_, _, err = manifest.Set("Spring-Boot-Classpath-Index", "BOOT-INF/classpath.idx")
		Expect(err).NotTo(HaveOccurred())
		_, _, err = manifest.Set("Spring-Boot-Lib", "BOOT-INF/lib/")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(appDir, "BOOT-INF"), 0755)).To(Succeed())

		contributor, err = boot.NewNativeImageClasspath(appDir, manifest)
		Expect(err).NotTo(HaveOccurred())
		Expect(err).NotTo(HaveOccurred())

		Expect(err).NotTo(HaveOccurred())
	})

	context("classpath.idx contains a list of jar", func() {
		it.Before(func() {
			Expect(ioutil.WriteFile(filepath.Join(appDir, "BOOT-INF", "classpath.idx"), []byte(`
- "some.jar"
- "other.jar"
`), 0644)).To(Succeed())
		})

		it("sets CLASSPATH for build", func() {
			layer, err := contributor.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.BuildEnvironment["CLASSPATH.append"]).To(Equal(strings.Join([]string{
				filepath.Join(appDir, "BOOT-INF", "classes"),
				filepath.Join(appDir, "BOOT-INF", "lib", "some.jar"),
				filepath.Join(appDir, "BOOT-INF", "lib", "other.jar"),
			}, ":")))
			Expect(layer.LayerTypes.Build).To(BeTrue())
			Expect(layer.LayerTypes.Launch).To(BeFalse())
		})
	})

	context("classpath.idx contains a list of relative paths to jars", func() {
		it.Before(func() {
			Expect(ioutil.WriteFile(filepath.Join(appDir, "BOOT-INF", "classpath.idx"), []byte(`
- "some/path/some.jar"
- "some/path/other.jar"
`), 0644)).To(Succeed())
		})

		it("sets CLASSPATH for build", func() {
			layer, err := contributor.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.BuildEnvironment["CLASSPATH.append"]).To(Equal(strings.Join([]string{
				filepath.Join(appDir, "BOOT-INF", "classes"),
				filepath.Join(appDir, "some", "path", "some.jar"),
				filepath.Join(appDir, "some", "path", "other.jar"),
			}, ":")))
			Expect(layer.LayerTypes.Build).To(BeTrue())
			Expect(layer.LayerTypes.Launch).To(BeFalse())
		})
	})
	context("Boot @argfile is found", func() {
		it.Before(func() {
			Expect(ioutil.WriteFile(filepath.Join(appDir, "BOOT-INF", "classpath.idx"), []byte(`
- "some.jar"
- "other.jar"
`), 0644)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(appDir, "META-INF", "native-image"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(appDir, "META-INF", "native-image", "argfile"), []byte("file-data"), 0644)).To(Succeed())
		})

		it("ensures BP_NATIVE_IMAGE_BUILD_ARGUMENTS_FILE is set when argfile is found", func() {
			layer, err := contributor.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.BuildEnvironment["BP_NATIVE_IMAGE_BUILD_ARGUMENTS_FILE.default"]).To(Equal(
				filepath.Join(appDir, "META-INF", "native-image", "argfile")))
			Expect(layer.LayerTypes.Build).To(BeTrue())
			Expect(layer.LayerTypes.Launch).To(BeFalse())
		})
	})
}
