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
	"strings"

	"github.com/paketo-buildpacks/libpak/bard"
)

type SpringCloudBindings struct {
	Logger bard.Logger
}

func (s SpringCloudBindings) Execute() (map[string]string, error) {
	var err error

	enabled := true
	if s, ok := os.LookupEnv("BPL_SPRING_CLOUD_BINDINGS_ENABLED"); ok {
		enabled, err = strconv.ParseBool(s)
		if err != nil {
			return nil, fmt.Errorf("unable to parse $BPL_SPRING_CLOUD_BINDINGS_ENABLED\n%w", err)
		}
	}

	if !enabled {
		return nil, nil
	}

	s.Logger.Info("Spring Cloud Bindings Enabled")

	var values []string
	if s, ok := os.LookupEnv("JAVA_TOOL_OPTIONS"); ok {
		values = append(values, s)
	}

	values = append(values, "-Dorg.springframework.cloud.bindings.boot.enable=true")

	return map[string]string{"JAVA_TOOL_OPTIONS": strings.Join(values, " ")}, nil
}
