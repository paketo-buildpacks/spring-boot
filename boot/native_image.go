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
	"github.com/paketo-buildpacks/libpak/sherpa"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/magiconair/properties"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"gopkg.in/yaml.v3"
)

type NativeImageClasspath struct {
	Logger          bard.Logger
	ApplicationPath string
	Manifest        *properties.Properties
}

func NewNativeImageClasspath(appDir string, manifest *properties.Properties) (NativeImageClasspath, error) {
	return NativeImageClasspath{
		ApplicationPath: appDir,
		Manifest:        manifest,
	}, nil
}

// Contribute appends application classes and entries from classpath.idx to the build time classpath.
//
// In a JVM application the Spring Boot Loader would make these modifications to the classpath when the executable JAR
// or WAR is launched. To set the classpath properly for a native-image build the buildpack must replicate this behavior.
func (n NativeImageClasspath) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	cp, err := n.classpathEntries()
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("failed to compute classpath\n%w", err)
	}
	expectedMetadata := map[string][]string{
		"classpath": cp,
	}

	lc := libpak.NewLayerContributor("Class Path", expectedMetadata, libcnb.LayerTypes{
		Build: true,
	})
	lc.Logger = n.Logger

	return lc.Contribute(layer, func() (libcnb.Layer, error) {
		layer.BuildEnvironment.Append(
			"CLASSPATH",
			string(filepath.ListSeparator),
			strings.Join(cp, string(filepath.ListSeparator)),
		)

		nativeImageArgFile := filepath.Join(n.ApplicationPath, "META-INF", "native-image", "argfile")
		if exists, err := sherpa.Exists(nativeImageArgFile); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to check for native-image arguments file at %s\n%w", nativeImageArgFile, err)
		} else if exists {
			lc.Logger.Bodyf(fmt.Sprintf("native args file %s", nativeImageArgFile))
			layer.BuildEnvironment.Default("BP_NATIVE_IMAGE_BUILD_ARGUMENTS_FILE", nativeImageArgFile)
		}

		return layer, nil
	})
}

func (NativeImageClasspath) Name() string {
	return "Class Path"
}

func (n NativeImageClasspath) classpathEntries() ([]string, error) {
	var cp []string

	classesDir, ok := n.Manifest.Get("Spring-Boot-Classes")
	if !ok {
		return nil, fmt.Errorf("manifest does not contain Spring-Boot-Classes")
	}
	cp = append(cp, filepath.Join(n.ApplicationPath, classesDir))

	classpathIdx, ok := n.Manifest.Get("Spring-Boot-Classpath-Index")
	if !ok {
		return nil, fmt.Errorf("manifest does not contain Spring-Boot-Classpath-Index")
	}

	file := filepath.Join(n.ApplicationPath, classpathIdx)
	in, err := os.Open(filepath.Join(n.ApplicationPath, classpathIdx))
	if err != nil {
		return nil, fmt.Errorf("unable to open %s\n%w", file, err)
	}
	defer in.Close()

	var libs []string
	if err := yaml.NewDecoder(in).Decode(&libs); err != nil {
		return nil, fmt.Errorf("unable to decode %s\n%w", file, err)
	}

	libDir, ok := n.Manifest.Get("Spring-Boot-Lib")
	if !ok {
		return nil, fmt.Errorf("manifest does not contain Spring-Boot-Lib")
	}

	for _, l := range libs {
		if dir, _ := filepath.Split(l); dir == "" {
			// In Spring Boot version 2.3.0.M4 -> 2.4.2 classpath.idx contains a list of jars
			cp = append(cp, filepath.Join(n.ApplicationPath, libDir, l))
		} else {
			// In Spring Boot version <= 2.3.0.M3 or >= 2.4.2 classpath.idx contains a list of relative paths to jars
			cp = append(cp, filepath.Join(n.ApplicationPath, l))
		}
	}
	return cp, nil
}
