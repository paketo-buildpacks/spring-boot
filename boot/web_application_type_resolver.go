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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ApplicationType uint8

const (
	None ApplicationType = iota
	Reactive
	Servlet
)

const (
	WebMVCIndicatorClass  = "org.springframework.web.servlet.DispatcherServlet"
	WebFluxIndicatorClass = "org.springframework.web.reactive.DispatcherHandler"
	JerseyIndicatorClass  = "org.glassfish.jersey.servlet.ServletContainer"
)

var ServletIndicatorClasses = []string{
	"javax.servlet.Servlet",
	"org.springframework.web.context.ConfigurableWebApplicationContext",
}

type WebApplicationTypeResolver struct {
	Classes map[string]interface{}
}

type result struct {
	err   error
	value []string
}

func NewWebApplicationResolver(classes string, lib string) (WebApplicationTypeResolver, error) {
	w := WebApplicationTypeResolver{
		Classes: make(map[string]interface{}),
	}

	if err := filepath.Walk(classes, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || filepath.Ext(path) != ".class" {
			return nil
		}

		class, err := filepath.Rel(classes, path)
		if err != nil {
			return fmt.Errorf("unable to determine relative path from %s to %s\n%w", path, classes, err)
		}

		class = strings.TrimSuffix(class, ".class")
		class = strings.ReplaceAll(class, "/", ".")
		w.Classes[class] = nil

		return nil
	}); err != nil && !os.IsNotExist(err) {
		return WebApplicationTypeResolver{}, fmt.Errorf("unable to find class names in %s\n%w", classes, err)
	}

	jars, err := filepath.Glob(filepath.Join(lib, "*.jar"))
	if err != nil {
		return WebApplicationTypeResolver{}, fmt.Errorf("unable to glob %s/*.jar\n%w", lib, err)
	}

	results := make(chan result)
	var wg sync.WaitGroup
	for _, jar := range jars {

		wg.Add(1)
		go func(jar string) {
			defer wg.Done()

			in, err := zip.OpenReader(jar)
			if err != nil {
				results <- result{err: fmt.Errorf("unable to open %s\n%w", jar, err)}
				return
			}
			defer in.Close()

			var classes []string
			for _, f := range in.File {
				if f.FileInfo().IsDir() || filepath.Ext(f.Name) != ".class" {
					continue
				}

				class := strings.TrimSuffix(f.Name, ".class")
				class = strings.ReplaceAll(class, "/", ".")
				classes = append(classes, class)
			}

			results <- result{value: classes}
		}(jar)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		if r.err != nil {
			return WebApplicationTypeResolver{}, fmt.Errorf("unable to list classes\n%w", err)
		}
		for _, class := range r.value {
			w.Classes[class] = nil
		}
	}

	return w, nil
}

func (w WebApplicationTypeResolver) Resolve() ApplicationType {
	if w.isPresent(WebFluxIndicatorClass) && !w.isPresent(WebMVCIndicatorClass) && !w.isPresent(JerseyIndicatorClass) {
		return Reactive
	}

	for _, class := range ServletIndicatorClasses {
		if !w.isPresent(class) {
			return None
		}
	}

	return Servlet
}

func (w WebApplicationTypeResolver) isPresent(class string) bool {
	_, ok := w.Classes[class]
	return ok
}
