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
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/spring-boot/v5/boot"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx    libcnb.DetectContext
		detect boot.Detect
	)

	nativeResult := libcnb.DetectResult{
		Pass: true,
		Plans: []libcnb.BuildPlan{
			{
				Provides: []libcnb.BuildPlanProvide{
					{Name: "spring-boot"},
					{Name: "native-processed"},
				},
				Requires: []libcnb.BuildPlanRequire{
					{Name: "jvm-application"},
					{Name: "spring-boot"},
				},
			},
		},
	}

	normalResult := libcnb.DetectResult{
		Pass: true,
		Plans: []libcnb.BuildPlan{
			{
				Provides: []libcnb.BuildPlanProvide{
					{Name: "spring-boot"},
				},
				Requires: []libcnb.BuildPlanRequire{
					{Name: "jvm-application"},
					{Name: "spring-boot"},
				},
			},
		},
	}

	it("always passes for standard build", func() {
		Expect(os.Unsetenv("BP_MAVEN_ACTIVE_PROFILES")).To(Succeed())
		Expect(os.RemoveAll(filepath.Join(ctx.Application.Path, "META-INF"))).To(Succeed())
		Expect(detect.Detect(ctx)).To(Equal(normalResult))
	})

	it("always passes for native build", func() {
		Expect(os.Unsetenv("BP_MAVEN_ACTIVE_PROFILES")).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(ctx.Application.Path, "META-INF"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(ctx.Application.Path, "META-INF", "MANIFEST.MF"), []byte(`
Spring-Boot-Version: 1.1.1
Spring-Boot-Classes: BOOT-INF/classes
Spring-Boot-Lib: BOOT-INF/lib
Spring-Boot-Native-Processed: true
`), 0644)).To(Succeed())
		Expect(detect.Detect(ctx)).To(Equal(nativeResult))
	})

	it("using BP_MAVEN_ACTIVE_PROFILES", func() {

		Expect(os.RemoveAll(filepath.Join(ctx.Application.Path, "META-INF"))).To(Succeed())

		Expect(os.Setenv("BP_MAVEN_ACTIVE_PROFILES", "native")).To(Succeed())
		Expect(detect.Detect(ctx)).To(Equal(nativeResult))

		Expect(os.Setenv("BP_MAVEN_ACTIVE_PROFILES", "p1,native")).To(Succeed())
		Expect(detect.Detect(ctx)).To(Equal(nativeResult))

		Expect(os.Setenv("BP_MAVEN_ACTIVE_PROFILES", "p1,?native")).To(Succeed())
		Expect(detect.Detect(ctx)).To(Equal(nativeResult))

		Expect(os.Setenv("BP_MAVEN_ACTIVE_PROFILES", "native,p1")).To(Succeed())
		Expect(detect.Detect(ctx)).To(Equal(nativeResult))

		Expect(os.Setenv("BP_MAVEN_ACTIVE_PROFILES", "?native")).To(Succeed())
		Expect(detect.Detect(ctx)).To(Equal(nativeResult))

		Expect(os.Setenv("BP_MAVEN_ACTIVE_PROFILES", "mynative,native")).To(Succeed())
		Expect(detect.Detect(ctx)).To(Equal(nativeResult))

		Expect(os.Setenv("BP_MAVEN_ACTIVE_PROFILES", "mynative")).To(Succeed())
		Expect(detect.Detect(ctx)).To(Equal(normalResult))

		Expect(os.Setenv("BP_MAVEN_ACTIVE_PROFILES", "!native")).To(Succeed())
		Expect(detect.Detect(ctx)).To(Equal(normalResult))

	})

}
