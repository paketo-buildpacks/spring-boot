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
	"github.com/paketo-buildpacks/libjvm"
	"strconv"
)

const (
	PlanEntrySpringBoot      = "spring-boot"
	PlanEntryJVMApplication  = "jvm-application"
	PlanEntryNativeProcessed = "native-processed"
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
	manifest, err := libjvm.NewManifest(context.Application.Path)
	if err != nil {
		return libcnb.DetectResult{}, fmt.Errorf("unable to read manifest in %s\n%w", context.Application.Path, err)
	}

	springBootNativeProcessedString, ok := manifest.Get("Spring-Boot-Native-Processed")
	springBootNativeProcessed, err := strconv.ParseBool(springBootNativeProcessedString)
	if ok && springBootNativeProcessed {
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
