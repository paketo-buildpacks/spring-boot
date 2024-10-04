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

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/spring-boot/v5/boot"
)

func testConfigurationMetadata(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path string
	)

	it.Before(func() {
		var err error

		path, err = ioutil.TempDir("", "configuration-metadata")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	context("from path", func() {
		// ... (existing test cases)

		it("returns dataflow decoded contents with names", func() {
			Expect(os.MkdirAll(filepath.Join(path, "META-INF"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "spring-configuration-metadata.json"),
				[]byte(`{ "properties": [ { "name": "alpha", "sourceType": "alpha" } ] }`), 0644)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "dataflow-configuration-metadata.properties"),
				[]byte("configuration-properties.names=alpha"), 0644)).To(Succeed())

			cm, err := boot.NewConfigurationMetadataFromPath(path)
			Expect(err).NotTo(HaveOccurred())

			Expect(boot.NewDataFlowConfigurationMetadata(path, cm)).To(Equal(boot.ConfigurationMetadata{
				Properties: []boot.Property{{Name: "alpha", SourceType: "alpha"}},
			}))
		})

		it("returns combined dataflow decoded contents with classes and names", func() {
			Expect(os.MkdirAll(filepath.Join(path, "META-INF"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "spring-configuration-metadata.json"),
				[]byte(`{ "groups": [ { "name": "alpha", "sourceType": "alpha" }, { "name": "beta", "sourceType": "beta" } ] }`), 0644)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "dataflow-configuration-metadata.properties"),
				[]byte("configuration-properties.classes=alpha\nconfiguration-properties.names=beta"), 0644)).To(Succeed())

			cm, err := boot.NewConfigurationMetadataFromPath(path)
			Expect(err).NotTo(HaveOccurred())

			Expect(boot.NewDataFlowConfigurationMetadata(path, cm)).To(Equal(boot.ConfigurationMetadata{
				Groups: []boot.Group{
					{Name: "alpha", SourceType: "alpha"},
					{Name: "beta", SourceType: "beta"},
				},
			}))
		})

		it("handles empty names gracefully", func() {
			Expect(os.MkdirAll(filepath.Join(path, "META-INF"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "spring-configuration-metadata.json"),
				[]byte(`{ "properties": [ { "name": "alpha", "sourceType": "alpha" } ] }`), 0644)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "dataflow-configuration-metadata.properties"),
				[]byte("configuration-properties.names="), 0644)).To(Succeed())

			cm, err := boot.NewConfigurationMetadataFromPath(path)
			Expect(err).NotTo(HaveOccurred())

			Expect(boot.NewDataFlowConfigurationMetadata(path, cm)).To(Equal(boot.ConfigurationMetadata{
				Properties: []boot.Property{{Name: "alpha", SourceType: "alpha"}},
			}))
		})
	})


func testConfigurationMetadata(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path string
	)

	it.Before(func() {
		var err error

		path, err = ioutil.TempDir("", "configuration-metadata")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	context("from path", func() {

		it("returns empty if file does not exist", func() {
			Expect(boot.NewConfigurationMetadataFromPath(path)).To(BeZero())
		})

		it("returns decoded contents", func() {
			Expect(os.MkdirAll(filepath.Join(path, "META-INF"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "spring-configuration-metadata.json"),
				[]byte(`{ "groups": [ { "name": "alpha" } ] }`), 0644)).To(Succeed())

			Expect(boot.NewConfigurationMetadataFromPath(path)).To(Equal(boot.ConfigurationMetadata{
				Groups: []boot.Group{{Name: "alpha"}},
			}))
		})

		it("returns dataflow decoded contents", func() {
			Expect(os.MkdirAll(filepath.Join(path, "META-INF"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "spring-configuration-metadata.json"),
				[]byte(`{ "groups": [ { "name": "alpha", "sourceType": "alpha" } ] }`), 0644))
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "dataflow-configuration-metadata.properties"),
				[]byte("configuration-properties.classes=alpha"), 0644))

			cm, err := boot.NewConfigurationMetadataFromPath(path)
			Expect(err).NotTo(HaveOccurred())

			Expect(boot.NewDataFlowConfigurationMetadata(path, cm)).To(Equal(boot.ConfigurationMetadata{
				Groups: []boot.Group{{Name: "alpha", SourceType: "alpha"}},
			}))
		})

		it("returns dataflow decoded contents handling trailing comma correctly", func() {
			Expect(os.MkdirAll(filepath.Join(path, "META-INF"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "spring-configuration-metadata.json"),
				[]byte(`{ "properties": [ { "name": "alpha", "sourceType": "alpha" }, { "name": "beta" } ] }`), 0644))
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "dataflow-configuration-metadata.properties"),
				[]byte("configuration-properties.classes=alpha,"), 0644))

			cm, err := boot.NewConfigurationMetadataFromPath(path)
			Expect(err).NotTo(HaveOccurred())

			Expect(boot.NewDataFlowConfigurationMetadata(path, cm)).To(Equal(boot.ConfigurationMetadata{
				Properties: []boot.Property{{Name: "alpha", SourceType: "alpha"}},
			}))
		})

		it("returns dataflow decoded contents", func() {
			Expect(os.MkdirAll(filepath.Join(path, "META-INF"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "spring-configuration-metadata.json"),
				[]byte(`{ "groups": [ { "name": "alpha", "sourceType": "alpha" } ] }`), 0644)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "dataflow-configuration-metadata-whitelist.properties"),
				[]byte("configuration-properties.classes=alpha"), 0644)).To(Succeed())

			cm, err := boot.NewConfigurationMetadataFromPath(path)
			Expect(err).NotTo(HaveOccurred())

			Expect(boot.NewDataFlowConfigurationMetadata(path, cm)).To(Equal(boot.ConfigurationMetadata{
				Groups: []boot.Group{{Name: "alpha", SourceType: "alpha"}},
			}))
		})
	})

	context("from JAR", func() {

		it("returns empty if file does not exist", func() {
			file := filepath.Join("testdata", "stub-empty.jar")

			Expect(boot.NewConfigurationMetadataFromJAR(file)).To(BeZero())
		})

		it("returns decoded contents", func() {
			file := filepath.Join("testdata", "stub-spring-configuration-metadata.jar")

			Expect(boot.NewConfigurationMetadataFromJAR(file)).To(Equal(boot.ConfigurationMetadata{
				Groups: []boot.Group{{Name: "alpha"}},
			}))
		})
	})

	context("detects Dataflow", func() {
		it("returns false if file does not exist", func() {
			Expect(boot.DataFlowConfigurationExists(path)).To(BeFalse())
		})

		it("returns true if the file does exist", func() {
			Expect(os.MkdirAll(filepath.Join(path, "META-INF"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "dataflow-configuration-metadata-whitelist.properties"),
				[]byte("configuration-properties.classes=alpha"), 0644)).To(Succeed())
			Expect(boot.DataFlowConfigurationExists(path)).To(BeFalse())
		})

		it("return false and the error if the file cannot be read", func() {
			Expect(os.MkdirAll(filepath.Join(path, "META-INF"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "META-INF", "dataflow-configuration-metadata-whitelist.properties"),
				[]byte("configuration-properties.classes=alpha"), 0644)).To(Succeed())

			Expect(os.Chmod(filepath.Join(path, "META-INF"), 0000)).To(Succeed())
			ok, err := boot.DataFlowConfigurationExists(path)
			Expect(os.Chmod(filepath.Join(path, "META-INF"), 0755)).To(Succeed())

			Expect(ok).To(BeFalse())
			Expect(err).To(MatchError(HaveSuffix("permission denied")))
		})
	})

}
