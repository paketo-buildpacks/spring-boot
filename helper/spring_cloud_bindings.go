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

package helper

import (
	"fmt"
	"os"
	"strconv"

	"github.com/paketo-buildpacks/libpak/sherpa"

	"github.com/heroku/color"

	"github.com/paketo-buildpacks/libpak/bard"
)

type SpringCloudBindings struct {
	Logger bard.Logger
}

func (s SpringCloudBindings) Execute() (map[string]string, error) {
	
	var err error
	enabled := true
	var opt string

	if val, ok := os.LookupEnv("BPL_SPRING_CLOUD_BINDINGS_ENABLED"); ok {
		s.Logger.Infof(color.YellowString("WARNING: BPL_SPRING_CLOUD_BINDINGS_ENABLED is deprecated, support will be removed in a coming release. Use BPL_SPRING_CLOUD_BINDINGS_DISABLED instead"))
		enabled, err = strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("unable to parse $BPL_SPRING_CLOUD_BINDINGS_ENABLED\n%w", err)
		}
	}
	// Switching from "BPL_SPRING_CLOUD_BINDINGS_ENABLED" to "BPL_SPRING_CLOUD_BINDINGS_DISABLED" which defaults to 'false' to follow convention
	if sherpa.ResolveBool("BPL_SPRING_CLOUD_BINDINGS_DISABLED") || !enabled {
		opt = "-Dorg.springframework.cloud.bindings.boot.enable=false"
	} else {
		opt = "-Dorg.springframework.cloud.bindings.boot.enable=true"
		s.Logger.Info("Spring Cloud Bindings Enabled")
	}

	values := sherpa.AppendToEnvVar("JAVA_TOOL_OPTIONS", " ", opt)

	return map[string]string{"JAVA_TOOL_OPTIONS": values}, nil
}
