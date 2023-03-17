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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/spring-boot/v5/boot"
)

func testWebApplicationTypeResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path string
	)

	it.Before(func() {
		var err error

		path, err = ioutil.TempDir("", "native-image-application")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	context("classes", func() {

		var Touch = func(name string) {
			file := filepath.Join(path, fmt.Sprintf("%s.class", strings.ReplaceAll(name, ".", "/")))
			Expect(os.MkdirAll(filepath.Dir(file), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(file, []byte{}, 0644)).To(Succeed())
		}

		it("no indicators", func() {
			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.None))
		})

		it("WebFluxIndicator", func() {
			Touch(boot.WebFluxIndicatorClass)

			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.Reactive))
		})

		it("WebFluxIndicator, WebMVCIndicator", func() {
			Touch(boot.WebFluxIndicatorClass)
			Touch(boot.WebMVCIndicatorClass)

			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.None))
		})

		it("WebFluxIndicator, JerseyIndicator", func() {
			Touch(boot.WebFluxIndicatorClass)
			Touch(boot.JerseyIndicatorClass)

			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.None))
		})

		it("ServletIndicatorClasses Javax", func() {
			Touch(boot.JavaxServlet)
			Touch(boot.ConfigurableWebApplicationContextIndicatorClass)

			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.Servlet))
		})

		it("ServletIndicatorClasses Jakarta", func() {
			Touch(boot.JakartaServlet)
			Touch(boot.ConfigurableWebApplicationContextIndicatorClass)

			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.Servlet))
		})

		it("Servlets only", func() {
			Touch(boot.JakartaServlet)
			Touch(boot.JavaxServlet)

			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.None))
		})
	})

	context("lib", func() {

		var Copy = func(name string) {
			in, err := os.Open(filepath.Join("testdata", "web-application-type", name))
			Expect(err).NotTo(HaveOccurred())
			defer in.Close()

			out, err := os.OpenFile(filepath.Join(path, name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			Expect(err).NotTo(HaveOccurred())
			defer out.Close()

			_, err = io.Copy(out, in)
			Expect(err).NotTo(HaveOccurred())
		}

		it("no indicators", func() {
			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.None))
		})

		it("WebFluxIndicator", func() {
			Copy("webfluxindicator.jar")

			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.Reactive))
		})

		it("WebFluxIndicator, WebMVCIndicator", func() {
			Copy("webfluxindicatorwebmvcindicator.jar")

			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.None))
		})

		it("WebFluxIndicator, JerseyIndicator", func() {
			Copy("webfluxindicatorjerseyindicator.jar")

			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.None))
		})

		it("ServletIndicatorClasses", func() {
			Copy("servletindicators.jar")

			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.Servlet))
		})

		it("ServletIndicatorClasses Jakarta", func() {
			Copy("servletindicators-jakarta.jar")

			w, err := boot.NewWebApplicationResolver(path, path)
			Expect(err).NotTo(HaveOccurred())

			Expect(w.Resolve()).To(Equal(boot.Servlet))
		})
	})
}
