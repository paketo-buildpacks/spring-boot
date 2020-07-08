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
	"os"
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/sherpa"

	_ "github.com/paketo-buildpacks/spring-boot/boot/statik"
)

type SpringCloudBindings struct {
	SpringBootLib    string
	LayerContributor libpak.DependencyLayerContributor
	Logger           bard.Logger
}

func NewSpringCloudBindings(
	springBootLib string,
	dependency libpak.BuildpackDependency,
	cache libpak.DependencyCache,
	plan *libcnb.BuildpackPlan) SpringCloudBindings {
	return SpringCloudBindings{
		SpringBootLib:    springBootLib,
		LayerContributor: libpak.NewDependencyLayerContributor(dependency, cache, plan),
	}

}

//go:generate statik -src . -include *.sh
func (s SpringCloudBindings) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	s.LayerContributor.Logger = s.Logger

	jarName := filepath.Base(s.LayerContributor.Dependency.URI)
	jarPath := filepath.Join(layer.Path, jarName)

	layer, err := s.LayerContributor.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
		if err := sherpa.CopyFile(artifact, jarPath); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to copy artifact to %s\n%w", jarName, err)
		}
		layer.Launch = true
		profileScript, err := sherpa.StaticFile("/" + ("enable-bindings.sh"))
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to load %s\n%w", "enable-bindings.sh", err)
		}
		layer.Profile.Add("enable-bindings.sh", profileScript)
		return layer, nil
	})
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to contribute spring-cloud-bindings layer\n%w", err)
	}

	if err := os.MkdirAll(s.SpringBootLib, 0777); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to ensure '%s' exists\n%w", s.SpringBootLib, err)
	}
	if err := os.Symlink(jarPath, filepath.Join(s.SpringBootLib, jarName)); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to link spring cloud bindings into BOOT-INF\n%w", err)
	}
	return layer, nil
}

func (s SpringCloudBindings) Name() string {
	return "spring-cloud-bindings"
}
