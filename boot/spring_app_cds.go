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
	"github.com/magiconair/properties"
	"github.com/paketo-buildpacks/libjvm"
	"github.com/paketo-buildpacks/libpak/sherpa"
	"github.com/paketo-buildpacks/spring-boot/v5/helper"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
)

type SpringAppCDS struct {
	Dependency       libpak.BuildpackDependency
	LayerContributor libpak.LayerContributor
	Logger           bard.Logger
	Executor         effect.Executor
	AppPath          string
	Manifest         *properties.Properties
	AotEnabled       bool
}

func NewSpringAppCDS(cache libpak.DependencyCache, appPath string, manifest *properties.Properties, aotEnabled bool) SpringAppCDS {
	contributor := libpak.NewLayerContributor("spring-app-cds", cache, libcnb.LayerTypes{
		Build: true,
	})
	return SpringAppCDS{
		LayerContributor: contributor,
		Executor:         effect.NewExecutor(),
		AppPath:          appPath,
		Manifest:         manifest,
		AotEnabled:       aotEnabled,
	}
}

func (s SpringAppCDS) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	s.LayerContributor.Logger = s.Logger
	layer, err := s.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {

		s.Logger.Bodyf("This is the value of AppPath: %s", s.AppPath)
		s.Logger.Body("Those are the files we have in the workspace")
		helper.StartOSCommand("", "ls", "-al", s.AppPath)

		// we extract the vital information from the Spring Boot app manifest
		implementationTitle, okIT := s.Manifest.Get("Implementation-Title")
		implementationValue, okIV := s.Manifest.Get("Implementation-Version")
		startClassValue, okSC := s.Manifest.Get("Start-Class")
		classpathIndex, okSI := s.Manifest.Get("Spring-Boot-Classpath-Index")
		if !(okIT && okIV && okSC && okSI) {
			return layer, fmt.Errorf("unable to contribute spring-app-cds layer - " +
				"missing Spring Boot Manifest entries, Implementation-Title or Implementation-Version" +
				"or Start-Class or Spring-Boot-Classpath-Index\n")
		}

		// the spring boot jar is already unzipped
		originalJarExplodedDirectory := s.AppPath

		// we prepare a location for the unpacked version
		targetUnpackedDirectory := os.TempDir() + "/" + fmt.Sprint(time.Now().UnixMilli()) + "/unpacked"
		os.MkdirAll(targetUnpackedDirectory, 0755)
		os.MkdirAll(targetUnpackedDirectory+"/application", 0755)
		os.MkdirAll(targetUnpackedDirectory+"/dependencies", 0755)

		// we create the application jar: the one that contains the user classes
		jarName := implementationTitle + "-" + implementationValue + ".jar"
		helper.StartOSCommand("", "jar", "cf", targetUnpackedDirectory+"/application/"+jarName, "-C", originalJarExplodedDirectory+"/BOOT-INF/classes/", ".")

		s.Logger.Bodyf("Those are the files we have in the target folder %s", targetUnpackedDirectory)
		helper.StartOSCommand("", "ls", "-al", targetUnpackedDirectory)

		// we prepare and create the MANIFEST.MF of the runner-app.jar
		tempDirectory := os.TempDir() + "/" + fmt.Sprint(time.Now().UnixMilli()) + "/"
		os.MkdirAll(tempDirectory+"/META-INF/", 0755)
		runAppJarManifest, _ := os.Create(tempDirectory + "/META-INF/MANIFEST.MF")
		// TODO: it should be possible to rather use the JDK Created-by
		const createdBy = "17.9.9 (Spring Boot Paketo Buildpack)"
		writeRunAppJarManifest(originalJarExplodedDirectory, runAppJarManifest, "application/"+jarName, createdBy, startClassValue, classpathIndex)

		// we create the runner-app.jar that will contain just its MANIFEST
		helper.StartOSCommand("", "jar", "cfm", targetUnpackedDirectory+"/run-app.jar", runAppJarManifest.Name())

		// we copy all the dependencies libs from the original jar to the dependencies/folder
		sherpa.CopyDir(originalJarExplodedDirectory+"/BOOT-INF/lib/", targetUnpackedDirectory+"/dependencies/")

		// we discard the original Spring Boot app jar
		os.RemoveAll(s.AppPath)

		// we copy the unpack folder to the app path, so that it'll be kept in the layer
		sherpa.CopyDir(targetUnpackedDirectory, s.AppPath)

		// we set the creation date to the buildpack default 1980/01/01 date; so that cds will be fine
		if err := s.Executor.Execute(effect.Execution{
			Command: "find",
			Env:     []string{"TZ=UTC"},
			Args:    []string{"./", "-exec", "touch", "-t", "198001010000.01", "{}", ";"},
			Dir:     s.AppPath,
			Stdout:  s.Logger.InfoWriter(),
			Stderr:  s.Logger.InfoWriter(),
		}); err != nil {
			return libcnb.Layer{}, fmt.Errorf("error running build\n%w", err)
		}

		// prepare the training run JVM opts
		var trainingRunArgs []string
		if s.AotEnabled {
			trainingRunArgs = append(trainingRunArgs, "-Dspring.aot.enabled=true")
		}
		trainingRunArgs = append(trainingRunArgs,
			"-Dspring.context.exit=onRefresh",
			"-XX:ArchiveClassesAtExit=application.jsa",
			"-jar", "run-app.jar")

		// perform the training run, application.dsa, the cache file, will be created
		if err := s.Executor.Execute(effect.Execution{
			Command: "java",
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
		return libcnb.Layer{}, fmt.Errorf("unable to contribute spring-app-cds layer\n%w", err)
	}
	return layer, nil
}

// OldContribute TODO: this function could still be interesting when an unpack'ed Spring Boot app was provided (run-app.jar was found)
func (s SpringAppCDS) OldContribute(layer libcnb.Layer) (libcnb.Layer, error) {
	s.LayerContributor.Logger = s.Logger
	layer, err := s.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {
		s.Logger.Body("Those are the files we have in the workspace BEFORE the training run", layer.Path)
		if err := s.Executor.Execute(effect.Execution{
			Command: "find",
			Env:     []string{"TZ=UTC"},
			Args:    []string{"./", "-exec", "touch", "-t", "198001010000.01", "{}", ";"},
			Dir:     s.AppPath,
			Stdout:  s.Logger.InfoWriter(),
			Stderr:  s.Logger.InfoWriter(),
		}); err != nil {
			return libcnb.Layer{}, fmt.Errorf("error running build\n%w", err)
		}

		if err := s.Executor.Execute(effect.Execution{
			Command: "java",
			Args: []string{"-Dspring.aot.enabled=true",
				"-Dspring.context.exit=onRefresh",
				"-XX:ArchiveClassesAtExit=application.jsa",
				"-jar", "run-app.jar"},
			Dir:    s.AppPath,
			Stdout: s.Logger.InfoWriter(),
			Stderr: s.Logger.InfoWriter(),
		}); err != nil {
			return libcnb.Layer{}, fmt.Errorf("error running build\n%w", err)
		}
		return layer, nil
	})
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to contribute spring-app-cds layer\n%w", err)
	}
	return layer, nil
}

