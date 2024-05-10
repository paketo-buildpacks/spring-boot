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

func testSpringPerformance(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		s = helper.SpringPerformance{}
	)

	context("$BPL_SPRING_AOT_ENABLED and BPL_JVM_CDS_ENABLED set to false", func() {
		it.Before(func() {
			Expect(os.Setenv("BPL_SPRING_AOT_ENABLED", "false")).To(Succeed())
			Expect(os.Setenv("BPL_JVM_CDS_ENABLED", "false")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BPL_SPRING_AOT_ENABLED")).To(Succeed())
			Expect(os.Unsetenv("BPL_JVM_CDS_ENABLED")).To(Succeed())
		})

		it("returns if $BPL_SPRING_AOT_ENABLED and $BPL_JVM_CDS_ENABLED are set to false", func() {
			Expect(s.Execute()).To(BeNil())
		})
	})

	context("$BPL_JVM_CDS_ENABLED set to false", func() {
		it.Before(func() {
			Expect(os.Setenv("BPL_JVM_CDS_ENABLED", "false")).To(Succeed())
			Expect(os.Setenv("BPL_SPRING_AOT_ENABLED", "true")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BPL_JVM_CDS_ENABLED")).To(Succeed())
		})

		it("only configures AOT if $BPL_JVM_CDS_ENABLED is set to false", func() {
			Expect(s.Execute()).To(Equal(map[string]string{
				"JAVA_TOOL_OPTIONS": "-Dspring.aot.enabled=true",
			}))
		})
	})

	context("$BPL_SPRING_AOT_ENABLED set to false", func() {
		it.Before(func() {
			Expect(os.Setenv("BPL_SPRING_AOT_ENABLED", "false")).To(Succeed())
			Expect(os.Setenv("BPL_JVM_CDS_ENABLED", "true")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BPL_SPRING_AOT_ENABLED")).To(Succeed())
			Expect(os.Unsetenv("BPL_JVM_CDS_ENABLED")).To(Succeed())
		})

		it("only configures CDS if $BPL_SPRING_AOT_ENABLED is set to false", func() {
			Expect(s.Execute()).To(Equal(map[string]string{
				"JAVA_TOOL_OPTIONS": "-XX:SharedArchiveFile=application.jsa",
			}))
		})
	})

	context("$BPL_SPRING_AOT_ENABLED and $BPL_JVM_CDS_ENABLED both set to true", func() {

		it.Before(func() {
			Expect(os.Setenv("BPL_SPRING_AOT_ENABLED", "true")).To(Succeed())
			Expect(os.Setenv("BPL_JVM_CDS_ENABLED", "true")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BPL_SPRING_AOT_ENABLED")).To(Succeed())
			Expect(os.Unsetenv("BPL_JVM_CDS_ENABLED")).To(Succeed())
		})

		it("contributes configuration", func() {
			Expect(s.Execute()).To(Equal(map[string]string{
				"JAVA_TOOL_OPTIONS": "-Dspring.aot.enabled=true -XX:SharedArchiveFile=application.jsa",
			}))
		})
	})

	context("$JAVA_TOOL_OPTIONS", func() {
		it.Before(func() {
			Expect(os.Setenv("JAVA_TOOL_OPTIONS", "test-java-tool-options")).To(Succeed())
			Expect(os.Setenv("BPL_SPRING_AOT_ENABLED", "true")).To(Succeed())
			Expect(os.Setenv("BPL_JVM_CDS_ENABLED", "true")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("JAVA_TOOL_OPTIONS")).To(Succeed())
			Expect(os.Unsetenv("BPL_SPRING_AOT_ENABLED")).To(Succeed())
			Expect(os.Unsetenv("BPL_JVM_CDS_ENABLED")).To(Succeed())
		})

		it("contributes configuration appended to existing $JAVA_TOOL_OPTIONS", func() {
			Expect(s.Execute()).To(Equal(map[string]string{
				"JAVA_TOOL_OPTIONS": "test-java-tool-options -Dspring.aot.enabled=true -XX:SharedArchiveFile=application.jsa",
			}))
		})
	})
	
}
