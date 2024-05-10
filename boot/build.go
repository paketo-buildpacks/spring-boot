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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/paketo-buildpacks/libpak/crush"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/paketo-buildpacks/libpak/sherpa"

	"github.com/Masterminds/semver/v3"
	"github.com/buildpacks/libcnb"
	"github.com/magiconair/properties"
	"github.com/paketo-buildpacks/libjvm"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"gopkg.in/yaml.v3"
)

const (
	LabelSpringBootVersion             = "org.springframework.boot.version"
	LabelImageTitle                    = "org.opencontainers.image.title"
	LabelImageVersion                  = "org.opencontainers.image.version"
	LabelBootConfigurationMetadata     = "org.springframework.boot.spring-configuration-metadata.json"
	LabelDataFlowConfigurationMetadata = "org.springframework.cloud.dataflow.spring-configuration-metadata.json"
	SpringCloudBindingsBoot2           = "1"
	SpringCloudBindingsBoot3           = "2"
)

type Build struct {
	Logger bard.Logger
}

func (b Build) Build(context libcnb.BuildContext) (libcnb.BuildResult, error) {

	result := libcnb.NewBuildResult()
	bootJarFound, reZipExplodedJar := false, false

	manifest, err := libjvm.NewManifest(context.Application.Path)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to read manifest in %s\n%w", context.Application.Path, err)
	}

	trainingRun := sherpa.ResolveBool("BP_JVM_CDS_ENABLED")

	version, versionFound := manifest.Get("Spring-Boot-Version")
	if !versionFound {
		if context.Application.Path, manifest, err = b.findSpringBootExecutableJAR(context.Application.Path); err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to find Spring Boot Executable Jar\n%w", err)
		} else {
			if version, versionFound = manifest.Get("Spring-Boot-Version"); !versionFound {
				// this isn't a boot app, return without printing title
				return libcnb.BuildResult{}, nil
			}
		}
		bootJarFound = true
	}
	mainClass, _ := manifest.Get("Main-Class")

	if trainingRun {
		if bootCDSExtractionSupported(version) {
			reZipExplodedJar = true
		} else {
			b.Logger.Bodyf("You enabled CDS optimization with BP_JVM_CDS_ENABLED=true but your Spring Boot app version is: %s, you need to upgrade to Spring Boot >= 3.3 first!\nCancelling CDS optimization", version)
			trainingRun = false
		}

	}

	b.Logger.Title(context.Buildpack)

	var helpers []string

	dc, err := libpak.NewDependencyCache(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency cache\n%w", err)
	}
	dc.Logger = b.Logger

	cr, err := libpak.NewConfigurationResolver(context.Buildpack, &b.Logger)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create configuration resolver\n%w", err)
	}

	pr := libpak.PlanEntryResolver{Plan: context.Plan}

	dr, err := libpak.NewDependencyResolver(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency resolver\n%w", err)
	}

	// add labels
	result.Labels, err = labels(context.Application.Path, manifest)
	if err != nil {
		return libcnb.BuildResult{}, err
	}

	// add dependencies to BOM
	lib, ok := manifest.Get("Spring-Boot-Lib")
	if !ok {
		return libcnb.BuildResult{}, fmt.Errorf("manifest does not contain Spring-Boot-Lib")
	}

	// gather libraries
	d, err := libjvm.NewMavenJARListing(filepath.Join(context.Application.Path, lib))
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to generate dependencies from %s\n%w", context.Application.Path, err)
	}
	var additionalLibs []string
	var classpathString string

	// Native Image
	buildNativeImage := false
	if n, ok, err := pr.Resolve("spring-boot"); err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve spring-boot plan entry\n%w", err)
	} else if ok {
		if v, ok := n.Metadata["native-image"].(bool); ok {
			buildNativeImage = v
		}
	}

	if _, ok, err := pr.Resolve("native-processed"); err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve native-processed plan entry\n%w", err)
	} else if ok {
		buildNativeImage = true
	}

	if buildNativeImage {
		// set CLASSPATH for native image build
		classpathLayer, err := NewNativeImageClasspath(context.Application.Path, manifest)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to create NativeImageClasspath\n%w", err)
		}
		classpathLayer.Logger = b.Logger
		result.Layers = append(result.Layers, classpathLayer)

		return result, nil

	}

	// Spring Cloud Bindings
	if scbJarFound := FindExistingDependency(d, "spring-cloud-bindings"); scbJarFound {
		b.Logger.Header("A Spring Cloud Bindings library was found in the Spring Boot libs - not adding another one")
	} else if !cr.ResolveBool("BP_SPRING_CLOUD_BINDINGS_DISABLED") {

		var scbVer string
		var scbSet bool
		if scbVer, scbSet = cr.Resolve("BP_SPRING_CLOUD_BINDINGS_VERSION"); !scbSet {
			if scbVer, err = getSCBVersion(version); err != nil {
				return libcnb.BuildResult{}, fmt.Errorf(
					"unable to parse the Spring Boot version from META-INF/MANIFEST.MF. " +
						"Please set BP_SPRING_CLOUD_BINDINGS_VERSION to force a version or " +
						"BP_SPRING_CLOUD_BINDINGS_DISABLED to bypass installing Spring Cloud Bindings")
			}
		}

		dep, err := dr.Resolve("spring-cloud-bindings", scbVer)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to find dependency\n%w", err)
		}

		helpers = append(helpers, "spring-cloud-bindings")

		bindingsLayer, be := NewSpringCloudBindings(filepath.Join(context.Application.Path, lib), dep, dc)
		bindingsLayer.Logger = b.Logger
		result.Layers = append(result.Layers, bindingsLayer)
		result.BOM.Entries = append(result.BOM.Entries, be)

		additionalLibs = append(additionalLibs, filepath.Base(dep.URI))
	}

	dir := filepath.Join(context.Application.Path, "META-INF", "native-image")
	aotEnabled := false
	aotBuild := sherpa.ResolveBool("BP_SPRING_AOT_ENABLED")
	if aotDirExists, _ := sherpa.DirExists(dir); aotDirExists && aotBuild {
		aotEnabled = true
	} else if !aotDirExists && aotBuild {
		b.Logger.Bodyf("unable to find AOT processed dir %s, however BP_SPRING_AOT_ENABLED has been set to true. Ensure that your app is AOT processed", dir)
	}

	if trainingRun || aotEnabled {

		helpers = append(helpers, "performance")

		if trainingRun {
			mainClass, _ = manifest.Get("Start-Class")
			classpathString = "runner.jar"
			if len(additionalLibs) > 0 {
				cpLibs := []string{}
				for _, lib := range additionalLibs {
					cpLibs = append(cpLibs, fmt.Sprintf(":%s", "lib/"+lib))
				}
				classpathString = fmt.Sprintf(classpathString+"%s", strings.Join(cpLibs, ""))
			}
		}

		cdsLayer := NewSpringPerformance(dc, context.Application.Path, manifest, aotEnabled, trainingRun, classpathString, reZipExplodedJar)
		cdsLayer.Logger = b.Logger
		result.Layers = append(result.Layers, cdsLayer)

	}

	result.BOM.Entries = append(result.BOM.Entries, libcnb.BOMEntry{
		Name:     "dependencies",
		Metadata: map[string]interface{}{"layer": "application", "dependencies": d},
		Launch:   true,
	})

	// validate generations
	gv, err := NewGenerationValidator(filepath.Join(context.Buildpack.Path, "spring-generations.toml"))
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create generation validator\n%w", err)
	}
	gv.Logger = b.Logger

	if err := gv.Validate("spring-boot", version); err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to validate spring-boot version\n%w", err)
	}

	// configure JVM for application type
	classes, ok := manifest.Get("Spring-Boot-Classes")
	if !ok {
		return libcnb.BuildResult{}, fmt.Errorf("manifest does not contain Spring-Boot-Classes")
	}
	wr, err := NewWebApplicationResolver(classes, lib)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create WebApplicationTypeResolver\n%w", err)
	}
	at, err := NewWebApplicationType(context.Application.Path, wr)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create WebApplicationType\n%w", err)
	}
	at.Logger = b.Logger
	result.Layers = append(result.Layers, at)

	if !trainingRun {
		// Slices
		if index, ok := manifest.Get("Spring-Boot-Layers-Index"); ok {
			b.Logger.Header("Creating slices from layers index")
			if result, err = b.createSlices(context.Application.Path, index, result); err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("error creating slices\n%w", err)
			}
		}
	}

	if len(helpers) > 0 {
		result = b.contributeHelpers(context, result, helpers)
	}

	if bootJarFound || trainingRun {
		if mainClass != "" {
			result.Processes = append(result.Processes, b.setProcessTypes(mainClass, classpathString)...)
		} else {
			return libcnb.BuildResult{}, fmt.Errorf("error finding Main-Class or Start-Class manifest entry for Process Type")
		}
	}

	return result, nil
}

