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
	DoTrainingRun              bool
	ClasspathString            string
	ReZip                      bool
	TrainingRunJavaToolOptions string
}

func NewSpringPerformance(cache libpak.DependencyCache, appPath string, manifest *properties.Properties, aotEnabled bool, doTrainingRun bool, classpathString string, reZip bool, trainingRunJavaToolOptions string) SpringPerformance {
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
		DoTrainingRun:              doTrainingRun,
		TrainingRunJavaToolOptions: trainingRunJavaToolOptions,
		ClasspathString:            classpathString,
		ReZip:                      reZip,
	}
}

func (s SpringPerformance) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	s.LayerContributor.Logger = s.Logger
	layer, err := s.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {

		layer.LaunchEnvironment.Default("BPL_SPRING_AOT_ENABLED", s.AotEnabled)

		if !s.DoTrainingRun {
			return layer, nil
		}

		layer.LaunchEnvironment.Default("BPL_JVM_CDS_ENABLED", s.DoTrainingRun)

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

		javaCommand := "java"
		jreHome := sherpa.GetEnvWithDefault("JRE_HOME", sherpa.GetEnvWithDefault("JAVA_HOME", ""))
		if jreHome != "" {
			javaCommand = jreHome + "/bin/java"
		}

		if err := s.springBootJarCDSLayoutExtract(javaCommand, jarPath); err != nil {
			return layer, fmt.Errorf("error extracting Boot jar at %s\n%w", jarPath, err)
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

		trainingRunArgs = append(trainingRunArgs,
			"-Dspring.context.exit=onRefresh",
			"-XX:ArchiveClassesAtExit=application.jsa",
			"-cp",
		)
		trainingRunArgs = append(trainingRunArgs, s.ClasspathString)
		trainingRunArgs = append(trainingRunArgs, startClassValue)

		var trainingRunEnvVariables []string

		if s.TrainingRunJavaToolOptions != "" {
			s.Logger.Bodyf("Training run will use this value as JAVA_TOOL_OPTIONS: %s", s.TrainingRunJavaToolOptions)
			trainingRunEnvVariables = append(trainingRunEnvVariables, fmt.Sprintf("JAVA_TOOL_OPTIONS=%s", s.TrainingRunJavaToolOptions))
		}

		// perform the training run, application.dsa, the cache file, will be created
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
		return libcnb.Layer{}, fmt.Errorf("unable to contribute spring-cds layer\n%w", err)
	}
	return layer, nil
}

func (s SpringPerformance) Name() string {
	return s.LayerContributor.Name
}

func (s SpringPerformance) springBootJarCDSLayoutExtract(javaCommand string, jarPath string) error {
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