func (s SpringAppCDS) Name() string {
	return s.LayerContributor.Name
}

func writeRunAppJarManifest(originalJarExplodedDirectory string, runAppJarManifest *os.File, relocatedOriginalJar string, createdBy string, startClassValue string, classpathIdx string) {
	originalManifest, _ := libjvm.NewManifest(originalJarExplodedDirectory)
	classPathValue, _ := retrieveClasspathFromIdx(originalManifest, originalJarExplodedDirectory, "dependencies/", relocatedOriginalJar, classpathIdx)

	type Manifest struct {
		MainClass string
		ClassPath string
		CreatedBy string
	}

	manifestValues := Manifest{startClassValue, rewriteWithMaxLineLength("Class-Path: "+classPathValue, 72), createdBy}
	tmpl, err := template.New("manifest").Parse("Manifest-Version: 1.0\n" +
		"Main-Class: {{.MainClass}}\n" +
		"{{.ClassPath}}\n" +
		"Created-By: {{.CreatedBy}}\n" +
		" ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("that's my manifest: %s", manifestValues)
	//buf := &bytes.Buffer{}
	err = tmpl.Execute(runAppJarManifest, manifestValues)
	if err != nil {
		panic(err)
	}
}

func rewriteWithMaxLineLength(s string, length int) string {
	result := ""
	currentLine := ""
	indent := 0
	remainder := ""

	for i, r := range s {
		currentLine = currentLine + string(r)
		remainder = remainder + string(r)
		j := i + 1
		if indent > 0 {
			j = i + 1 + indent
		}
		if i > 0 && j%length == 0 {
			result = result + currentLine + "\n"
			currentLine = " "
			indent = indent + 1
			remainder = " "
		}
	}
	result = result + remainder
	return result
}
func retrieveClasspathFromIdx(manifest *properties.Properties, dir string, relocatedDir string, relocatedOriginalJar string, classpathIdx string) (string, error) {
	file := filepath.Join(dir, classpathIdx)
	in, err := os.Open(filepath.Join(dir, classpathIdx))
	if err != nil {
		return "", fmt.Errorf("unable to open %s\n%w", file, err)
	}
	defer in.Close()

	var libs []string
	if err := yaml.NewDecoder(in).Decode(&libs); err != nil {
		return "", fmt.Errorf("unable to decode %s\n%w", file, err)
	}

	var relocatedLibs []string
	relocatedLibs = append(relocatedLibs, relocatedOriginalJar)
	for _, lib := range libs {
		relocatedLibs = append(relocatedLibs, strings.ReplaceAll(lib, "BOOT-INF/lib/", relocatedDir))
	}

	return strings.Join(relocatedLibs, " "), nil
}
