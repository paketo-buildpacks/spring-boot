package boot

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/sherpa"
)

var javaVersionPattern = regexp.MustCompile(`(?i)\bversion\s+"([^"]+)"`)

func JavaMajorVersionFromJRE(executor effect.Executor) (int, error) {
	javaCommand := JavaCommand()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := executor.Execute(effect.Execution{
		Command: javaCommand,
		Args:    []string{"-version"},
		Stdout:  &stdout,
		Stderr:  &stderr,
	}); err != nil {
		return -1, fmt.Errorf("unable to execute %s -version: %w", javaCommand, err)
	}

	outputParts := []string{}
	if s := strings.TrimSpace(stdout.String() + stderr.String()); s != "" {
		outputParts = append(outputParts, s)
	}

	output := strings.TrimSpace(strings.Join(outputParts, "\n"))
	if output == "" {
		return -1, fmt.Errorf("java -version produced no output")
	}

	major, err := JavaMajorVersion(output)
	if err != nil {
		return -1, fmt.Errorf("unable to parse JVM major version: %w", err)
	}

	return major, nil
}

func JavaCommand() string {
	javaCommand := "java"
	if jreHome := sherpa.GetEnvWithDefault("JRE_HOME", sherpa.GetEnvWithDefault("JAVA_HOME", "")); jreHome != "" {
		javaCommand = filepath.Join(jreHome, "bin", "java")
	}
	return javaCommand
}

func JavaMajorVersion(output string) (int, error) {
	matches := javaVersionPattern.FindStringSubmatch(output)
	if len(matches) < 2 {
		return 0, fmt.Errorf("java version string not found in output")
	}

	version := strings.TrimSpace(matches[1])
	if strings.HasPrefix(version, "1.") && len(version) > 2 {
		version = version[2:]
	}

	segments := strings.FieldsFunc(version, func(r rune) bool {
		return r == '.' || r == '-' || r == '_' || r == '+'
	})
	if len(segments) == 0 {
		return 0, fmt.Errorf("unable to split java version segments from %q", version)
	}

	major, err := strconv.Atoi(segments[0])
	if err != nil {
		return 0, fmt.Errorf("unable to parse java major version from %q: %w", segments[0], err)
	}

	return major, nil
}
