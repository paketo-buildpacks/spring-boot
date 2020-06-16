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
)

type Detect struct{}

func (Detect) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {
	cr, err := libpak.NewConfigurationResolver(context.Buildpack, nil)
	if err != nil {
		return libcnb.DetectResult{}, fmt.Errorf("unable to create configuration resolver\n%w", err)
	}

	var requires []libcnb.BuildPlanRequire

	if _, ok := cr.Resolve("BP_BOOT_NATIVE_IMAGE"); ok {
		requires = append(requires,
			libcnb.BuildPlanRequire{
				Name:     "jdk",
				Metadata: map[string]interface{}{"native-image": true},
			},
			libcnb.BuildPlanRequire{
				Name:     "jvm-application",
				Metadata: map[string]interface{}{"native-image": true},
			},
		)
	} else {
		requires = append(requires, libcnb.BuildPlanRequire{Name: "jvm-application"})
	}

	return libcnb.DetectResult{
		Pass: true,
		Plans: []libcnb.BuildPlan{
			{Requires: requires},
		},
	}, nil
}
