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

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/sherpa"
)

type WebApplicationType struct {
	LayerContributor libpak.LayerContributor
	Logger           bard.Logger
	Resolver         WebApplicationTypeResolver
}

func NewWebApplicationType(applicationPath string, resolver WebApplicationTypeResolver) (WebApplicationType, error) {
	var err error

	expected := make(map[string]interface{}, 1)
	expected["files"], err = sherpa.NewFileListing(applicationPath)
	if err != nil {
		return WebApplicationType{}, fmt.Errorf("unable to create file listing for %s\n%w", applicationPath, err)
	}

	contributor := libpak.NewLayerContributor(
		"Web Application Type",
		expected,
		libcnb.LayerTypes{
			Launch: true,
		},
	)
	return WebApplicationType{
		LayerContributor: contributor,
		Resolver:         resolver,
	}, nil
}

func (w WebApplicationType) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	w.LayerContributor.Logger = w.Logger

	return w.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {
		switch w.Resolver.Resolve() {
		case None:
			w.Logger.Body("Non-web application detected")
			layer.LaunchEnvironment.Default("BPL_JVM_THREAD_COUNT", "50")
		case Reactive:
			w.Logger.Body("Reactive web application detected")
			layer.LaunchEnvironment.Default("BPL_JVM_THREAD_COUNT", "50")
		case Servlet:
			w.Logger.Body("Servlet web application detected")
			layer.LaunchEnvironment.Default("BPL_JVM_THREAD_COUNT", "250")
		}

		return layer, nil
	})
}

func (WebApplicationType) Name() string {
	return "web-application-type"
}
