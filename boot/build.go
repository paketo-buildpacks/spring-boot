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
	"github.com/paketo-buildpacks/libpak/sherpa"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
	manifest, err := libjvm.NewManifest(context.Application.Path)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to read manifest in %s\n%w", context.Application.Path, err)
	}

	version, ok := manifest.Get("Spring-Boot-Version")
	cds, _ := sherpa.FileExists("run-app.jar")
	result := libcnb.NewBuildResult()

	if cds {
		// cds specific
		b.Logger.Title(context.Buildpack)
		h, be := libpak.NewHelperLayer(context.Buildpack, "spring-class-data-sharing")
		h.Logger = b.Logger

		// add labels
		result.Labels, err = labels(context, manifest)
		if err != nil {
			return libcnb.BuildResult{}, err
		}

		result.Layers = append(result.Layers, h)
		result.BOM.Entries = append(result.BOM.Entries, be)

		dc, err := libpak.NewDependencyCache(context)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency cache\n%w", err)
		}
		dc.Logger = b.Logger
		bindingsLayer := NewSpringClassDataSharing(dc, context.Application.Path)
		bindingsLayer.Logger = b.Logger
		result.Layers = append(result.Layers, bindingsLayer)
		result.BOM.Entries = append(result.BOM.Entries, be)
		return result, nil
	}

	if !ok {
		// this isn't a boot app, return without printing title
		return libcnb.BuildResult{}, nil
	}
	fmt.Printf("passed spring detection")

	b.Logger.Title(context.Buildpack)

	pr := libpak.PlanEntryResolver{Plan: context.Plan}

	dr, err := libpak.NewDependencyResolver(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency resolver\n%w", err)
	}

	cr, err := libpak.NewConfigurationResolver(context.Buildpack, &b.Logger)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create configuration resolver\n%w", err)
	}

	dc, err := libpak.NewDependencyCache(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency cache\n%w", err)
	}
	dc.Logger = b.Logger

	// add labels
	result.Labels, err = labels(context, manifest)
	if err != nil {
		return libcnb.BuildResult{}, err
	}

	// add dependencies to BOM
	lib, ok := manifest.Get("Spring-Boot-Lib")
	if !ok {
		return libcnb.BuildResult{}, fmt.Errorf("manifest does not contain Spring-Boot-Lib")
	}
	d, err := libjvm.NewMavenJARListing(filepath.Join(context.Application.Path, lib))
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to generate dependencies from %s\n%w", context.Application.Path, err)
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

	} else {
		scbJarFound := FindExistingDependency(d, "spring-cloud-bindings")
		if scbJarFound {
			b.Logger.Header("A Spring Cloud Bindings library was found in the Spring Boot libs - not adding another one")
		}
		// contribute Spring Cloud Bindings - false by default
		if !cr.ResolveBool("BP_SPRING_CLOUD_BINDINGS_DISABLED") && !scbJarFound {
			h, be := libpak.NewHelperLayer(context.Buildpack, "spring-cloud-bindings")
			h.Logger = b.Logger
			result.Layers = append(result.Layers, h)
			result.BOM.Entries = append(result.BOM.Entries, be)

			scbVer, scbSet := cr.Resolve("BP_SPRING_CLOUD_BINDINGS_VERSION")
			if !scbSet {
				scbVerFromBoot, err := getSCBVersion(version)
				if err != nil {
					return libcnb.BuildResult{}, fmt.Errorf("Unable to read the Spring Boot version from META-INF/MANIFEST.MF. Please set BP_SPRING_CLOUD_BINDINGS_VERSION to force a version or BP_SPRING_CLOUD_BINDINGS_DISABLED to bypass installing Spring Cloud Bindings")
				}
				scbVer = scbVerFromBoot
			}

			dep, err := dr.Resolve("spring-cloud-bindings", scbVer)
			if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to find dependency\n%w", err)
			}

			bindingsLayer, be := NewSpringCloudBindings(filepath.Join(context.Application.Path, lib), dep, dc)
			bindingsLayer.Logger = b.Logger
			result.Layers = append(result.Layers, bindingsLayer)
			result.BOM.Entries = append(result.BOM.Entries, be)
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

		// slice app dir
		if index, ok := manifest.Get("Spring-Boot-Layers-Index"); ok {
			b.Logger.Header("Creating slices from layers index")

			file := filepath.Join(context.Application.Path, index)
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
		}
	}

	return result, nil
}

func labels(context libcnb.BuildContext, manifest *properties.Properties) ([]libcnb.Label, error) {
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

	mdLabels, err := configurationMetadataLabels(context.Application.Path, manifest)
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