func labels(jarPath string, manifest *properties.Properties) ([]libcnb.Label, error) {
	var labels []libcnb.Label

	if s, ok := manifest.Get("Spring-Boot-Version"); ok {
		labels = append(labels, libcnb.Label{Key: LabelSpringBootVersion, Value: s})
	}

	if s, ok := manifest.Get("Implementation-Title"); ok {
		labels = append(labels, libcnb.Label{Key: LabelImageTitle, Value: s})
	}

	if s, ok := manifest.Get("Implementation-Version"); ok {
		labels = append(labels, libcnb.Label{Key: LabelImageVersion, Value: s})
	}

	mdLabels, err := configurationMetadataLabels(jarPath, manifest)
	if err != nil {
		return nil, fmt.Errorf("unable to generate data flow configuration metadata\n%w", err)
	}
	labels = append(labels, mdLabels...)

	return labels, nil
}

func configurationMetadataLabels(appDir string, manifest *properties.Properties) ([]libcnb.Label, error) {
	if ok, err := DataFlowConfigurationExists(appDir); !ok || err != nil {
		return []libcnb.Label{}, err
	}

	var labels []libcnb.Label
	md, err := NewConfigurationMetadataFromPath(appDir)
	if err != nil {
		return nil, fmt.Errorf("unable to read configuration metadata from %s\n%w", appDir, err)
	}

	lib, ok := manifest.Get("Spring-Boot-Lib")
	if !ok {
		return nil, errors.New("manifest does not contain Spring-Boot-Lib")
	}
	file := filepath.Join(lib, "*.jar")
	files, err := filepath.Glob(file)
	if err != nil {
		return nil, fmt.Errorf("unable to glob %s\n%w", file, err)
	}

	for _, file := range files {
		jarMD, err := NewConfigurationMetadataFromJAR(file)
		if err != nil {
			return nil, fmt.Errorf("unable to read configuration metadata from %s\n%w", file, err)
		}

		md.Groups = append(md.Groups, jarMD.Groups...)
		md.Properties = append(md.Properties, jarMD.Properties...)
		md.Hints = append(md.Hints, jarMD.Hints...)
	}
	if len(md.Groups) > 0 || len(md.Properties) > 0 || len(md.Hints) > 0 {
		b := &bytes.Buffer{}
		if err := json.NewEncoder(b).Encode(md); err != nil {
			return nil, fmt.Errorf("unable to encode configuration metadata\n%w", err)
		}
		labels = append(labels, libcnb.Label{
			Key:   LabelBootConfigurationMetadata,
			Value: strings.TrimSpace(b.String()),
		})
	}

	md, err = NewDataFlowConfigurationMetadata(appDir, md)
	if err != nil {
		return nil, fmt.Errorf("unable to generate data flow configuration metadata\n%w", err)
	}
	if len(md.Groups) > 0 || len(md.Properties) > 0 || len(md.Hints) > 0 {
		b := &bytes.Buffer{}
		if err := json.NewEncoder(b).Encode(md); err != nil {
			return nil, fmt.Errorf("unable to encode configuration metadata\n%w", err)
		}
		labels = append(labels, libcnb.Label{
			Key:   LabelDataFlowConfigurationMetadata,
			Value: strings.TrimSpace(b.String()),
		})
	}

	return labels, err
}

