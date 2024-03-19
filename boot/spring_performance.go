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
	"archive/zip"
	"fmt"

	"github.com/paketo-buildpacks/libpak/sherpa"

	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/magiconair/properties"
	"github.com/paketo-buildpacks/spring-boot/v5/helper"
	"gopkg.in/yaml.v3"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
)

type SpringPerformance struct {
	Dependency       libpak.BuildpackDependency
	LayerContributor libpak.LayerContributor
	Logger           bard.Logger
	Executor         effect.Executor
	AppPath          string
	Manifest         *properties.Properties
	AotEnabled       bool
	DoTrainingRun    bool
	ClasspathString  string
	ReZip            bool
}

func NewSpringPerformance(cache libpak.DependencyCache, appPath string, manifest *properties.Properties, aotEnabled bool, doTrainingRun bool, classpathString string, reZip bool) SpringPerformance {
	contributor := libpak.NewLayerContributor("Performance", cache, libcnb.LayerTypes{
		Build:  true,
		Launch: true,
	})
	return SpringPerformance{
		LayerContributor: contributor,
		Executor:         effect.NewExecutor(),
		AppPath:          appPath,
		Manifest:         manifest,
		AotEnabled:       aotEnabled,
		DoTrainingRun:    doTrainingRun,
		ClasspathString:  classpathString,
		ReZip:            reZip,
	}
}

