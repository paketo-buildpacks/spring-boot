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

package helper

import (
	"bytes"
	"fmt"
	"github.com/paketo-buildpacks/libpak/sherpa"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/paketo-buildpacks/libpak/bard"
)

type SpringClassDataSharing struct {
	Logger bard.Logger
}

func (s SpringClassDataSharing) Execute() (map[string]string, error) {

	s.Logger.Info("Spring Class Data Sharing Enabled, contributing -Dspring.aot.enabled=true and -XX:SharedArchiveFile=application.jsa to JAVA_OPTS")
	s.Logger.Body("Those are the files we have in the workspace")

	StartOSCommand("", "ls", "-al", "./")

	var values []string
	if s, ok := os.LookupEnv("JAVA_TOOL_OPTIONS"); ok {
		values = append(values, s)
	}

	values = append(values, "-Dspring.aot.enabled=true")
	values = append(values, "-XX:SharedArchiveFile=application.jsa")

	return map[string]string{"JAVA_TOOL_OPTIONS": strings.Join(values, " ")}, nil
}

func StartOSCommand(envVariable string, command string, arguments ...string) {
	cmd := exec.Command(command, arguments...)
	cmd.Env = os.Environ()
	if envVariable != "" {
		cmd.Env = append(cmd.Env, envVariable)
	}
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
	}
	fmt.Println("Result: " + out.String())
}

func resetAllFilesMtimeAndATime(root string, date time.Time) ([]string, error) {
	println("Entering resetAllFIles")
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			println(path)
			file, err := os.Open(path)
			if err != nil {
				log.Printf("Could not open file: %s", path)
			}
			sherpa.CopyFile(file, fmt.Sprintf("%s.bak", path))

			if err := os.Chtimes(path, date, date); err != nil {
				log.Printf("Could not update atime and mtime for %s\n", fmt.Sprintf("%s.bak", path))
			}
			os.Remove(path)
			os.Rename(fmt.Sprintf("%s.bak", path), path)
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
