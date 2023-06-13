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
	"github.com/magiconair/properties"
	"github.com/paketo-buildpacks/libjvm"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"regexp"
	"strconv"
	"strings"
)

const (
	PlanEntrySpringBoot       = "spring-boot"
	PlanEntryJVMApplication   = "jvm-application"
	PlanEntryNativeProcessed  = "native-processed"
	MavenConfigActiveProfiles = "BP_MAVEN_ACTIVE_PROFILES"
)

type Detect struct {
	Logger bard.Logger
}

func (d Detect) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {
	result := libcnb.DetectResult{
		Pass: true,
		Plans: []libcnb.BuildPlan{
			{
				Provides: []libcnb.BuildPlanProvide{
					{Name: PlanEntrySpringBoot},
				},
				Requires: []libcnb.BuildPlanRequire{
					{Name: PlanEntryJVMApplication},
					{Name: PlanEntrySpringBoot},
				},
			},
		},
	}
	manifest, err := libjvm.NewManifest(context.Application.Path)
	if err != nil {
		return libcnb.DetectResult{}, fmt.Errorf("unable to read manifest in %s\n%w", context.Application.Path, err)
	}

	cr, err := libpak.NewConfigurationResolver(context.Buildpack, nil)
	if err != nil {
		return libcnb.DetectResult{}, fmt.Errorf("unable to create configuration resolver\n%w", err)
	}

	mavenNativeProfileDetected := isMavenNativeProfileDetected(&cr, &d.Logger)
	springBootNativeProcessedDetected := isSpringBootNativeProcessedDetected(manifest, &d.Logger)

	if springBootNativeProcessedDetected || mavenNativeProfileDetected {
		result = libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: PlanEntrySpringBoot},
						{Name: PlanEntryNativeProcessed},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: PlanEntryJVMApplication},
						{Name: PlanEntrySpringBoot},
					},
				},
			},
		}
	}
	return result, nil
}

func isSpringBootNativeProcessedDetected(manifest *properties.Properties, logger *bard.Logger) bool {
	springBootNativeProcessedString, found := manifest.Get("Spring-Boot-Native-Processed")
	springBootNativeProcessed, _ := strconv.ParseBool(springBootNativeProcessedString)
	detected := found && springBootNativeProcessed
	if detected {
		logger.Bodyf("Spring-Boot-Native-Processed MANIFEST entry was detected, activating native image")
	}
	return detected
}

func isMavenNativeProfileDetected(cr *libpak.ConfigurationResolver, logger *bard.Logger) bool {
	mavenActiveProfiles, _ := cr.Resolve(MavenConfigActiveProfiles)
	mavenActiveProfilesAsSlice := strings.Split(mavenActiveProfiles, ",")
	r, _ := regexp.Compile("^native$|^\\?native$")

	for _, profile := range mavenActiveProfilesAsSlice {
		if r.MatchString(profile) {
			logger.Bodyf("Maven native profile was detected in %s, activating native image", MavenConfigActiveProfiles)
			return true
		}
	}
	return false
}
