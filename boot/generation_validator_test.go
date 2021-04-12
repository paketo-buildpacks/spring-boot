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
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/heroku/color"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/spring-boot/boot"
)

func testGenerationValidator(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		b  *bytes.Buffer
		gv boot.GenerationValidator
	)

	it.Before(func() {
		var err error

		b = bytes.NewBuffer(nil)

		gv, err = boot.NewGenerationValidator(filepath.Join("testdata", "test-spring-generations.toml"))
		Expect(err).NotTo(HaveOccurred())
		gv.Logger = bard.NewLogger(b)
	})

	it("ignores missing file", func() {
		_, err := boot.NewGenerationValidator(filepath.Join("testdata", "unknown-spring-generations.toml"))
		Expect(err).NotTo(HaveOccurred())
	})

	it("ignores unknown slug", func() {
		Expect(gv.Validate("unknown-slug", "test-version")).NotTo(HaveOccurred())
		Expect(b.Len()).To(BeZero())
	})

	it("ignores unknown version", func() {
		Expect(gv.Validate("spring-boot", "2.4.0.RELEASE")).NotTo(HaveOccurred())
		Expect(b.Len()).To(BeZero())
	})

	it("ignores invalid version", func() {
		Expect(gv.Validate("spring-boot", "unknown")).NotTo(HaveOccurred())
		Expect(b.Len()).To(BeZero())
	})

	it("does not log warning", func() {
		Expect(gv.Validate("spring-boot", "2.3.0.RELEASE")).NotTo(HaveOccurred())
		Expect(b.Len()).To(BeZero())
	})

	it("logs commercial warning", func() {
		Expect(gv.Validate("spring-boot", "2.0.0.RELEASE")).NotTo(HaveOccurred())
		Expect(b.String()).To(Equal(fmt.Sprintf("  %s\n", color.New(color.FgYellow, color.Bold, color.Faint).Sprint(
			"This application uses Spring Boot 2.0.0.RELEASE. Commercial updates for 2.0.x ended on 2020-04-01."))))
	})

	it("logs open source warning", func() {
		// implementation of Validate uses `time.Now()` and will eventually need to be updated
		// `boot/testdata/test-spring-generations.toml has the defined generations for this test
		Expect(gv.Validate("spring-boot", "2.2.0.RELEASE")).NotTo(HaveOccurred())
		Expect(b.String()).To(Equal(fmt.Sprintf("  %s\n", color.New(color.FgYellow, color.Bold, color.Faint).Sprint(
			"This application uses Spring Boot 2.2.0.RELEASE. Open Source updates for 2.2.x ended on 2020-10-01."))))
	})

}
