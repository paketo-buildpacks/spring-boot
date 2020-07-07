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

		s   boot.SpringCloudBindings
		ctx libcnb.BuildContext
		springBootLib string
	)

	it.Before(func() {
		var err error

		ctx.Layers.Path, err = ioutil.TempDir("", "spring-cloud-bindings-layers")
		Expect(err).NotTo(HaveOccurred())

		springBootLib, err = ioutil.TempDir("", "spring-cloud-bindings-app")
		Expect(err).NotTo(HaveOccurred())

		dep := libpak.BuildpackDependency{
			URI:    "https://localhost/stub-spring-cloud-bindings.jar",
			SHA256: "723126712c0b22a7fe409664adf1fbb78cf3040e313a82c06696f5058e190534",
		}
		cache := libpak.DependencyCache{CachePath: "testdata"}
		s = boot.NewSpringCloudBindings(springBootLib, dep, cache, &libcnb.BuildpackPlan{})
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
		Expect(os.RemoveAll(springBootLib)).To(Succeed())
	})

	context("Layer", func() {
		var layer libcnb.Layer
		it.Before(func() {
			var err error
			layer, err = ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			layer, err = s.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())
			Expect(layer.Launch).To(BeTrue())
		})

		it("is a launch layer", func() {
			Expect(layer.Launch).To(BeTrue())
		})

		it("contributes bindings jar", func() {
			Expect(filepath.Join(layer.Path, "stub-spring-cloud-bindings.jar")).To(BeARegularFile())
		})

		it("symlinks bindings jar to BOOT-INF", func() {
			linkPath := filepath.Join(s.SpringBootLib, "stub-spring-cloud-bindings.jar")
			Expect(linkPath).To(BeAnExistingFile())
			target, err := os.Readlink(linkPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(target).To(Equal(filepath.Join(layer.Path, "stub-spring-cloud-bindings.jar")))
		})

		it("contributes a profile script", func() {
			Expect(layer.Profile["enable-bindings.sh"]).To(Equal(expectedProfile))
		})
	})
}

const expectedProfile = `if [[ "${BPL_SPRING_CLOUD_BINDINGS_ENABLED:=y}" == "y" ]]; then
    printf "Spring Cloud Bindings Boot Auto-Configuration Enabled\n"
    export JAVA_OPTS="${JAVA_OPTS} -Dorg.springframework.cloud.bindings.boot.enable=true"
fi`
