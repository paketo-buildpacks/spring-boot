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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buildpacks/libcnb"
	"github.com/magiconair/properties"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/effect/mocks"
	"github.com/paketo-buildpacks/libpak/sherpa"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/spring-boot/boot"
)

func testNativeImage(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx      libcnb.BuildContext
		executor *mocks.Executor
	)

	it.Before(func() {
		var err error

		ctx.Application.Path, err = ioutil.TempDir("", "native-image-application")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = ioutil.TempDir("", "native-image-layers")
		Expect(err).NotTo(HaveOccurred())

		executor = &mocks.Executor{}
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("contributes native image", func() {
		dep := libpak.BuildpackDependency{
			URI:    "https://localhost/stub-spring-graalvm-native.jar",
			SHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}
		dc := libpak.DependencyCache{CachePath: "testdata"}

		m := properties.NewProperties()
		_, _, err := m.Set("Start-Class", "test-start-class")
		Expect(err).NotTo(HaveOccurred())
		_, _, err = m.Set("Spring-Boot-Classes", "BOOT-INF/classes/")
		Expect(err).NotTo(HaveOccurred())
		_, _, err = m.Set("Spring-Boot-Classpath-Index", "BOOT-INF/classpath.idx")
		Expect(err).NotTo(HaveOccurred())
		_, _, err = m.Set("Spring-Boot-Lib", "BOOT-INF/lib/")
		Expect(err).NotTo(HaveOccurred())

		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "fixture-marker"), []byte{}, 0644)).To(Succeed())

		Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "BOOT-INF"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "BOOT-INF", "classpath.idx"), []byte(`
- "test-jar.jar"`), 0644)).To(Succeed())

		n, err := boot.NewNativeImage(ctx.Application.Path, "test-argument-1 test-argument-2", dep, dc, m,
			ctx.StackID, []sherpa.FileEntry{}, &libcnb.BuildpackPlan{})
		Expect(err).NotTo(HaveOccurred())
		n.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		executor.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
			Expect(ioutil.WriteFile(filepath.Join(layer.Path, "test-start-class"), []byte{}, 0644)).To(Succeed())
		}).Return(nil)

		layer, err = n.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.Cache).To(BeTrue())
		Expect(filepath.Join(layer.Path, "test-start-class")).To(BeARegularFile())
		Expect(filepath.Join(ctx.Application.Path, "test-start-class")).To(BeARegularFile())
		Expect(filepath.Join(ctx.Application.Path, "fixture-marker")).NotTo(BeAnExistingFile())

		execution := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(execution.Command).To(Equal("native-image"))
		Expect(execution.Args).To(Equal([]string{
			"test-argument-1",
			"test-argument-2",
			fmt.Sprintf("-H:Name=%s", filepath.Join(layer.Path, "test-start-class")),
			"-cp",
			strings.Join([]string{
				ctx.Application.Path,
				filepath.Join(ctx.Application.Path, "BOOT-INF", "classes"),
				filepath.Join(ctx.Application.Path, "BOOT-INF", "lib", "test-jar.jar"),
				filepath.Join("testdata", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "stub-spring-graalvm-native.jar"),
			}, ":"),
			"test-start-class",
		}))
		Expect(execution.Dir).To(Equal(layer.Path))
	})

	context("tiny stack", func() {
		it.Before(func() {
			ctx.StackID = libpak.TinyStackID
		})

		it("contributes native image", func() {
			dep := libpak.BuildpackDependency{
				URI:    "https://localhost/stub-spring-graalvm-native.jar",
				SHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			}
			dc := libpak.DependencyCache{CachePath: "testdata"}

			m := properties.NewProperties()
			_, _, err := m.Set("Start-Class", "test-start-class")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = m.Set("Spring-Boot-Classes", "BOOT-INF/classes/")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = m.Set("Spring-Boot-Classpath-Index", "BOOT-INF/classpath.idx")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = m.Set("Spring-Boot-Lib", "BOOT-INF/lib/")
			Expect(err).NotTo(HaveOccurred())

			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "fixture-marker"), []byte{}, 0644)).To(Succeed())

			Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "BOOT-INF"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(ctx.Application.Path, "BOOT-INF", "classpath.idx"), []byte(`
- "test-jar.jar"`), 0644)).To(Succeed())

			n, err := boot.NewNativeImage(ctx.Application.Path, "test-argument-1 test-argument-2", dep, dc, m,
				ctx.StackID, []sherpa.FileEntry{}, &libcnb.BuildpackPlan{})
			Expect(err).NotTo(HaveOccurred())
			n.Executor = executor

			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			executor.On("Execute", mock.Anything).Run(func(args mock.Arguments) {
				Expect(ioutil.WriteFile(filepath.Join(layer.Path, "test-start-class"), []byte{}, 0644)).To(Succeed())
			}).Return(nil)

			layer, err = n.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.Cache).To(BeTrue())
			Expect(filepath.Join(layer.Path, "test-start-class")).To(BeARegularFile())
			Expect(filepath.Join(ctx.Application.Path, "test-start-class")).To(BeARegularFile())
			Expect(filepath.Join(ctx.Application.Path, "fixture-marker")).NotTo(BeAnExistingFile())

			execution := executor.Calls[0].Arguments[0].(effect.Execution)
			Expect(execution.Command).To(Equal("native-image"))
			Expect(execution.Args).To(Equal([]string{
				"test-argument-1",
				"test-argument-2",
				"-H:+StaticExecutableWithDynamicLibC",
				fmt.Sprintf("-H:Name=%s", filepath.Join(layer.Path, "test-start-class")),
				"-cp",
				strings.Join([]string{
					filepath.Join(ctx.Application.Path),
					filepath.Join(ctx.Application.Path, "BOOT-INF", "classes"),
					filepath.Join(ctx.Application.Path, "BOOT-INF", "lib", "test-jar.jar"),
					filepath.Join("testdata", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "stub-spring-graalvm-native.jar"),
				}, ":"),
				"test-start-class",
			}))
			Expect(execution.Dir).To(Equal(layer.Path))
		})
	})
}