func calcSize(paths []string) (string, error) {
	var size float64

	for _, path := range paths {
		if err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			size += float64(info.Size())

			return nil
		}); err != nil {
			return "", err
		}
	}

	return friendlySize(size), nil
}

func friendlySize(size float64) string {
	unit := "B"

	if size/1024.0 > 1.0 {
		size /= 1024.0
		unit = "KB"
	}

	if size/1024.0 > 1.0 {
		size /= 1024.0
		unit = "MB"
	}

	if size/1024.0 > 1.0 {
		size /= 1024.0
		unit = "GB"
	}

	return fmt.Sprintf("%0.1f %s", size, unit)
}

func FindExistingDependency(jars []libjvm.MavenJAR, dependencyName string) bool {
	for _, lib := range jars {
		if lib.Name == dependencyName {
			return true
		}
	}
	return false
}

func getSCBVersion(manifestVer string) (string, error) {
	bootTwoConstraint, _ := semver.NewConstraint("<= 3.0.0")
	bv, err := bootVersion(manifestVer)
	if err != nil {
		return SpringCloudBindingsBoot2, err
	}
	if bootTwoConstraint.Check(bv) {
		return SpringCloudBindingsBoot2, nil
	}
	return SpringCloudBindingsBoot3, nil
}

func bootVersion(version string) (*semver.Version, error) {
	pattern := regexp.MustCompile(`[\d]+(?:\.[\d]+(?:\.[\d]+)?)?`)
	bootV := pattern.FindString(version)
	semverBoot, err := semver.NewVersion(bootV)
	if err != nil {
		return nil, fmt.Errorf("unable to parse spring-boot version\n%w", err)
	}
	return semverBoot, nil
}

