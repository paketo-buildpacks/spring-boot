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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/heroku/color"
	"github.com/magiconair/properties"
	"github.com/mattn/go-shellwords"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/sherpa"
	"gopkg.in/yaml.v3"
)

type NativeImage struct {
	ApplicationPath  string
	Arguments        []string
	Dependency       libpak.BuildpackDependency
	DependencyCache  libpak.DependencyCache
	Executor         effect.Executor
	LayerContributor libpak.LayerContributor
	Logger           bard.Logger
	Manifest         *properties.Properties
}

func NewNativeImage(applicationPath string, arguments string, dependency libpak.BuildpackDependency,
	cache libpak.DependencyCache, manifest *properties.Properties, plan *libcnb.BuildpackPlan) (NativeImage, error) {

	var err error

	plan.Entries = append(plan.Entries, dependency.AsBuildpackPlanEntry())

	expected := map[string]interface{}{
		"dependency": dependency,
	}

	expected["files"], err = sherpa.NewFileListing(applicationPath)
	if err != nil {
		return NativeImage{}, fmt.Errorf("unable to create file listing for %s\n%w", applicationPath, err)
	}

	expected["arguments"], err = shellwords.Parse(arguments)
	if err != nil {
		return NativeImage{}, fmt.Errorf("unable to parse arguments from %s\n%w", arguments, err)
	}

	return NativeImage{
		ApplicationPath:  applicationPath,
		Arguments:        expected["arguments"].([]string),
		Dependency:       dependency,
		DependencyCache:  cache,
		Executor:         effect.NewExecutor(),
		LayerContributor: libpak.NewLayerContributor("Native Image", expected),
		Manifest:         manifest,
	}, nil

}

func (n NativeImage) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	n.LayerContributor.Logger = n.Logger

	startClass, ok := n.Manifest.Get("Start-Class")
	if !ok {
		return libcnb.Layer{}, fmt.Errorf("manifest does not contain Start-Class")
	}

	layer, err := n.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {
		var cp []string

		s, ok := n.Manifest.Get("Spring-Boot-Classes")
		if !ok {
			return libcnb.Layer{}, fmt.Errorf("manifest does not contain Spring-Boot-Classes")
		}
		cp = append(cp, filepath.Join(n.ApplicationPath, s))

		s, ok = n.Manifest.Get("Spring-Boot-Classpath-Index")
		if !ok {
			return libcnb.Layer{}, fmt.Errorf("manifest does not contain Spring-Boot-Classpath-Index")
		}

		file := filepath.Join(n.ApplicationPath, s)
		in, err := os.Open(file)
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to open %s\n%w", file, err)
		}
		defer in.Close()

		var libs []string
		if err := yaml.NewDecoder(in).Decode(&libs); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to decode %s\n%w", file, err)
		}

		s, ok = n.Manifest.Get("Spring-Boot-Lib")
		if !ok {
			return libcnb.Layer{}, fmt.Errorf("manifest does not contain Spring-Boot-Lib")
		}

		for _, l := range libs {
			cp = append(cp, filepath.Join(n.ApplicationPath, s, l))
		}

		// Pick up /META-INF
		cp = append(cp, n.ApplicationPath)

		n.Logger.Header(color.BlueString("%s %s", n.Dependency.Name, n.Dependency.Version))

		artifact, err := n.DependencyCache.Artifact(n.Dependency)
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to get dependency %s\n%w", n.Dependency.ID, err)
		}
		defer artifact.Close()
		cp = append(cp, artifact.Name())

		arguments := append(n.Arguments,
			fmt.Sprintf("-H:Name=%s", filepath.Join(layer.Path, startClass)),
			"-cp", strings.Join(cp, ":"),
			startClass,
		)

		n.Logger.Bodyf("Executing native-image %s", strings.Join(arguments, " "))
		if err := n.Executor.Execute(effect.Execution{
			Command: "native-image",
			Args:    arguments,
			Dir:     layer.Path,
			Stdout:  n.Logger.InfoWriter(),
			Stderr:  n.Logger.InfoWriter(),
		}); err != nil {
			return libcnb.Layer{}, fmt.Errorf("error running build\n%w", err)
		}

		layer.Cache = true
		return layer, nil
	})
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to contribute native-image layer\n%w", err)
	}

	n.Logger.Header("Removing bytecode")
	cs, err := ioutil.ReadDir(n.ApplicationPath)
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to list children of %s\n%w", n.ApplicationPath, err)
	}
	for _, c := range cs {
		file := filepath.Join(n.ApplicationPath, c.Name())
		if err := os.RemoveAll(file); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to remove %s\n%w", file, err)
		}
	}

	file := filepath.Join(layer.Path, startClass)
	in, err := os.Open(file)
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to open %s\n%w", file, err)
	}
	defer in.Close()

	file = filepath.Join(n.ApplicationPath, startClass)
	out, err := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to open %s\n%w", file, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to copy\n%w", err)
	}

	return layer, nil
}

func (NativeImage) Name() string {
	return "native-image"
}
