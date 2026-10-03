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
	"strings"
	"time"

	"github.com/buildpacks/libcnb"
	"github.com/magiconair/properties"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
)

// DefaultAotCachePath is where a pre-recorded AOT cache is looked for by default.
const DefaultAotCachePath = "aot-cache/application.aot"

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
	// AotCachePath is where a pre-recorded AOT cache is looked for.
	AotCachePath string
	// AotCachePathExplicit records that the user named the path themselves.
	AotCachePathExplicit bool
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

		// A blank path behaves exactly like an unset one: the default location, and a
		// missing cache there is a fallback rather than a failure.
		aotCachePath, aotCachePathExplicit := s.AotCachePath, s.AotCachePathExplicit
		if strings.TrimSpace(aotCachePath) == "" {
			aotCachePath, aotCachePathExplicit = DefaultAotCachePath, false
		}

		switch s.PerformanceType {
		case Without:
			return layer, nil
		case CdsAotCache:
			layer.LaunchEnvironment.Default("BPL_JVM_AOTCACHE_ENABLED", true)
		}

		// Check for pre-recorded AOT cache
		preRecordedCache := ""
		cacheFile := aotCachePath
		if !filepath.IsAbs(cacheFile) {
			cacheFile = filepath.Join(s.AppPath, cacheFile)
		}
		info, statErr := os.Stat(cacheFile)
		// A path the user named themselves that does not resolve to a cache file is a
		// misconfiguration, not a reason to silently skip the optimization.
		if s.PerformanceType == CdsAotCache && aotCachePathExplicit && (statErr != nil || info.IsDir()) {
			return layer, fmt.Errorf("no pre-recorded AOT cache at %s (from BP_JVM_AOTCACHE_PATH)", cacheFile)
		}
		if statErr == nil && !info.IsDir() && s.PerformanceType == CdsAotCache {
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
				// Pre-recorded cache exists — skip training and use it directly. The JVM is
				// asked to load the cache with -XX:AOTMode=on before the build accepts it, which
				// is the authority on whether the cache belongs to this image (it only loads on
				// the JDK version and architecture that recorded it, with the classpath it was
				// recorded with).
				layer.Launch = true
				s.Logger.Bodyf("Found pre-recorded AOT cache at %s, skipping the training run", cacheFile)

				cacheFileHandle, err := os.Open(cacheFile)
				if err != nil {
					return layer, fmt.Errorf("error opening AOT cache file\n%w", err)
				}
				defer func() { _ = cacheFileHandle.Close() }()
				stashDir, err := os.MkdirTemp("", "pre-recorded-aot-cache")
				if err != nil {
					return layer, fmt.Errorf("error creating temp directory for AOT cache file\n%w", err)
				}
				preRecordedCache = filepath.Join(stashDir, "application.aot")
				if err := sherpa.CopyFile(cacheFileHandle, preRecordedCache); err != nil {
					return layer, fmt.Errorf("error copying AOT cache file\n%w", err)
				}
				// The cache is build input rather than application content: dropping it keeps
				// it out of runner.jar and stops it being shipped a second time in the app
				// layer. Only the cache file itself is removed, and only when it sits inside
				// the application directory.
				if rel, err := filepath.Rel(s.AppPath, cacheFile); err == nil && filepath.IsLocal(rel) {
					if err := os.Remove(cacheFile); err != nil {
						return layer, fmt.Errorf("error removing %s\n%w", cacheFile, err)
					}
					// tidy up the directory it came from, only if the cache is the last thing in it
					if dir := filepath.Dir(cacheFile); dir != s.AppPath {
						_ = os.Remove(dir)
					}
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
			// The extracted layout replaces the application directory below. This error is
			// deliberately ignored: where the build user owns the contents but not the
			// directory itself - kpack, for one - the contents are emptied and only the final
			// rmdir is refused, which is all that is needed here. Propagating it would break
			// those builds.
			_ = os.RemoveAll(s.AppPath)
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
			if err != nil {
				return err
			}
			if path == "." {
				// skip walk root, it is not needed and can break some builds eg. kpack
				return nil
			}
			if baseTime, err := time.Parse(time.DateTime, "1980-01-01 00:00:01"); err != nil {
				return fmt.Errorf("error parsing date-time\n%w", err)
			} else if err := os.Chtimes(filepath.Join(s.AppPath, path), baseTime, baseTime); err != nil {
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
			defer func() { _ = f.Close() }()
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
