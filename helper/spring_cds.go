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
	"github.com/paketo-buildpacks/libpak/bard"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type SpringCds struct {
	Logger bard.Logger
}

func (s SpringCds) Execute() (map[string]string, error) {

	s.Logger.Info("Spring App CDS Enabled, contributing -XX:SharedArchiveFile=application.jsa to JAVA_OPTS")
	s.Logger.Body("Those are the files we have in the workspace")

	StartOSCommand("", "ls", "-al", "./")
	StartOSCommand("", "env")
	var values []string
	if s, ok := os.LookupEnv("JAVA_TOOL_OPTIONS"); ok {
		values = append(values, s)
	}
	if val, ok := os.LookupEnv("BPL_SPRING_AOT_ENABLED"); ok {
		enabled, err := strconv.ParseBool(val)
		if enabled && err == nil {
			s.Logger.Info("Spring AOT Enabled, contributing -Dspring.aot.enabled=true to JAVA_OPTS")
			values = append(values, "-Dspring.aot.enabled=true")
		}
	}
	values = append(values, "-XX:SharedArchiveFile=application.jsa")

	return map[string]string{"JAVA_TOOL_OPTIONS": strings.Join(values, " ")}, nil
}

func StartOSCommand(envVariable string, command string, arguments ...string) {
	fmt.Println("StartOSCommand")
	fmt.Println(command, arguments)
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
