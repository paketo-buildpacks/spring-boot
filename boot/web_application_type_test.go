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
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/sherpa"
	"github.com/paketo-buildpacks/spring-boot/boot"
	"github.com/sclevine/spec"
)

func testWebApplicationType(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx libcnb.BuildContext
		w   boot.WebApplicationType
	)

	it.Before(func() {
		var err error

		ctx.Layers.Path, err = ioutil.TempDir("", "web-application-type")
		Expect(err).NotTo(HaveOccurred())

		wr := boot.WebApplicationTypeResolver{Classes: map[string]interface{}{}}

		w = boot.NewWebApplicationType(wr, []sherpa.FileEntry{})
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("contributes None application type configuration", func() {
		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = w.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.Launch).To(BeTrue())
		Expect(layer.LaunchEnvironment["BPL_JVM_THREAD_COUNT.default"]).To(Equal("50"))
	})

	it("contributes Reactive application type configuration", func() {
		w.Resolver.Classes[boot.WebFluxIndicatorClass] = nil

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = w.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.Launch).To(BeTrue())
		Expect(layer.LaunchEnvironment["BPL_JVM_THREAD_COUNT.default"]).To(Equal("50"))
	})

	it("contributes Servlet application type configuration", func() {
		for _, class := range boot.ServletIndicatorClasses {
			w.Resolver.Classes[class] = nil
		}

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = w.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.Launch).To(BeTrue())
		Expect(layer.LaunchEnvironment["BPL_JVM_THREAD_COUNT.default"]).To(Equal("250"))
	})

}
