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

package boot

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/magiconair/properties"
)

type Group struct {
	Name         string `json:"name"`
	Type         string `json:"type,omitempty"`
	Description  string `json:"description,omitempty"`
	SourceType   string `json:"sourceType,omitempty"`
	SourceMethod string `json:"sourceMethod,omitempty"`
}

type Deprecation struct {
	Level       string `json:"level,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

type Property struct {
	Name         string       `json:"name"`
	Type         string       `json:"type,omitempty"`
	Description  string       `json:"description,omitempty"`
	SourceType   string       `json:"sourceType,omitempty"`
	DefaultValue interface{}  `json:"defaultValue,omitempty"`
	Deprecation  *Deprecation `json:"deprecation,omitempty"`
}

type ValueHint struct {
	Value       interface{} `json:"value"`
	Description string      `json:"description,omitempty"`
}

type ValueProvider struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type Hint struct {
	Name      string          `json:"name"`
	Values    []ValueHint     `json:"values,omitempty"`
	Providers []ValueProvider `json:"providers,omitempty"`
}

type ConfigurationMetadata struct {
	Groups     []Group    `json:"groups,omitempty"`
	Properties []Property `json:"properties,omitempty"`
	Hints      []Hint     `json:"hints,omitempty"`
}

func NewConfigurationMetadataFromPath(path string) (ConfigurationMetadata, error) {
	file := filepath.Join(path, "META-INF", "spring-configuration-metadata.json")
	in, err := os.Open(file)
	if os.IsNotExist(err) {
		return ConfigurationMetadata{}, nil
	} else if err != nil {
		return ConfigurationMetadata{}, fmt.Errorf("unable to open %s\n%w", file, err)
	}
	defer in.Close()

	var c ConfigurationMetadata
	if err := json.NewDecoder(in).Decode(&c); err != nil {
		return ConfigurationMetadata{}, fmt.Errorf("unable to decode %s\n%w", file, err)
	}

	return c, nil
}

func NewConfigurationMetadataFromJAR(jar string) (ConfigurationMetadata, error) {
	zIn, err := zip.OpenReader(jar)
	if os.IsExist(err) {
		return ConfigurationMetadata{}, nil
	} else if err != nil {
		return ConfigurationMetadata{}, fmt.Errorf("unable to open %s\n%w", jar, err)
	}
	defer zIn.Close()

	var c ConfigurationMetadata
	for _, f := range zIn.File {
		if f.Name != filepath.Join("META-INF", "spring-configuration-metadata.json") {
			continue
		}

		in, err := f.Open()
		if err != nil {
			return ConfigurationMetadata{}, fmt.Errorf("unable to open %s\n%w", f.Name, err)
		}
		defer in.Close()

		if err := json.NewDecoder(in).Decode(&c); err != nil {
			return ConfigurationMetadata{}, fmt.Errorf("unable to decode %s\n%w", f.Name, err)
		}

		break
	}

	return c, nil
}

func DataFlowConfigurationExists(path string) (bool, error) {
	_, err := os.Stat(filepath.Join(path, "META-INF", "dataflow-configuration-metadata.properties"))
	if err != nil && os.IsNotExist(err) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return true, nil
}

func NewDataFlowConfigurationMetadata(path string, metadata ConfigurationMetadata) (ConfigurationMetadata, error) {
	file := filepath.Join(path, "META-INF", "dataflow-configuration-metadata.properties")
	b, err := ioutil.ReadFile(file)
	if os.IsNotExist(err) {
		file := filepath.Join(path, "META-INF", "dataflow-configuration-metadata-whitelist.properties")
		b, err = ioutil.ReadFile(file)
		if os.IsNotExist(err) {
			return ConfigurationMetadata{}, nil
		} else if err != nil {
			return ConfigurationMetadata{}, fmt.Errorf("unable to open %s\n%w", file, err)
		}
	} else if err != nil {
		return ConfigurationMetadata{}, fmt.Errorf("unable to open %s\n%w", file, err)
	}

	p, err := properties.Load(b, properties.UTF8)
	if err != nil {
		return ConfigurationMetadata{}, fmt.Errorf("unable to load properties from %s\n%w", file, err)
	}

	// Load classes
	s, ok := p.Get("configuration-properties.classes")
	var classes []string
	if ok {
		for _, s := range strings.Split(s, ",") {
			class := strings.TrimSpace(s)
			if class != "" {
				classes = append(classes, class)
			}
		}
	}

	// Load names
	s, ok = p.Get("configuration-properties.names")
	var names []string
	if ok {
		for _, s := range strings.Split(s, ",") {
			name := strings.TrimSpace(s)
			if name != "" {
				names = append(names, name)
			}
		}
	}

	// Merge classes and names
	var combined []string
	combined = append(combined, classes...)
	combined = append(combined, names...)

	m := ConfigurationMetadata{}

	for _, g := range metadata.Groups {
		for _, c := range combined {
			if c == g.SourceType || c == g.Name {
				m.Groups = append(m.Groups, g)
				break
			}
		}
	}

	for _, p := range metadata.Properties {
		for _, c := range combined {
			if c == p.SourceType || c == p.Name {
				m.Properties = append(m.Properties, p)
				break
			}
		}
	}

	for _, h := range metadata.Hints {
		for _, p := range m.Properties {
			if p.Name == h.Name {
				m.Hints = append(m.Hints, h)
				break
			}
		}
	}

	return m, nil
}
