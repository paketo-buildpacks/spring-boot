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
	"os"
	"os/exec"

	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/sherpa"
)

type SpringCds struct {
	Logger bard.Logger
}

func (s SpringCds) Execute() (map[string]string, error) {

	s.Logger.Body("Those are the files we have in the workspace")

	StartOSCommand("", "ls", "-al", "./")
	StartOSCommand("", "env")

	opts := sherpa.GetEnvWithDefault("JAVA_TOOL_OPTIONS", "")

	if sherpa.ResolveBool("BPL_SPRING_AOT_ENABLED") {
		s.Logger.Info("Spring AOT Enabled, contributing -Dspring.aot.enabled=true to JAVA_OPTS")
		opts = sherpa.AppendToEnvVar("JAVA_TOOL_OPTIONS", " ",  "-Dspring.aot.enabled=true")
	}

	if sherpa.ResolveBool("BPL_JVM_CDS_ENABLED") {
		s.Logger.Info("Spring CDS Enabled, contributing -XX:SharedArchiveFile=application.jsa to JAVA_OPTS")
		opts = sherpa.AppendToEnvVar("JAVA_TOOL_OPTIONS", " ",  "-XX:SharedArchiveFile=application.jsa")
	}

	return map[string]string{"JAVA_TOOL_OPTIONS": opts}, nil
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
