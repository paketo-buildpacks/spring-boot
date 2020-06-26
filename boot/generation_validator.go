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

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver/v3"
	"github.com/heroku/color"
	"github.com/paketo-buildpacks/libpak/bard"
)

const DatePattern = "2006-01-02"

var (
	MaxDate           = time.Unix(1<<63-62135596801, 999999999)
	Warningf          = color.New(color.FgYellow, color.Bold, color.Faint).SprintfFunc()
	NormalizedVersion = regexp.MustCompile(`[\d]+(?:\.[\d]+(?:\.[\d]+)?)?`)
)

type Generation struct {
	Name       *semver.Constraints `toml:"name"`
	OSS        time.Time           `toml:"oss"`
	Commercial time.Time           `toml:"commercial"`
}

func (g *Generation) UnmarshalTOML(data interface{}) error {
	var err error

	d, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unable to cast data")
	}

	if g.Name, err = semver.NewConstraint(d["name"].(string)); err != nil {
		return fmt.Errorf("unable to parse %s to constraint\n%w", d["name"], err)
	}

	oss := d["oss"].(string)
	if len(oss) > 0 {
		if g.OSS, err = time.Parse(DatePattern, oss); err != nil {
			return fmt.Errorf("unable to parse %s to date\n%w", oss, err)
		}
	} else {
		g.OSS = MaxDate
	}

	commercial := d["commercial"].(string)
	if len(commercial) > 0 {
		if g.Commercial, err = time.Parse(DatePattern, commercial); err != nil {
			return fmt.Errorf("unable to parse %s to date\n%w", commercial, err)
		}
	} else {
		g.Commercial = MaxDate
	}

	return nil
}

type Project struct {
	Name        string       `toml:"name"`
	Slug        string       `toml:"slug"`
	Status      string       `toml:"status"`
	Generations []Generation `toml:"generations"`
}

type Projects struct {
	Projects []Project `toml:"projects"`
}

type GenerationValidator struct {
	Logger   bard.Logger
	Projects []Project
}

func NewGenerationValidator(path string) (GenerationValidator, error) {
	var p Projects

	if _, err := toml.DecodeFile(path, &p); err != nil && !os.IsNotExist(err) {
		return GenerationValidator{}, fmt.Errorf("unable to decode %s\n%w", path, err)
	}

	return GenerationValidator{Projects: p.Projects}, nil
}

func (v GenerationValidator) Validate(slug string, version string) error {
	for _, p := range v.Projects {
		if slug == p.Slug {
			ver, err := semver.NewVersion(NormalizedVersion.FindString(version))
			if err != nil {
				return fmt.Errorf("unable to parse %s to version\n%w", version, err)
			}

			for _, g := range p.Generations {
				if g.Name.Check(ver) {
					t := time.Now()

					if t.After(g.Commercial) {
						v.Logger.Header(Warningf("This application uses %s %s. Commercial updates for %s ended on %s.",
							p.Name, version, g.Name, g.Commercial.Format(DatePattern)))
					} else if t.After(g.OSS) {
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
