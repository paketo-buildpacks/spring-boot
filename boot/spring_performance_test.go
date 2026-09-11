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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
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

		ctx                 libcnb.BuildContext
		executor            *mocks.Executor
		aotEnabled          bool
		performanceType     boot.SpringPerformanceType
		javaVersion21Output = `openjdk version "21.0.5" 2024-10-15 LTS
OpenJDK Runtime Environment Temurin-21.0.5+11 (build 21.0.5+11-LTS)
OpenJDK 64-Bit Server VM Temurin-21.0.5+11 (build 21.0.5+11-LTS, mixed mode, sharing)`

		javaVersion25Output = `openjdk version "25.0.1" 2025-10-21
OpenJDK Runtime Environment Temurin-25.0.1+9 (build 25.0.1+9)
OpenJDK 64-Bit Server VM Temurin-25.0.1+9 (build 25.0.1+9, mixed mode, sharing)`

		// Output of `java -XshowSettings:properties -version` for a JDK 25 runtime on amd64.
		javaSettingsOutput = `Property settings:
    java.version = 25.0.1
    java.vendor = Eclipse Adoptium
    os.arch = amd64
    os.name = linux`
	)

	allArgs := func() []string {
		var args []string
		for _, call := range executor.Calls {
			args = append(args, call.Arguments[0].(effect.Execution).Args...)
		}
		return args
	}

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
		aotEnabled = false
		performanceType = boot.Without
	})

	it("contributes Spring Performance for Boot 3.3+, both CdsAotCache & AOT enabled", func() {
		Expect(os.Setenv("JRE_HOME", "/that/does/not/exist")).To(Succeed())

		aotEnabled = true
		performanceType = boot.CdsAotCache
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).
			Return(nil).
			Run(func(args mock.Arguments) {
				execution := args.Get(0).(effect.Execution)
				if (slices.Contains(execution.Args, "-version")) && execution.Stderr != nil {
					_, err := io.WriteString(execution.Stderr, javaVersion21Output)
					Expect(err).NotTo(HaveOccurred())
				}
			}).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LaunchEnvironment["BPL_SPRING_AOT_ENABLED.default"]).To(Equal("true"))
		Expect(layer.LaunchEnvironment["BPL_JVM_AOTCACHE_ENABLED.default"]).To(Equal("true"))

		Expect(executor.Calls).To(HaveLen(3))
		e, ok := executor.Calls[2].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())
		Expect(e.Args).To(ContainElement("-Dspring.aot.enabled=true"))
		Expect(e.Args).To(ContainElements("-Dspring.context.exit=onRefresh",
			"-XX:ArchiveClassesAtExit=application.jsa", "-cp"))
		Expect(layer.Build).To(BeTrue())

	})

	it("contributes Spring Performance for Boot 3.3+, AOT only enabled", func() {
		aotEnabled = true
		performanceType = boot.Without
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
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

	it("contributes Spring Performance for Boot 3.3+, extract layout only enabled", func() {
		aotEnabled = false
		performanceType = boot.ExtractLayout
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LaunchEnvironment["BPL_SPRING_AOT_ENABLED.default"]).To(Equal("false"))
		Expect(layer.LaunchEnvironment["BPL_JVM_CDS_ENABLED.default"]).To(Equal(""))
		Expect(executor.Calls).To(HaveLen(1))

		e, ok := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())
		Expect(e.Args).NotTo(ContainElement("-Dspring.aot.enabled=true"))
		Expect(e.Args).NotTo(ContainElements("-Dspring.context.exit=onRefresh",
			"-XX:ArchiveClassesAtExit=application.jsa", "-cp"))
		Expect(e.Args).To(ContainElement("-Djarmode=tools"))

		Expect(layer.Build).To(BeTrue())

	})

	it("contributes Spring Performance for Boot 3.3+, CdsAotCache only enabled", func() {
		aotEnabled = false
		performanceType = boot.CdsAotCache
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).
			Return(nil).
			Run(func(args mock.Arguments) {
				execution := args.Get(0).(effect.Execution)
				if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
					_, err := io.WriteString(execution.Stderr, javaVersion21Output)
					Expect(err).NotTo(HaveOccurred())
				}
			}).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LaunchEnvironment["BPL_SPRING_AOT_ENABLED.default"]).To(Equal("false"))
		Expect(layer.LaunchEnvironment["BPL_JVM_AOTCACHE_ENABLED.default"]).To(Equal("true"))
		Expect(executor.Calls).To(HaveLen(3))

		e, ok := executor.Calls[2].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())
		Expect(e.Args).NotTo(ContainElement("-Dspring.aot.enabled=true"))
		Expect(e.Args).To(ContainElements("-Dspring.context.exit=onRefresh",
			"-XX:ArchiveClassesAtExit=application.jsa", "-cp"))

		Expect(layer.Build).To(BeTrue())

	})

	it("contributes user-provided JAVA_TOOL_OPTIONS to training run", func() {
		Expect(os.Setenv("JAVA_TOOL_OPTIONS", "default-opt")).To(Succeed())
		aotEnabled = false
		performanceType = boot.CdsAotCache
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).
			Return(nil).
			Run(func(args mock.Arguments) {
				execution := args.Get(0).(effect.Execution)
				if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
					_, err := io.WriteString(execution.Stderr, javaVersion21Output)
					Expect(err).NotTo(HaveOccurred())
				}
			}).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "user-cds-opt")
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.Calls).To(HaveLen(3))
		e, ok := executor.Calls[2].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())

		Expect(e.Env).To(ContainElement("JAVA_TOOL_OPTIONS=user-cds-opt"))
		Expect(layer.Build).To(BeTrue())

		Expect(os.Unsetenv("JAVA_TOOL_OPTIONS")).To(Succeed())
	})

	it("contributes Spring Performance for Boot 3.3+, both CdsAotCache & AOT enabled - with SCB symlink", func() {
		aotEnabled = true
		performanceType = boot.CdsAotCache
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).
			Return(nil).
			Run(func(args mock.Arguments) {
				execution := args.Get(0).(effect.Execution)
				if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
					_, err := io.WriteString(execution.Stderr, javaVersion21Output)
					Expect(err).NotTo(HaveOccurred())
				}
			}).Return(nil)

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

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LaunchEnvironment["BPL_SPRING_AOT_ENABLED.default"]).To(Equal("true"))
		Expect(layer.LaunchEnvironment["BPL_JVM_AOTCACHE_ENABLED.default"]).To(Equal("true"))

		Expect(executor.Calls).To(HaveLen(3))
		e, ok := executor.Calls[2].Arguments[0].(effect.Execution)
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

		aotEnabled = true
		performanceType = boot.CdsAotCache
		dc := libpak.DependencyCache{CachePath: "testdata"}
		executor.On("Execute", mock.Anything).
			Return(nil).
			Run(func(args mock.Arguments) {
				execution := args.Get(0).(effect.Execution)
				if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
					_, err := io.WriteString(execution.Stderr, javaVersion21Output)
					Expect(err).NotTo(HaveOccurred())
				}
			}).Return(nil)

		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
		props, err := libjvm.NewManifest(ctx.Application.Path)
		Expect(err).NotTo(HaveOccurred())

		s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
		s.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(executor.Calls).To(HaveLen(3))
		e, ok := executor.Calls[2].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())

		Expect(e.Command).To(Equal("/that/does/not/exist/bin/java"))
		Expect(layer.Build).To(BeTrue())

		Expect(os.Unsetenv("JRE_HOME")).To(Succeed())
	})

	context("when pre-recorded AOT cache exists", func() {
		it.Before(func() {
			cacheDir := filepath.Join(ctx.Application.Path, "aot-cache")
			Expect(os.MkdirAll(cacheDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "application.aot"), []byte("cache-data"), 0644)).To(Succeed())
		})

		it("skips training run and uses cache", func() {
			aotEnabled = true
			performanceType = boot.CdsAotCache
			dc := libpak.DependencyCache{CachePath: "testdata"}
			executor.On("Execute", mock.Anything).
				Return(nil).
				Run(func(args mock.Arguments) {
					execution := args.Get(0).(effect.Execution)
					if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
						_, err := io.WriteString(execution.Stderr, javaVersion25Output)
						Expect(err).NotTo(HaveOccurred())
					}
				}).Return(nil)

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
			Spring-Boot-Version: 3.3.1
			Spring-Boot-Classes: BOOT-INF/classes
			Spring-Boot-Lib: BOOT-INF/lib
			`), 0644)).To(Succeed())
			props, err := libjvm.NewManifest(ctx.Application.Path)
			Expect(err).NotTo(HaveOccurred())

			s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
			s.Executor = executor

			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			layer, err = s.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			// Training should NOT have been run, but the jar layout is still extracted: the
			// launch process type runs from it, and the cache pins the classpath and the
			// timestamps it was recorded against.
			Expect(allArgs()).NotTo(ContainElement("-XX:AOTCacheOutput=application.aot"))
			Expect(allArgs()).NotTo(ContainElement("-XX:ArchiveClassesAtExit=application.jsa"))
			Expect(allArgs()).To(ContainElement("-Djarmode=tools"))
			Expect(filepath.Join(layer.Path, "runner.jar")).To(BeAnExistingFile())

			// and the image JRE was asked to load it, strictly, before it was accepted
			Expect(allArgs()).To(ContainElements("-XX:AOTMode=on", "-XX:AOTCache="+filepath.Join(layer.Path, "application.aot")))

			// the cache is not left behind to ship a second time in the application layer
			Expect(filepath.Join(ctx.Application.Path, "aot-cache")).NotTo(BeADirectory())

			// Cache file should be copied to the layer path
			Expect(filepath.Join(layer.Path, "application.aot")).To(BeAnExistingFile())

			// AOTCache should be set in launch environment
			Expect(layer.LaunchEnvironment).To(HaveKey("BPL_JVM_AOTCACHE.default"))
			Expect(layer.LaunchEnvironment["BPL_JVM_AOTCACHE.default"]).To(Equal(filepath.Join(layer.Path, "application.aot")))

			// Layer should be marked as launch
			Expect(layer.Launch).To(BeTrue())
		})
	})

	context("when pre-recorded AOT cache has matching metadata", func() {
		it.Before(func() {
			cacheDir := filepath.Join(ctx.Application.Path, "aot-cache")
			Expect(os.MkdirAll(cacheDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "application.aot"), []byte("cache-data"), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "application.aot.meta"), []byte(`{"javaVersion":"25.0.1","osArch":"amd64"}`), 0644)).To(Succeed())
		})

		it("verifies the cache and skips training run", func() {
			aotEnabled = true
			performanceType = boot.CdsAotCache
			dc := libpak.DependencyCache{CachePath: "testdata"}
			executor.On("Execute", mock.Anything).
				Return(nil).
				Run(func(args mock.Arguments) {
					execution := args.Get(0).(effect.Execution)
					if slices.Contains(execution.Args, "-XshowSettings:properties") && execution.Stderr != nil {
						_, err := io.WriteString(execution.Stderr, javaSettingsOutput)
						Expect(err).NotTo(HaveOccurred())
					} else if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
						_, err := io.WriteString(execution.Stderr, javaVersion25Output)
						Expect(err).NotTo(HaveOccurred())
					}
				}).Return(nil)

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
			Spring-Boot-Version: 3.3.1
			Spring-Boot-Classes: BOOT-INF/classes
			Spring-Boot-Lib: BOOT-INF/lib
			`), 0644)).To(Succeed())
			props, err := libjvm.NewManifest(ctx.Application.Path)
			Expect(err).NotTo(HaveOccurred())

			s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
			s.Executor = executor

			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			layer, err = s.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			// Not the training run
			Expect(allArgs()).NotTo(ContainElement("-XX:AOTCacheOutput=application.aot"))
			Expect(allArgs()).NotTo(ContainElement("-XX:ArchiveClassesAtExit=application.jsa"))

			// Cache verified and used
			Expect(filepath.Join(layer.Path, "application.aot")).To(BeAnExistingFile())
			Expect(layer.LaunchEnvironment).To(HaveKey("BPL_JVM_AOTCACHE.default"))
			Expect(layer.Launch).To(BeTrue())
		})
	})

	context("when pre-recorded AOT cache has mismatched version metadata", func() {
		it.Before(func() {
			cacheDir := filepath.Join(ctx.Application.Path, "aot-cache")
			Expect(os.MkdirAll(cacheDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "application.aot"), []byte("cache-data"), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "application.aot.meta"), []byte(`{"javaVersion":"25.0.2","osArch":"amd64"}`), 0644)).To(Succeed())
		})

		it("fails the build on version mismatch", func() {
			aotEnabled = true
			performanceType = boot.CdsAotCache
			dc := libpak.DependencyCache{CachePath: "testdata"}
			executor.On("Execute", mock.Anything).
				Return(nil).
				Run(func(args mock.Arguments) {
					execution := args.Get(0).(effect.Execution)
					if slices.Contains(execution.Args, "-XshowSettings:properties") && execution.Stderr != nil {
						_, err := io.WriteString(execution.Stderr, javaSettingsOutput)
						Expect(err).NotTo(HaveOccurred())
					} else if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
						_, err := io.WriteString(execution.Stderr, javaVersion25Output)
						Expect(err).NotTo(HaveOccurred())
					}
				}).Return(nil)

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
			Spring-Boot-Version: 3.3.1
			Spring-Boot-Classes: BOOT-INF/classes
			Spring-Boot-Lib: BOOT-INF/lib
			`), 0644)).To(Succeed())
			props, err := libjvm.NewManifest(ctx.Application.Path)
			Expect(err).NotTo(HaveOccurred())

			s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
			s.Executor = executor

			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			layer, err = s.Contribute(layer)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("recorded with Java 25.0.2 but the image JRE is Java 25.0.1"))
		})
	})

	context("when pre-recorded AOT cache has mismatched arch metadata", func() {
		it.Before(func() {
			cacheDir := filepath.Join(ctx.Application.Path, "aot-cache")
			Expect(os.MkdirAll(cacheDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "application.aot"), []byte("cache-data"), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "application.aot.meta"), []byte(`{"javaVersion":"25.0.1","osArch":"aarch64"}`), 0644)).To(Succeed())
		})

		it("fails the build on architecture mismatch", func() {
			aotEnabled = true
			performanceType = boot.CdsAotCache
			dc := libpak.DependencyCache{CachePath: "testdata"}
			executor.On("Execute", mock.Anything).
				Return(nil).
				Run(func(args mock.Arguments) {
					execution := args.Get(0).(effect.Execution)
					if slices.Contains(execution.Args, "-XshowSettings:properties") && execution.Stderr != nil {
						_, err := io.WriteString(execution.Stderr, javaSettingsOutput)
						Expect(err).NotTo(HaveOccurred())
					} else if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
						_, err := io.WriteString(execution.Stderr, javaVersion25Output)
						Expect(err).NotTo(HaveOccurred())
					}
				}).Return(nil)

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
			Spring-Boot-Version: 3.3.1
			Spring-Boot-Classes: BOOT-INF/classes
			Spring-Boot-Lib: BOOT-INF/lib
			`), 0644)).To(Succeed())
			props, err := libjvm.NewManifest(ctx.Application.Path)
			Expect(err).NotTo(HaveOccurred())

			s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
			s.Executor = executor

			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			layer, err = s.Contribute(layer)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("recorded for architecture aarch64 but the image JRE architecture is amd64"))
		})
	})

	context("when pre-recorded AOT cache is empty", func() {
		it.Before(func() {
			cacheDir := filepath.Join(ctx.Application.Path, "aot-cache")
			Expect(os.MkdirAll(cacheDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "application.aot"), []byte(""), 0644)).To(Succeed())
		})

		it("falls back to the training run", func() {
			aotEnabled = true
			performanceType = boot.CdsAotCache
			dc := libpak.DependencyCache{CachePath: "testdata"}
			executor.On("Execute", mock.Anything).
				Return(nil).
				Run(func(args mock.Arguments) {
					execution := args.Get(0).(effect.Execution)
					if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
						_, err := io.WriteString(execution.Stderr, javaVersion21Output)
						Expect(err).NotTo(HaveOccurred())
					}
				}).Return(nil)

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
			Spring-Boot-Version: 3.3.1
			Spring-Boot-Classes: BOOT-INF/classes
			Spring-Boot-Lib: BOOT-INF/lib
			`), 0644)).To(Succeed())
			props, err := libjvm.NewManifest(ctx.Application.Path)
			Expect(err).NotTo(HaveOccurred())

			s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
			s.Executor = executor

			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			layer, err = s.Contribute(layer)
			// No error: an empty pre-recorded cache is ignored, not a build failure.
			Expect(err).NotTo(HaveOccurred())

			// The training run must have been executed (executor was called) so a cache
			// is still produced; this runs exactly once and is not a retry.
			Expect(executor.Calls).NotTo(BeEmpty())
			Expect(layer.LaunchEnvironment).NotTo(HaveKey("BPL_JVM_AOTCACHE.default"))
		})
	})

	context("when no pre-recorded AOT cache exists", func() {
		it("runs training as normal", func() {
			aotEnabled = true
			performanceType = boot.CdsAotCache
			dc := libpak.DependencyCache{CachePath: "testdata"}
			executor.On("Execute", mock.Anything).
				Return(nil).
				Run(func(args mock.Arguments) {
					execution := args.Get(0).(effect.Execution)
					if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
						_, err := io.WriteString(execution.Stderr, javaVersion21Output)
						Expect(err).NotTo(HaveOccurred())
					}
				}).Return(nil)

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
			Spring-Boot-Version: 3.3.1
			Spring-Boot-Classes: BOOT-INF/classes
			Spring-Boot-Lib: BOOT-INF/lib
			`), 0644)).To(Succeed())
			props, err := libjvm.NewManifest(ctx.Application.Path)
			Expect(err).NotTo(HaveOccurred())

			s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
			s.Executor = executor

			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			layer, err = s.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			// Training SHOULD have been run
			Expect(executor.Calls).NotTo(BeEmpty())
			// AOTCacheInput should NOT be set
			Expect(layer.LaunchEnvironment).NotTo(HaveKey("BPL_JVM_AOTCACHE_INPUT.default"))
		})
	})

	context("when the image JRE refuses the pre-recorded AOT cache", func() {
		it.Before(func() {
			cacheDir := filepath.Join(ctx.Application.Path, "aot-cache")
			Expect(os.MkdirAll(cacheDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "application.aot"), []byte("not-a-real-cache"), 0644)).To(Succeed())
		})

		it("fails the build rather than shipping a cache that will not load", func() {
			aotEnabled = false
			performanceType = boot.CdsAotCache
			dc := libpak.DependencyCache{CachePath: "testdata"}
			executor.On("Execute", mock.Anything).Return(func(execution effect.Execution) error {
				if slices.Contains(execution.Args, "-XX:AOTMode=on") {
					return fmt.Errorf("exit status 1")
				}
				if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
					_, err := io.WriteString(execution.Stderr, javaVersion25Output)
					Expect(err).NotTo(HaveOccurred())
				}
				return nil
			})

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
			props, err := libjvm.NewManifest(ctx.Application.Path)
			Expect(err).NotTo(HaveOccurred())

			s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "runner.jar", true, "")
			s.Executor = executor

			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			_, err = s.Contribute(layer)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("the image JRE refused to load the pre-recorded AOT cache"))
		})
	})

	context("when pre-recorded AOT cache metadata uses the java.version spelling", func() {
		it.Before(func() {
			cacheDir := filepath.Join(ctx.Application.Path, "aot-cache")
			Expect(os.MkdirAll(cacheDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "application.aot"), []byte("cache-data"), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "application.aot.meta"), []byte(`{"java.version":"25.0.1","os.arch":"aarch64"}`), 0644)).To(Succeed())
		})

		it("still compares it against the image JRE", func() {
			aotEnabled = false
			performanceType = boot.CdsAotCache
			dc := libpak.DependencyCache{CachePath: "testdata"}
			executor.On("Execute", mock.Anything).
				Return(nil).
				Run(func(args mock.Arguments) {
					execution := args.Get(0).(effect.Execution)
					if slices.Contains(execution.Args, "-XshowSettings:properties") && execution.Stderr != nil {
						_, err := io.WriteString(execution.Stderr, javaSettingsOutput)
						Expect(err).NotTo(HaveOccurred())
					} else if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
						_, err := io.WriteString(execution.Stderr, javaVersion25Output)
						Expect(err).NotTo(HaveOccurred())
					}
				}).Return(nil)

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
			props, err := libjvm.NewManifest(ctx.Application.Path)
			Expect(err).NotTo(HaveOccurred())

			s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
			s.Executor = executor

			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			_, err = s.Contribute(layer)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("recorded for architecture aarch64 but the image JRE architecture is amd64"))
		})
	})

	context("when the image JRE is too old for an AOT cache", func() {
		it.Before(func() {
			cacheDir := filepath.Join(ctx.Application.Path, "aot-cache")
			Expect(os.MkdirAll(cacheDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(cacheDir, "application.aot"), []byte("cache-data"), 0644)).To(Succeed())
		})

		it("falls back to the training run", func() {
			aotEnabled = false
			performanceType = boot.CdsAotCache
			dc := libpak.DependencyCache{CachePath: "testdata"}
			executor.On("Execute", mock.Anything).
				Return(nil).
				Run(func(args mock.Arguments) {
					execution := args.Get(0).(effect.Execution)
					if slices.Contains(execution.Args, "-version") && execution.Stderr != nil {
						_, err := io.WriteString(execution.Stderr, javaVersion21Output)
						Expect(err).NotTo(HaveOccurred())
					}
				}).Return(nil)

			Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 3.3.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
`), 0644)).To(Succeed())
			props, err := libjvm.NewManifest(ctx.Application.Path)
			Expect(err).NotTo(HaveOccurred())

			s := boot.NewSpringPerformance(dc, ctx.Application.Path, props, aotEnabled, performanceType, "", true, "")
			s.Executor = executor

			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			layer, err = s.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(allArgs()).To(ContainElement("-XX:ArchiveClassesAtExit=application.jsa"))
			Expect(layer.LaunchEnvironment).NotTo(HaveKey("BPL_JVM_AOTCACHE.default"))
		})
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
