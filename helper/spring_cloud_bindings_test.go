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

package helper_test

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/spring-boot/v5/helper"
)

func testSpringCloudBindings(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		s = helper.SpringCloudBindings{}
	)

	context("$BPL_SPRING_CLOUD_BINDINGS_ENABLED", func() {
		it.Before(func() {
			Expect(os.Setenv("BPL_SPRING_CLOUD_BINDINGS_ENABLED", "false")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BPL_SPRING_CLOUD_BINDINGS_ENABLED")).To(Succeed())
		})

		it("returns if $BPL_SPRING_CLOUD_BINDINGS_ENABLED is set to false", func() {
			Expect(s.Execute()).To(Equal(map[string]string{
				"JAVA_TOOL_OPTIONS": "-Dorg.springframework.cloud.bindings.boot.enable=false",
			}))
		})
	})

	context("$BPL_SPRING_CLOUD_BINDINGS_DISABLED", func() {
		it.Before(func() {
			Expect(os.Setenv("BPL_SPRING_CLOUD_BINDINGS_DISABLED", "true")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BPL_SPRING_CLOUD_BINDINGS_DISABLED")).To(Succeed())
		})

		it("returns if $BPL_SPRING_CLOUD_BINDINGS_DISABLED is set to true", func() {
			Expect(s.Execute()).To(Equal(map[string]string{
				"JAVA_TOOL_OPTIONS": "-Dorg.springframework.cloud.bindings.boot.enable=false",
			}))
		})
	})

	it("contributes configuration", func() {
		Expect(s.Execute()).To(Equal(map[string]string{
			"JAVA_TOOL_OPTIONS": "-Dorg.springframework.cloud.bindings.boot.enable=true",
		}))
	})

	context("$JAVA_TOOL_OPTIONS", func() {
		it.Before(func() {
			Expect(os.Setenv("JAVA_TOOL_OPTIONS", "test-java-tool-options")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("JAVA_TOOL_OPTIONS")).To(Succeed())
		})

		it("contributes configuration appended to existing $JAVA_TOOL_OPTIONS", func() {
			Expect(s.Execute()).To(Equal(map[string]string{
				"JAVA_TOOL_OPTIONS": "test-java-tool-options -Dorg.springframework.cloud.bindings.boot.enable=true",
			}))
		})
	})

}
