package main

import (
	"archive/zip"
	"fmt"
	"github.com/magiconair/properties"
	"github.com/paketo-buildpacks/libjvm"
	"github.com/paketo-buildpacks/libpak/sherpa"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"
)

const createdBy = "17.9.9 (Spring Boot Paketo Buildpack)"

func TestHelloName(t *testing.T) {

	const originalJarBasename = "demo-0.0.1-SNAPSHOT.jar"
	const originalJarFullPath = "/Users/anthonyd2/workspaces/paketo-buildpacks/samples/java/gradle/build/libs/" + originalJarBasename
	const targetUnpackedDirectory = "/Users/anthonyd2/workspaces/paketo-buildpacks/spring-boot/unpacked"
	originalJarExplodedDirectory, _ := os.CreateTemp("", "unpack")

	os.RemoveAll(originalJarExplodedDirectory.Name() + "/")
	Unzip(originalJarFullPath, originalJarExplodedDirectory.Name())
	os.MkdirAll(targetUnpackedDirectory+"/application", 0755)
	os.MkdirAll(targetUnpackedDirectory+"/dependencies", 0755)
	Zip(targetUnpackedDirectory+"/application/"+originalJarBasename, originalJarExplodedDirectory.Name()+"/BOOT-INF/classes/", true)

	tempDirectory := fmt.Sprint(time.Now().UnixMilli()) + "/"
	os.MkdirAll(os.TempDir()+tempDirectory+"/META-INF/", 0755)
	runAppJarManifest, _ := os.Create(os.TempDir() + tempDirectory + "/META-INF/MANIFEST.MF")
	writeRunAppJarManifest(originalJarExplodedDirectory.Name(), runAppJarManifest, "application/"+originalJarBasename)
	Zip(targetUnpackedDirectory+"/run-app.jar", os.TempDir()+tempDirectory, false)
	sherpa.CopyDir(originalJarExplodedDirectory.Name()+"/BOOT-INF/lib/", targetUnpackedDirectory+"/dependencies/")

}

func writeRunAppJarManifest(originalJarExplodedDirectory string, runAppJarManifest *os.File, relocatedOriginalJar string) {
	originalManifest, _ := libjvm.NewManifest(originalJarExplodedDirectory)
	startClassValue, _ := retrieveStartClassValue(originalManifest)
	classPathValue, _ := retrieveClasspathFromIdx(originalManifest, originalJarExplodedDirectory, "dependencies/", relocatedOriginalJar)

	type Manifest struct {
		MainClass string
		ClassPath string
		CreatedBy string
	}

	manifestValues := Manifest{startClassValue, rewriteWithMaxLineLength("Class-Path: "+classPathValue, 72), createdBy}
	tmpl, err := template.New("manifest").Parse("Manifest-Version: 1.0\n" +
		"Main-Class: {{.MainClass}}\n" +
		"{{.ClassPath}}\n" +
		"Created-By: {{.CreatedBy}}\n" +
		" ")
	if err != nil {
		panic(err)
	}
	//buf := &bytes.Buffer{}
	err = tmpl.Execute(runAppJarManifest, manifestValues)
	if err != nil {
		panic(err)
	}

	//reformattedClassPath :=
	//runAppJarManifest.Write([]byte(reformattedClassPath))
}

func rewriteWithMaxLineLength(s string, length int) string {

	//a := []rune(s)
	result := ""
	currentLine := ""
	indent := 0
	remainder := ""

	for i, r := range s {
		currentLine = currentLine + string(r)
		remainder = remainder + string(r)
		//fmt.Printf("i%d r %c\n", i, r)
		j := i + 1
		if indent > 0 {
			j = i + 1 + indent
		}
		if i > 0 && j%length == 0 {
			//fmt.Printf("%v\n", currentLine)
			result = result + currentLine + "\n"
			currentLine = " "
			indent = indent + 1
			remainder = " "
		}
	}
	result = result + remainder
	//fmt.Printf("%v\n", remainder)
	return result
}
func retrieveClasspathFromIdx(manifest *properties.Properties, dir string, relocatedDir string, relocatedOriginalJar string) (string, error) {
	classpathIdx, ok := manifest.Get("Spring-Boot-Classpath-Index")
	if !ok {
		return "", fmt.Errorf("manifest does not contain Spring-Boot-Classpath-Index")
	}

	file := filepath.Join(dir, classpathIdx)
	in, err := os.Open(filepath.Join(dir, classpathIdx))
	if err != nil {
		return "", fmt.Errorf("unable to open %s\n%w", file, err)
	}
	defer in.Close()

	var libs []string
	if err := yaml.NewDecoder(in).Decode(&libs); err != nil {
		return "", fmt.Errorf("unable to decode %s\n%w", file, err)
	}

	var relocatedLibs []string
	relocatedLibs = append(relocatedLibs, relocatedOriginalJar)
	for _, lib := range libs {
		relocatedLibs = append(relocatedLibs, strings.ReplaceAll(lib, "BOOT-INF/lib/", relocatedDir))
	}

	return strings.Join(relocatedLibs, " "), nil
}

func retrieveStartClassValue(manifest *properties.Properties) (string, error) {
	startClass, ok := manifest.Get("Start-Class")
	if !ok {
		return "", fmt.Errorf("no Start-Class foudn int he manifest, are you sure the jar it was built with Spring Boot")
	} else {
		return startClass, nil
	}

}

func Zip(archivePath string, folderPath string, create bool) {
	os.Chdir(folderPath)
	if create {
		CreateEmptyManifest()
	}

	file, err := os.Create(archivePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		path = strings.TrimPrefix(path, folderPath)

		fmt.Printf("Crawling: %#v\n", path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			var err error
			if path != "" {
				_, err = w.Create(path)
			}
			if err != nil {
				return err
			}
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		f, err := w.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}
	err = filepath.Walk(folderPath, walker)

	//f, err := w.Create("META-INF/MANIFEST.MF")
	//_, err = io.Copy(f, file)

	if err != nil {
		panic(err)
	}

}

func CreateEmptyManifest() (*os.File, error) {
	// Create a temporary file
	err := os.Mkdir("META-INF", 0755)
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.Create("META-INF/MANIFEST.MF")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	fmt.Println(file.Name())

	// Write some text to the file
	manifestContent :=
		`Manifest-Version: 1.0
Created-By: 17.0.8 (Spring Boot Paketo Buildpack)
`
	_, err = file.WriteString(manifestContent)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Close the file
	err = file.Close()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println("The temporary file is created:", file.Name())
	return file, err
}

// heavily inspired by https://stackoverflow.com/a/58192644/24069
func Unzip(src, dest string) error {
	dest = filepath.Clean(dest) + "/"

	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer CloseOrPanic(r)()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		path := filepath.Join(dest, f.Name)
		// Check for ZipSlip: https://snyk.io/research/zip-slip-vulnerability
		if !strings.HasPrefix(path, dest) {
			return fmt.Errorf("%s: illegal file path", path)
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer CloseOrPanic(rc)()

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer CloseOrPanic(f)()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func CloseOrPanic(f io.Closer) func() {
	return func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}
}
