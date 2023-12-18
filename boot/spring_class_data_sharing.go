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
	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/spring-boot/v5/helper"
	"log"
	"os"
	"path/filepath"
	"time"
)

type SpringClassDataSharing struct {
	Dependency       libpak.BuildpackDependency
	LayerContributor libpak.LayerContributor
	Logger           bard.Logger
}

func NewSpringClassDataSharing(cache libpak.DependencyCache) SpringClassDataSharing {
	contributor := libpak.NewLayerContributor("spring-class-data-sharing", cache, libcnb.LayerTypes{
		Launch: true,
	})
	return SpringClassDataSharing{
		LayerContributor: contributor,
	}
}

func (s SpringClassDataSharing) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	s.LayerContributor.Logger = s.Logger
	layer, err := s.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {
		s.Logger.Body("Those are the files we have in the workspace BEFORE the training run", layer.Path)
		helper.StartOSCommand("", "ls", "-al", "./")
		helper.StartOSCommand("TZ=UTC", "find", "./", "-exec", "touch", "-t", "198001010000.01", "{}", ";")
		s.Logger.Body("Launching training run for CDS app", layer.Path)
		helper.StartOSCommand("", "/layers/paketo-buildpacks_oracle/jdk/bin/java",
			"-Dspring.aot.enabled=true",
			"-Dspring.context.exit=onRefresh",
			"-XX:ArchiveClassesAtExit=application.jsa",
			"-jar", "run-app.jar")

		s.Logger.Body("Those are the files we have in the workspace AFTER the training run", layer.Path)
		helper.StartOSCommand("", "ls", "-al", "./")
		helper.StartOSCommand("", "ls", "-al", "./application")
		helper.StartOSCommand("", "ls", "-al", "./dependencies")
		return layer, nil
	})
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to contribute spring-class-data-sharing layer\n%w", err)
	}
	return layer, nil
}

func (s SpringClassDataSharing) Name() string {
	return s.LayerContributor.Name
}

func resetAllFilesMtimeAndATime(root string, date time.Time) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			println(path)
			mtime := date
			atime := date
			if err := os.Chtimes(path, atime, mtime); err != nil {
				log.Printf("Could not update atime and mtime for %s\n", path)
			}
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
