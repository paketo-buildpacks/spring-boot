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
	"os"

	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/sherpa"
)

type SpringPerformance struct {
	Logger bard.Logger
}

func (s SpringPerformance) Execute() (map[string]string, error) {
	var values []string
	aot := sherpa.ResolveBool("BPL_SPRING_AOT_ENABLED")
	aotCache := sherpa.ResolveBool("BPL_JVM_CDS_ENABLED") || sherpa.ResolveBool("BPL_JVM_AOTCACHE_ENABLED")
	if !aot && !aotCache {
		return nil, nil
	}

	if aot {
		s.Logger.Info("Spring AOT Enabled, contributing -Dspring.aot.enabled=true to JAVA_TOOL_OPTIONS")
		values = append(values, "-Dspring.aot.enabled=true")
	}

	if aotCache {

		applicationJsa := "application.jsa"
		applicationAot := "application.aot"

		if _, errJsa := os.Stat(applicationJsa); errJsa == nil {
			s.Logger.Info("Spring CDS Enabled, contributing -XX:SharedArchiveFile=application.jsa to JAVA_TOOL_OPTIONS")
			values = append(values, "-XX:SharedArchiveFile=application.jsa")
		} else {
			if _, errAot := os.Stat(applicationAot); errAot == nil {
				s.Logger.Info("Spring AOT Cache Enabled, contributing -XX:AOTCache=application.aot to JAVA_TOOL_OPTIONS")
				values = append(values, "-XX:AOTCache=application.aot")
			} else {
				s.Logger.Info("Something went wrong, neither application.jsa nor application.aot found, CDS/AOT Cache optimization disabled")
				s.Logger.Infof("Errors looking for  application.jsa and application.aot: %v - %v", errJsa, errAot)
			}
		}
	}
	opts := sherpa.AppendToEnvVar("JAVA_TOOL_OPTIONS", " ", values...)
	return map[string]string{"JAVA_TOOL_OPTIONS": opts}, nil
}
