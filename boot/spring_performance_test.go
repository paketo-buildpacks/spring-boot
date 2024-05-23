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
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libjvm"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/effect/mocks"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/spring-boot/v5/boot"
)

func testSpringPerformance(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx        libcnb.BuildContext
		executor   *mocks.Executor
		aotEnabled bool
		cdsEnabled bool
	)

	it.Before(func() {
		var err error

		ctx.Layers.Path, err = os.MkdirTemp("", "spring-performance-layers")
		Expect(err).NotTo(HaveOccurred())

		ctx.Application.Path, err = os.MkdirTemp("", "spring-performance-app-dir")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "META-INF"), 0755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "BOOT-INF/lib"), 0755)).To(Succeed())

		executor = &mocks.Executor{}
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		aotEnabled, cdsEnabled = false, false
	})

	it("contributes Spring Performance for Boot 3.3+, both CDS & AOT enabled", func() {
		aotEnabled, cdsEnabled = true, true
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, cdsEnabled, "", true)
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LaunchEnvironment["BPL_SPRING_AOT_ENABLED.default"]).To(Equal("true"))
		Expect(layer.LaunchEnvironment["BPL_JVM_CDS_ENABLED.default"]).To(Equal("true"))

		Expect(executor.Calls).To(HaveLen(2))
		e, ok := executor.Calls[1].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())
		Expect(e.Args).To(ContainElement("-Dspring.aot.enabled=true"))
		Expect(e.Args).To(ContainElements("-Dspring.context.exit=onRefresh",
			"-XX:ArchiveClassesAtExit=application.jsa", "-cp"))
		Expect(layer.Build).To(BeTrue())

	})

	it("contributes Spring Performance for Boot 3.3+, AOT only enabled", func() {
		aotEnabled, cdsEnabled = true, false
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, cdsEnabled, "", true)
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LaunchEnvironment["BPL_SPRING_AOT_ENABLED.default"]).To(Equal("true"))
		Expect(layer.LaunchEnvironment["BPL_JVM_CDS_ENABLED.default"]).To(Equal(""))
		Expect(executor.Calls).To(HaveLen(0))

		Expect(layer.Build).To(BeTrue())

	})

	it("contributes Spring Performance for Boot 3.3+, CDS only enabled", func() {
		aotEnabled, cdsEnabled = false, true
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, cdsEnabled, "", true)
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LaunchEnvironment["BPL_SPRING_AOT_ENABLED.default"]).To(Equal("false"))
		Expect(layer.LaunchEnvironment["BPL_JVM_CDS_ENABLED.default"]).To(Equal("true"))
		Expect(executor.Calls).To(HaveLen(2))

		e, ok := executor.Calls[1].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())
		Expect(e.Args).NotTo(ContainElement("-Dspring.aot.enabled=true"))
		Expect(e.Args).To(ContainElements("-Dspring.context.exit=onRefresh",
			"-XX:ArchiveClassesAtExit=application.jsa", "-cp"))

		Expect(layer.Build).To(BeTrue())

	})

	it("contributes user-provided JAVA_TOOL_OPTIONS to training run", func() {
		Expect(os.Setenv("JAVA_TOOL_OPTIONS", "default-opt")).To(Succeed())
		Expect(os.Setenv("CDS_TRAINING_JAVA_TOOL_OPTIONS", "user-cds-opt")).To(Succeed())

		aotEnabled, cdsEnabled = true, true
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, cdsEnabled, "", true)
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.Calls).To(HaveLen(2))
		e, ok := executor.Calls[1].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())

		Expect(e.Env).To(ContainElement("JAVA_TOOL_OPTIONS=user-cds-opt"))
		Expect(layer.Build).To(BeTrue())

		Expect(os.Unsetenv("JAVA_TOOL_OPTIONS")).To(Succeed())
		Expect(os.Unsetenv("CDS_TRAINING_JAVA_TOOL_OPTIONS")).To(Succeed())
	})

	it("contributes default JAVA_TOOL_OPTIONS to training run", func() {
		Expect(os.Setenv("JAVA_TOOL_OPTIONS", "default-opt")).To(Succeed())

		aotEnabled, cdsEnabled = true, true
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, cdsEnabled, "", true)
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.Calls).To(HaveLen(2))
		e, ok := executor.Calls[1].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())

		Expect(e.Env).To(ContainElement("JAVA_TOOL_OPTIONS=default-opt"))
		Expect(layer.Build).To(BeTrue())

		Expect(os.Unsetenv("JAVA_TOOL_OPTIONS")).To(Succeed())
	})

	it("contributes Spring Performance for Boot 3.3+, both CDS & AOT enabled - with SCB symlink", func() {
		aotEnabled, cdsEnabled = true, true
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
	Spring-Boot-Version: 3.3.1
	Spring-Boot-Classes: BOOT-INF/classes
	Spring-Boot-Lib: BOOT-INF/lib
	`), 0644)).To(Succeed())

		cwd, _ := os.Getwd()
		old := filepath.Join(cwd, "testdata", "spring-cloud-bindings", "spring-cloud-bindings-1.2.3.jar")
		now := filepath.Join(ctx.Application.Path, "BOOT-INF", "lib", "spring-cloud-bindings-1.2.3.jar")
		os.Symlink(old, now)

		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, cdsEnabled, "", true)
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LaunchEnvironment["BPL_SPRING_AOT_ENABLED.default"]).To(Equal("true"))
		Expect(layer.LaunchEnvironment["BPL_JVM_CDS_ENABLED.default"]).To(Equal("true"))

		Expect(executor.Calls).To(HaveLen(2))
		e, ok := executor.Calls[1].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())
		Expect(e.Args).To(ContainElement("-Dspring.aot.enabled=true"))
		Expect(e.Args).To(ContainElements("-Dspring.context.exit=onRefresh",
			"-XX:ArchiveClassesAtExit=application.jsa", "-cp"))

		unzip(filepath.Join(layer.Path, "runner.jar"), filepath.Join(layer.Path, "extract"))
		fileInfo, err := os.Lstat(filepath.Join(layer.Path, "extract", "BOOT-INF", "lib", "spring-cloud-bindings-1.2.3.jar"))
		Expect(err).NotTo(HaveOccurred())
		// SCB jar is included in the jar, but not as a link, as a real file.
		Expect(fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink).To(BeFalse())
		Expect(layer.Build).To(BeTrue())

	})

	it("fails with a non existing JRE_HOME path", func() {
		Expect(os.Setenv("JRE_HOME", "/that/does/not/exist")).To(Succeed())

		aotEnabled, cdsEnabled = true, true
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, cdsEnabled, "", true)
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.Calls).To(HaveLen(2))
		e, ok := executor.Calls[1].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())

		Expect(e.Command).To(Equal("/that/does/not/exist/bin/java"))
		Expect(layer.Build).To(BeTrue())

		Expect(os.Unsetenv("JRE_HOME")).To(Succeed())
	})

}

func unzip(src, dest string) error {
	dest = filepath.Clean(dest) + "/"

	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer CloseOrPanic(r)()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		path := filepath.Join(dest, f.Name)
		// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
		//if !strings.HasPrefix(path, dest) {
		//	return fmt.Errorf("%s: illegal file path", path)
		//}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer CloseOrPanic(rc)()

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer CloseOrPanic(f)()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func CloseOrPanic(f io.Closer) func() {
	return func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}
}