func (s SpringPerformance) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	s.LayerContributor.Logger = s.Logger
	layer, err := s.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {

		s.Logger.Body("Those are the files we have in the workspace")
		helper.StartOSCommand("", "ls", "-al", s.AppPath)

		// prepare the training run JVM opts
		var trainingRunArgs []string
		jarPath := s.AppPath

		if s.AotEnabled || s.DoTrainingRun {
			layer.LaunchEnvironment.Default("BPL_SPRING_AOT_ENABLED", s.AotEnabled)
			if s.DoTrainingRun {
				trainingRunArgs = append(trainingRunArgs, fmt.Sprintf("-Dspring.aot.enabled=%t", s.AotEnabled))
				layer.LaunchEnvironment.Default("BPL_JVM_CDS_ENABLED", "true")
			} else {
				return layer, nil
			}
		} else {
			return layer, nil
		}

		s.Logger.Bodyf("Are we gonna rezip? : %t", s.ReZip)
		if s.ReZip {
			jarDestDir := os.TempDir() + "/" + fmt.Sprint(time.Now().UnixMilli()) + "/jar-dest"
			if err := os.MkdirAll(jarDestDir, 0755); err != nil {
				return layer, fmt.Errorf("error creating temp directory for jar\n%w", err)
			}
			tempJarPath := filepath.Join(jarDestDir, "runner.jar")
			helper.StartOSCommand("", "ls", "-al", s.AppPath)
			helper.StartOSCommand("", "ls", "-al", filepath.Join(s.AppPath, "BOOT-INF", "lib"))
			if err := s.createJar(s.AppPath+"/", tempJarPath); err != nil {
				return layer, fmt.Errorf("error recreating jar\n%w", err)
			}
			f, err := os.Open(tempJarPath)
			if err != nil {
				return layer, fmt.Errorf("error opening jar\n%w", err)
			}
			if err := sherpa.CopyFile(f, filepath.Join(layer.Path, "runner.jar")); err != nil {
				return layer, fmt.Errorf("error copying jar\n%w", err)
			}

			//return layer, nil
			jarPath = tempJarPath
			os.RemoveAll(s.AppPath)
		}
		s.Logger.Body("We're gonna extract this one:")
		helper.StartOSCommand("", "ls", "-al", jarPath)
		if err := s.springBootJarCDSLayoutExtract(jarPath); err != nil {
			return layer, fmt.Errorf("error extracting Boot jar at %s\n%w", jarPath, err)
		}
		// if unpack == true, follow this (with any updates needed), else (3.3+) run new command, +- touch file timestamps? + skip to training run section below
		startClassValue, _ := s.Manifest.Get("Start-Class")
		s.Logger.Bodyf("This is the value of AppPath: %s", s.AppPath)
		s.Logger.Body("Those are the files we have in the workspace")
		helper.StartOSCommand("", "ls", "-al", s.AppPath)

		/*
			// we extract the vital information from the Spring Boot app manifest
			implementationTitle, okIT := s.Manifest.Get("Implementation-Title")
			implementationValue, okIV := s.Manifest.Get("Implementation-Version")
			startClassValue, okSC := s.Manifest.Get("Start-Class")
			classpathIndex, okSI := s.Manifest.Get("Spring-Boot-Classpath-Index")
			if !(okIT && okIV && okSC && okSI) {
				return layer, fmt.Errorf("unable to contribute spring-cds layer - " +
					"missing Spring Boot Manifest entries, Implementation-Title or Implementation-Version" +
					"or Start-Class or Spring-Boot-Classpath-Index")
			}

			// the spring boot jar is already unzipped
			originalJarExplodedDirectory := s.AppPath

			// we prepare a location for the unpacked version
			targetUnpackedDirectory := os.TempDir() + "/" + fmt.Sprint(time.Now().UnixMilli()) + "/unpacked"
			os.MkdirAll(targetUnpackedDirectory+"/application", 0755)
			os.MkdirAll(targetUnpackedDirectory+"/dependencies", 0755)

			// we create the application jar: the one that contains the user classes
			jarName := implementationTitle + "-" + implementationValue + ".jar"
			createJar(originalJarExplodedDirectory+"/BOOT-INF/classes/", targetUnpackedDirectory+"/application/"+jarName)

			s.Logger.Bodyf("Those are the files we have in the target folder %s", targetUnpackedDirectory+"/dependencies")
			helper.StartOSCommand("", "ls", "-al", targetUnpackedDirectory+"/dependencies")

			// we prepare and create the MANIFEST.MF of the runner-app.jar
			tempDirectory := os.TempDir() + "/" + fmt.Sprint(time.Now().UnixMilli()) + "/"
			os.MkdirAll(tempDirectory+"/META-INF/", 0755)
			runAppJarManifest, _ := os.Create(tempDirectory + "/META-INF/MANIFEST.MF")
			s.writeRunAppJarManifest(originalJarExplodedDirectory, runAppJarManifest, "application/"+jarName, startClassValue, classpathIndex)

			s.Logger.Bodyf("that's the run-app.jar manifest:\n")
			helper.StartOSCommand("", "cat", runAppJarManifest.Name())
			s.Logger.Bodyf("That's the jar we're building: %s in %s", filepath.Dir(runAppJarManifest.Name()), targetUnpackedDirectory+"/run-app.jar")

			// we create the runner-app.jar that will contain just its MANIFEST
			createJar(filepath.Dir(runAppJarManifest.Name()), targetUnpackedDirectory+"/runner.jar")
			//helper.StartOSCommand("", "unzip", "-t", targetUnpackedDirectory+"/run-app.jar")
			//helper.StartOSCommand("", "jar", "cfm", targetUnpackedDirectory+"/run-app.jar", runAppJarManifest.Name())

			// we copy all the dependencies libs from the original jar to the dependencies/folder
			sherpa.CopyDir(originalJarExplodedDirectory+"/BOOT-INF/lib/", targetUnpackedDirectory+"/dependencies/")
		*/

		// we copy the unpack folder to the app path, so that it'll be kept in the layer
		//		sherpa.CopyDir(targetUnpackedDirectory, s.AppPath)

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

		trainingRunArgs = append(trainingRunArgs,
			"-Dspring.context.exit=onRefresh",
			"-XX:ArchiveClassesAtExit=application.jsa",
			"-cp",
		)

		trainingRunArgs = append(trainingRunArgs, s.ClasspathString)
		trainingRunArgs = append(trainingRunArgs, startClassValue)

		s.Logger.Bodyf("training args %s", strings.Join(trainingRunArgs, ", "))

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
		return libcnb.Layer{}, fmt.Errorf("unable to contribute spring-cds layer\n%w", err)
	}
	return layer, nil
}

