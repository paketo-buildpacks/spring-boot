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
	"io/fs"

	"github.com/paketo-buildpacks/libpak/crush"
	"github.com/paketo-buildpacks/libpak/sherpa"

	"os"
	"path/filepath"
	"time"

	"github.com/buildpacks/libcnb"
	"github.com/magiconair/properties"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
)

type SpringPerformance struct {
	Dependency                 libpak.BuildpackDependency
	LayerContributor           libpak.LayerContributor
	Logger                     bard.Logger
	Executor                   effect.Executor
	AppPath                    string
	Manifest                   *properties.Properties
	AotEnabled                 bool
	PerformanceType            SpringPerformanceType
	ClasspathString            string
	ReZip                      bool
	TrainingRunJavaToolOptions string
}

func NewSpringPerformance(cache libpak.DependencyCache, appPath string, manifest *properties.Properties, aotEnabled bool, performanceType SpringPerformanceType, classpathString string, reZip bool, trainingRunJavaToolOptions string) SpringPerformance {
	contributor := libpak.NewLayerContributor("Performance", cache, libcnb.LayerTypes{
		Build:  true,
		Launch: true,
	})
	return SpringPerformance{
		LayerContributor:           contributor,
		Executor:                   effect.NewExecutor(),
		AppPath:                    appPath,
		Manifest:                   manifest,
		AotEnabled:                 aotEnabled,
		PerformanceType:            performanceType,
		TrainingRunJavaToolOptions: trainingRunJavaToolOptions,
		ClasspathString:            classpathString,
		ReZip:                      reZip,
	}
}

