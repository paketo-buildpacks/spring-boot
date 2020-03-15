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
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libjvm"
	"github.com/paketo-buildpacks/libpak/bard"
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
	if !ok {
		return libcnb.BuildResult{}, nil
	}

	b.Logger.Title(context.Buildpack)
	result := libcnb.BuildResult{}

	entry := libcnb.BuildpackPlanEntry{
		Name:     "spring-boot",
		Version:  version,
		Metadata: map[string]interface{}{},
	}
	result.Plan.Entries = append(result.Plan.Entries, entry)

	if index, ok := manifest.Get("Spring-Boot-Layers-Index"); ok {
		b.Logger.Body("Using layers index")

		layers, err := b.layers(filepath.Join(context.Application.Path, index))
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to read layers index\n%w", err)
		}

		var libs []string
		for _, l := range layers {
			libs = append(libs, filepath.Join(l, "lib"))
		}

		entry.Metadata["dependencies"], err = Dependencies(libs...)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to generate dependencies from %s\n%w", context.Application.Path, err)
		}

		result.Slices, err = IndexSlices(context.Application.Path, layers...)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to generate slices from %s\n%w", context.Application.Path, err)
		}

		return result, nil
	}

	classes, ok := manifest.Get("Spring-Boot-Classes")
	if !ok {
		classes = "BOOT-INF/classes"
	}

	libs, ok := manifest.Get("Spring-Boot-Lib")
	if !ok {
		libs = "BOOT-INF/lib"
	}

	entry.Metadata["dependencies"], err = Dependencies(filepath.Join(context.Application.Path, libs))
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to generate dependencies from %s\n%w", context.Application.Path, err)
	}

	result.Slices, err = ConventionSlices(context.Application.Path, filepath.Join(context.Application.Path, classes), filepath.Join(context.Application.Path, libs))
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to generate slices from %s\n%w", context.Application.Path, err)
	}

	return result, nil
}

func (Build) layers(path string) ([]string, error) {
	in, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open %s\n%w", path, err)
	}
	defer in.Close()

	var layers []string

	root := filepath.Dir(path)
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		layers = append(layers, filepath.Join(root, "layers", scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("unable to read %s\n%w", path, err)
	}

	return layers, nil
}