// OldContribute TODO: this function could still be interesting when an unpack'ed Spring Boot app was provided (run-app.jar was found)
func (s SpringPerformance) OldContribute(layer libcnb.Layer) (libcnb.Layer, error) {
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
		return libcnb.Layer{}, fmt.Errorf("unable to contribute spring-cds layer\n%w", err)
	}
	return layer, nil
}

func (s SpringPerformance) Name() string {
	return s.LayerContributor.Name
}

func (s SpringPerformance) springBootJarCDSLayoutExtract(jarPath string) error {
	s.Logger.Bodyf("Extracting Jar")
	if err := effect.NewExecutor().Execute(effect.Execution{
		Command: "java",
		Args:    []string{"-Djarmode=tools", "-jar", jarPath, "extract", "--destination", s.AppPath},
		Dir:     filepath.Dir(jarPath),
		Stdout:  s.Logger.InfoWriter(),
		Stderr:  s.Logger.InfoWriter(),
	}); err != nil {
		return fmt.Errorf("error extracting Jar with jarmode\n%w", err)
	}
	return nil
}

func (s SpringPerformance) createJar(jarSource string, jarDest string) error {
	s.Logger.Bodyf("Creating Jar")
	if err := effect.NewExecutor().Execute(effect.Execution{
		Command: "jar",
		Args:    []string{"-cfm0", jarDest, filepath.Join(jarSource, "META-INF", "MANIFEST.MF"), "."},
		Dir:     filepath.Dir(jarSource),
		Stdout:  s.Logger.InfoWriter(),
		Stderr:  s.Logger.InfoWriter(),
	}); err != nil {
		return fmt.Errorf("error creating Jar\n%w", err)
	}
	return nil
}

func (s SpringPerformance) writeRunAppJarManifest(originalJarExplodedDirectory string, runAppJarManifest *os.File, relocatedOriginalJar string, startClassValue string, classpathIdx string) {
	classPathValue, _ := retrieveClasspathFromIdx(originalJarExplodedDirectory, "dependencies/", relocatedOriginalJar, classpathIdx)

	//for _, lib :=  range s.AdditionalLibs {
	//	classPathValue = strings.Join([]string{classPathValue, fmt.Sprintf("dependencies/%s", lib)}, " ")
	//}

	type Manifest struct {
		MainClass string
		ClassPath string
	}

	manifestValues := Manifest{
		rewriteWithMaxLineLength("Main-Class: "+startClassValue, 72),
		rewriteWithMaxLineLength("Class-Path: "+classPathValue, 72),
	}
	tmpl, err := template.New("manifest").Parse(
		"Manifest-Version: 1.0\r\n" +
			"{{.MainClass}}\r\n" +
			"{{.ClassPath}}\r\n" +
			"\r\n")
	if err != nil {
		panic(err)
	}
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
			// this is no mistake here! Java won't open a Jar with a \n only MANIFEST!
			result = result + currentLine + "\r\n"
			currentLine = " "
			indent = indent + 1
			remainder = " "
		}
	}
	result = result + remainder
	return result
}
func retrieveClasspathFromIdx(dir string, relocatedDir string, relocatedOriginalJar string, classpathIdx string) (string, error) {
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



// heavily inspired by: https://gosamples.dev/zip-file/
func CreateJar(source, target string) error {

	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	// Register a custom Deflate compressor.
	//writer.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
	//	return flate.NewWriter(out, flate.NoCompression)
	//})
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		absolutePath := ""

		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			if absolutePath, err = filepath.EvalSymlinks(path); err != nil {
				return fmt.Errorf("unable to eval symlink %s\n%w", absolutePath, err)
			}
			if file, err := os.Open(absolutePath); err != nil {
				return fmt.Errorf("unable to open %s\n%w", absolutePath, err)
			} else {
				if info, err = file.Stat(); err != nil {
					return fmt.Errorf("unable to stat %s\n%w", absolutePath, err)
				}
			}
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Store
		

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if absolutePath != "" {
			path = absolutePath
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		writer.Flush()
		return err
	})
	
}
