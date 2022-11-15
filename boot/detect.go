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
	"github.com/paketo-buildpacks/libpak/sherpa"
	"path/filepath"
)

const (
	PlanEntrySpringBoot     = "spring-boot"
	PlanEntryNativeArgFile  = "native-image-argfile"
	PlanEntryJVMApplication = "jvm-application"
)

type Detect struct{}

func (Detect) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {
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
	nativeImageArgFile := filepath.Join(context.Application.Path, "META-INF", "native-image", "argfile")
	if exists, err := sherpa.Exists(nativeImageArgFile); err != nil{
		return libcnb.DetectResult{}, fmt.Errorf("unable to check for native-image arguments file at %s\n%w", nativeImageArgFile, err)
	} else if exists{
		result = libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: PlanEntrySpringBoot},
						{Name: PlanEntryNativeArgFile},
					},
					Requires: []libcnb.BuildPlanRequire{
						{Name: PlanEntrySpringBoot},
						{Name: PlanEntryJVMApplication},
					},
				},
			},
		}
	}
	return result, nil
}
