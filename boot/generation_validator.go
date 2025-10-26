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
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/heroku/color"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/pelletier/go-toml"
)

const DatePattern = "2006-01-02"

var (
	MaxDate           = time.Unix(1<<63-62135596801, 999999999)
	Warningf          = color.New(color.FgYellow, color.Bold, color.Faint).SprintfFunc()
	NormalizedVersion = regexp.MustCompile(`[\d]+(?:\.[\d]+(?:\.[\d]+)?)?`)
)

type Generation struct {
	Name *semver.Constraints `toml:"Name"`
	OSS  time.Time           `toml:"OSS"`
}

func (g *Generation) UnmarshalTOML(data interface{}) error {
	var err error

	d, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unable to cast data")
	}

	// it's possible if Name is formatted like Finchley.x, such as spring cloud, an error is returned
	// not really a problem; it just means this version won't be considered, fine for spring cloud
	g.Name, _ = semver.NewConstraint(d["Name"].(string))

	oss := d["OSS"].(string)
	if len(oss) > 0 {
		if g.OSS, err = time.Parse(DatePattern, oss); err != nil {
			return fmt.Errorf("unable to parse %s to date\n%w", oss, err)
		}
	} else {
		g.OSS = MaxDate
	}

	return nil
}

type Project struct {
	Name        string       `toml:"Name"`
	Slug        string       `toml:"Slug"`
	Status      string       `toml:"Status"`
	Generations []Generation `toml:"Generations"`
}

type Projects struct {
	Projects []Project `toml:"Projects"`
}

type GenerationValidator struct {
	Logger   bard.Logger
	Projects []Project
}

func NewGenerationValidator(path string) (GenerationValidator, error) {
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return GenerationValidator{}, fmt.Errorf("unable to read %s\n%w", path, err)
	}

	var p Projects
	if err := toml.Unmarshal(b, &p); err != nil {
		return GenerationValidator{}, fmt.Errorf("unable to decode %s\n%w", path, err)
	}

	return GenerationValidator{Projects: p.Projects}, nil
}

func (v GenerationValidator) Validate(slug string, version string) error {
	nv := NormalizedVersion.FindString(version)
	if nv == "" {
		return nil
	}

	for _, p := range v.Projects {
		if slug == p.Slug {
			ver, err := semver.NewVersion(nv)
			if err != nil {
				return fmt.Errorf("unable to parse %s to version\n%w", version, err)
			}

			for _, g := range p.Generations {
				if g.Name.Check(ver) {
					t := time.Now()
					if t.After(g.OSS) {
						v.Logger.Header(Warningf("This application uses %s %s. Open Source updates for %s ended on %s.",
							p.Name, version, g.Name, g.OSS.Format(DatePattern)))
					}

					return nil
				}
			}

			return nil
		}
	}

	return nil
}
