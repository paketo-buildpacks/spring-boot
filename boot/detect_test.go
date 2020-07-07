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

package boot_test

import (
	"os"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/spring-boot/boot"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx    libcnb.DetectContext
		detect boot.Detect
	)

	it("always passes", func() {
		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Requires: []libcnb.BuildPlanRequire{
						{Name: "jvm-application"},
					},
				},
			},
		}))
	})

	context("BP_BOOT_NATIVE_IMAGE", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_BOOT_NATIVE_IMAGE", ""))
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_BOOT_NATIVE_IMAGE"))
		})

		it("adds native image requirement", func() {
			Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
				Pass: true,
				Plans: []libcnb.BuildPlan{
					{
						Requires: []libcnb.BuildPlanRequire{
							{Name: "jvm-application"},
							{
								Name:     "jdk",
								Metadata: map[string]interface{}{"native-image": true},
							},
						},
					},
				},
			}))
		})
	})

}