func (s SpringPerformance) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	s.LayerContributor.Logger = s.Logger
	layer, err := s.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {

		layer.LaunchEnvironment.Default("BPL_SPRING_AOT_ENABLED", s.AotEnabled)

		if s.PerformanceType == Without {
			return layer, nil
		} else if s.PerformanceType == CdsAotCache {
			layer.LaunchEnvironment.Default("BPL_JVM_AOTCACHE_ENABLED", true)
		}

		// Check for pre-recorded AOT cache
		preRecordedCache := ""
		cacheFile := filepath.Join(s.AppPath, "aot-cache", "application.aot")
		if info, err := os.Stat(cacheFile); err == nil && s.PerformanceType == CdsAotCache {
			if info.Size() <= 0 {
				// A zero-byte cache means nothing was recorded (the JVM can emit an empty
				// application.aot when there is nothing to cache). Treat it as if no
				// pre-recorded cache exists and fall through to the training run, which
				// runs exactly once and is not a retry.
				s.Logger.Bodyf("Ignoring empty pre-recorded AOT cache at %s", cacheFile)
			} else if jreVersion, err := JavaMajorVersionFromJRE(s.Executor); err != nil {
				return layer, fmt.Errorf("error extracting finding out Java Version\n%w", err)
			} else if jreVersion < 24 {
				// -XX:AOTCache, which loads a cache, arrived in Java 24 with
				// https://openjdk.org/jeps/483; older JREs get the CDS training run instead
				s.Logger.Bodyf("Ignoring pre-recorded AOT cache at %s: loading one needs Java 24 or later, this image runs Java %d", cacheFile, jreVersion)
			} else {
				// Pre-recorded cache exists — skip training and use it directly. Before
				// accepting it, verify it was produced for the JRE baked into the image so it
				// is not reused across a mismatched JDK or platform (AOT caches are JDK- and
				// platform-specific).
				layer.Launch = true

				meta, present, err := loadAotCacheMetadata(cacheFile)
				if err != nil {
					// the sidecar is written by whatever recorded the cache, so it is advisory:
					// an unreadable one leaves the cache unverified rather than failing the build
					s.Logger.Bodyf("Could not read AOT cache metadata: %s", err)
					present = false
				}
				if !present {
					// Existing workloads may record the cache without metadata. We cannot verify
					// compatibility, so warn and proceed rather than break the build.
					s.Logger.Bodyf("Pre-recorded AOT cache at %s has no metadata; skipping compatibility verification", cacheFile)
				} else {
					jre, err := JREPropertiesFromJRE(s.Executor)
					if err != nil {
						s.Logger.Bodyf("Could not verify AOT cache compatibility: %s", err)
					} else {
						compatible, reason := validateAotCacheMetadata(meta, jre)
						if !compatible {
							return layer, fmt.Errorf("%s", reason)
						}
						s.Logger.Bodyf("Verified pre-recorded AOT cache at %s matches the image JRE (%s, %s)", cacheFile, jre.JavaVersion, jre.OsArch)
					}
				}

				cacheFileHandle, err := os.Open(cacheFile)
				if err != nil {
					return layer, fmt.Errorf("error opening AOT cache file\n%w", err)
				}
				defer cacheFileHandle.Close()
				stashDir, err := os.MkdirTemp("", "pre-recorded-aot-cache")
				if err != nil {
					return layer, fmt.Errorf("error creating temp directory for AOT cache file\n%w", err)
				}
				preRecordedCache = filepath.Join(stashDir, "application.aot")
				if err := sherpa.CopyFile(cacheFileHandle, preRecordedCache); err != nil {
					return layer, fmt.Errorf("error copying AOT cache file\n%w", err)
				}
				// aot-cache is build input rather than application content: dropping it keeps it
				// out of runner.jar and stops it being shipped a second time in the app layer
				if err := os.RemoveAll(filepath.Dir(cacheFile)); err != nil {
					return layer, fmt.Errorf("error removing %s\n%w", filepath.Dir(cacheFile), err)
				}
			}
		}

		// prepare the training run JVM opts
		var trainingRunArgs []string

		if s.AotEnabled {
			trainingRunArgs = append(trainingRunArgs, "-Dspring.aot.enabled=true")
		}

		jarPath := s.AppPath

		if s.ReZip {
			jarDestDir := os.TempDir() + "/" + fmt.Sprint(time.Now().UnixMilli()) + "/jar-dest"
			if err := os.MkdirAll(jarDestDir, 0755); err != nil {
				return layer, fmt.Errorf("error creating temp directory for jar\n%w", err)
			}
			tempJarPath := filepath.Join(jarDestDir, "runner.jar")
			if err := crush.CreateJar(s.AppPath+"/", tempJarPath); err != nil {
				return layer, fmt.Errorf("error recreating jar\n%w", err)
			}
			f, err := os.Open(tempJarPath)
			if err != nil {
				return layer, fmt.Errorf("error opening jar\n%w", err)
			}
			if err = sherpa.CopyFile(f, filepath.Join(layer.Path, "runner.jar")); err != nil {
				return layer, fmt.Errorf("error copying jar\n%w", err)
			}

			jarPath = tempJarPath
			os.RemoveAll(s.AppPath)
		}

		javaCommand := JavaCommand()

		if err := s.springBootJarLayoutExtract(javaCommand, jarPath); err != nil {
			return layer, fmt.Errorf("error extracting Boot jar at %s\n%w", jarPath, err)
		}

		if s.PerformanceType == ExtractLayout {
			return layer, nil
		}

		startClassValue, _ := s.Manifest.Get("Start-Class")

		if err := fs.WalkDir(os.DirFS(s.AppPath), ".", func(path string, d fs.DirEntry, err error) error {
			if baseTime, err := time.Parse(time.DateTime, "1980-01-01 00:00:01"); err != nil {
				return fmt.Errorf("error parsing date-time\n%w", err)
			} else if err := os.Chtimes(path, baseTime, baseTime); err != nil {
				return fmt.Errorf("error resetting file times\n%w", err)
			}
			return nil
		}); err != nil {
			return libcnb.Layer{}, err
		}

		// Use the pre-recorded AOT cache, if there is one, instead of the training run. It has to
		// happen here rather than earlier: the launch process type runs from the extracted layout,
		// and an AOT cache records the timestamp of every classpath entry it was recorded against,
		// so runner.jar has to look exactly as it did then.
		if preRecordedCache != "" {
			layerPath := filepath.Join(layer.Path, "application.aot")
			f, err := os.Open(preRecordedCache)
			if err != nil {
				return layer, fmt.Errorf("error opening AOT cache file\n%w", err)
			}
			defer f.Close()
			if err := sherpa.CopyFile(f, layerPath); err != nil {
				return layer, fmt.Errorf("error writing AOT cache file to layer\n%w", err)
			}
			// -XX:AOTCache on its own is lenient: the JVM only warns and carries on when it cannot
			// map the cache. -XX:AOTMode=on makes it fail, so a cache that does not belong to this
			// image is caught here instead of silently shipping as dead weight. The launch
			// environment keeps the lenient flag, so a surprise at runtime costs the optimization
			// rather than the application.
			if err := s.Executor.Execute(effect.Execution{
				Command: javaCommand,
				Args:    []string{"-XX:AOTMode=on", fmt.Sprintf("-XX:AOTCache=%s", layerPath), "-cp", s.ClasspathString, "-version"},
				Dir:     s.AppPath,
				Stdout:  s.Logger.InfoWriter(),
				Stderr:  s.Logger.InfoWriter(),
			}); err != nil {
				return layer, fmt.Errorf("the image JRE refused to load the pre-recorded AOT cache at %s\n"+
					"a cache only loads on the JDK version and architecture that recorded it, and with the "+
					"classpath it was recorded with, which here is %q\n%w", cacheFile, s.ClasspathString, err)
			}
			layer.LaunchEnvironment.Default("BPL_JVM_AOTCACHE", layerPath)
			return layer, nil
		}

		jreVersion, err := JavaMajorVersionFromJRE(s.Executor)
		if err != nil {
			return layer, fmt.Errorf("error extracting finding out Java Version\n%w", err)
		}

		trainingRunArgs = append(trainingRunArgs, "-Dspring.context.exit=onRefresh")
		if jreVersion >= 25 {
			// we can use https://openjdk.org/jeps/514
			trainingRunArgs = append(trainingRunArgs, "-XX:AOTCacheOutput=application.aot")
		} else {
			trainingRunArgs = append(trainingRunArgs, "-XX:ArchiveClassesAtExit=application.jsa")
		}
		trainingRunArgs = append(trainingRunArgs, "-cp", s.ClasspathString, startClassValue)

		var trainingRunEnvVariables []string

		if s.TrainingRunJavaToolOptions != "" {
			s.Logger.Bodyf("Training run will use this value as JAVA_TOOL_OPTIONS: %s", s.TrainingRunJavaToolOptions)
			trainingRunEnvVariables = append(trainingRunEnvVariables, fmt.Sprintf("JAVA_TOOL_OPTIONS=%s", s.TrainingRunJavaToolOptions))
		}

		// perform the training run, application.dsa or .aot, the cache file, will be created
		if err := s.Executor.Execute(effect.Execution{
			Command: javaCommand,
			Env:     trainingRunEnvVariables,
			Args:    trainingRunArgs,
			Dir:     s.AppPath,
			Stdout:  s.Logger.InfoWriter(),
			Stderr:  s.Logger.InfoWriter(),
		}); err != nil {
			return libcnb.Layer{}, fmt.Errorf("error running build\n%w", err)
		}

		return layer, nil
	})

	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to contribute spring-performance layer\n%w", err)
	}
	return layer, nil
}

func (s SpringPerformance) Name() string {
	return s.LayerContributor.Name
}

func (s SpringPerformance) springBootJarLayoutExtract(javaCommand string, jarPath string) error {
	s.Logger.Bodyf("Extracting Jar")
	if err := s.Executor.Execute(effect.Execution{
		Command: javaCommand,
		Args:    []string{"-Djarmode=tools", "-jar", jarPath, "extract", "--destination", s.AppPath},
		Dir:     filepath.Dir(jarPath),
		Stdout:  s.Logger.InfoWriter(),
		Stderr:  s.Logger.InfoWriter(),
	}); err != nil {
		return fmt.Errorf("error extracting Jar with jarmode\n%w", err)
	}
	return nil
}
