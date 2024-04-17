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

	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/sherpa"
)

type SpringPerformance struct {
	Logger bard.Logger
}

func (s SpringPerformance) Execute() (map[string]string, error) {
	var values []string
	aot := sherpa.ResolveBool("BPL_SPRING_AOT_ENABLED")
	cds := sherpa.ResolveBool("BPL_JVM_CDS_ENABLED") 
	if !aot && !cds{
		return nil, nil
	}
	
	if aot {
		s.Logger.Info("Spring AOT Enabled, contributing -Dspring.aot.enabled=true to JAVA_TOOL_OPTIONS")
		values = append(values, "-Dspring.aot.enabled=true")
	}

	if cds {
		s.Logger.Info("Spring CDS Enabled, contributing -XX:SharedArchiveFile=application.jsa to JAVA_TOOL_OPTIONS")
		values = append(values, "-XX:SharedArchiveFile=application.jsa")
	}
	opts := sherpa.AppendToEnvVar("JAVA_TOOL_OPTIONS", " ", values...)
	return map[string]string{"JAVA_TOOL_OPTIONS": opts}, nil
}