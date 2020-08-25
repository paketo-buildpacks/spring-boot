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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/spring-boot/boot"
)

func testSpringCloudBindings(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx libcnb.BuildContext
	)

	it.Before(func() {
		var err error

		ctx.Layers.Path, err = ioutil.TempDir("", "spring-cloud-bindings-layers")
		Expect(err).NotTo(HaveOccurred())

		ctx.Application.Path, err = ioutil.TempDir("", "spring-cloud-bindings-app")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
	})

	it("contributes Spring Cloud Bindings", func() {
		dep := libpak.BuildpackDependency{
			URI:    "https://localhost/stub-spring-cloud-bindings.jar",
			SHA256: "723126712c0b22a7fe409664adf1fbb78cf3040e313a82c06696f5058e190534",
		}
		dc := libpak.DependencyCache{CachePath: "testdata"}

		s := boot.NewSpringCloudBindings(filepath.Join(ctx.Application.Path, "test-lib"), dep, dc, &libcnb.BuildpackPlan{})
		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = s.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.Launch).To(BeTrue())
		Expect(filepath.Join(layer.Path, "stub-spring-cloud-bindings.jar")).To(BeARegularFile())
		Expect(os.Readlink(filepath.Join(ctx.Application.Path, "test-lib", "stub-spring-cloud-bindings.jar"))).
			To(Equal(filepath.Join(layer.Path, "stub-spring-cloud-bindings.jar")))
	})
}