func (b *Build) createSlices(path string, index string, result libcnb.BuildResult) (libcnb.BuildResult, error) {

	file := filepath.Join(path, index)
	in, err := os.Open(file)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to open %s\n%w", file, err)
	}
	defer in.Close()
	var layers []map[string][]string
	if err := yaml.NewDecoder(in).Decode(&layers); err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to decode %s\n%w", file, err)
	}
	for _, layer := range layers {
		for name, paths := range layer {
			size, err := calcSize(paths)
			if err != nil {
				size = "size unavailable"
			}
			b.Logger.Body(fmt.Sprintf("%s (%s)", name, size))
			result.Slices = append(result.Slices, libcnb.Slice{Paths: paths})
		}
	}

	return result, nil
}

func (b *Build) contributeHelpers(context libcnb.BuildContext, result libcnb.BuildResult, helpers []string) libcnb.BuildResult {
	h, bom := libpak.NewHelperLayer(context.Buildpack, helpers...)
	h.Logger = b.Logger
	result.Layers = append(result.Layers, h)
	result.BOM.Entries = append(result.BOM.Entries, bom)
	return result
}

func (b *Build) setProcessTypes(mainClass string, classpathString string) []libcnb.Process {

	command := "java"
	arguments := []string{}
	if classpathString != "" {
		arguments = append(arguments, "-cp")
		arguments = append(arguments, classpathString)
	}
	arguments = append(arguments, mainClass)

	processes := []libcnb.Process{}
	processes = append(processes,
		libcnb.Process{
			Type:      "spring-boot-app",
			Command:   command,
			Arguments: arguments,
			Direct:    true,
		},
		libcnb.Process{
			Type:      "task",
			Command:   command,
			Arguments: arguments,
			Direct:    true,
		},
		libcnb.Process{
			Type:      "web",
			Command:   command,
			Arguments: arguments,
			Direct:    true,
			Default:   true,
		})
	return processes
}

func (b *Build) findSpringBootExecutableJAR(appPath string) (string, *properties.Properties, error) {

	props := &properties.Properties{}
	jarPath := ""
	stopWalk := errors.New("stop walking")

	fileSystem := os.DirFS(appPath)
	err := fs.WalkDir(fileSystem, ".", func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// make sure it is a file
		if dirEntry.IsDir() {
			return nil
		}

		// make sure it is a JAR file
		if !strings.HasSuffix(path, ".jar") {
			return nil
		}

		fullPath := appPath + "/" + path
		// get the MANIFEST of the JAR file
		props, err = libjvm.NewManifestFromJAR(fullPath)
		if err != nil {
			return fmt.Errorf("unable to load manifest\n%w", err)
		}

		// we take it if it has a Main-Class AND a Spring-Boot-Version
		_, okMC := props.Get("Main-Class")
		_, okSBV := props.Get("Spring-Boot-Version")
		if okMC && okSBV {
			jarPath = fullPath
			return stopWalk
		}

		return nil
	})

	if err != nil && !errors.Is(err, stopWalk) {
		return "", nil, err
	}

	tempExplodedJar := os.TempDir() + "/" + fmt.Sprint(time.Now().UnixMilli()) + "/"

	jar, err := os.Open(jarPath)
	if err != nil {
		return "", nil, err
	}
	defer jar.Close()
	crush.Extract(jar, tempExplodedJar, 0)
	os.RemoveAll(appPath)
	sherpa.CopyDir(tempExplodedJar, appPath)
	jarPath = appPath

	return jarPath, props, nil
}

func bootCDSExtractionSupported(manifestVer string) bool {
	bootThreeThreeConstraint, _ := semver.NewConstraint(">= 3.3.0")
	bv, err := bootVersion(manifestVer)
	if err != nil {
		return false
	}
	if bootThreeThreeConstraint.Check(bv) {
		return true
	}
	return false
}
