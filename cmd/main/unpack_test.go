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
	"text/template"
	"time"
)

// This file was a place to experiment with zipping / unzipping in a proper Jar fashion
// Not an actual test file

const createdBy = "17.9.9 (Spring Boot Paketo Buildpack)"

func HelloName() {

	const originalJarBasename = "demo-0.0.1-SNAPSHOT.jar"
	const originalJarFullPath = "~/workspaces/paketo-buildpacks/samples/java/gradle/build/libs/" + originalJarBasename
	const targetUnpackedDirectory = "~/workspaces/paketo-buildpacks/spring-boot/unpacked"
	originalJarExplodedDirectory, _ := os.CreateTemp("", "unpack")

	os.RemoveAll(originalJarExplodedDirectory.Name() + "/")
	//Unzip(originalJarFullPath, originalJarExplodedDirectory.Name())
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

//func archiveWithFastZip(source, target string) {
//	// Create archiveWithFastZip file
//	w, err := os.Create(target)
//	if err != nil {
//		panic(err)
//	}
//	defer w.Close()
//
//	// Create new Archiver
//	//var options ArchiverOption = nil
//	a, err := fastzip.NewArchiver(w, source, fastzip.WithArchiverMethod(zip.Deflate))
//	if err != nil {
//		panic(err)
//	}
//	defer a.Close()
//
//	// Register a non-default level compressor if required
//	//a.RegisterCompressor(zip.Deflate, fastzip.FlateCompressor(1))
//
//	// Walk directory, adding the files we want to add
//	files := make(map[string]os.FileInfo)
//	err = filepath.Walk(source, func(pathname string, info os.FileInfo, err error) error {
//		if info.IsDir() {
//			return nil
//		}
//		files[pathname] = info
//		return nil
//	})
//
//	// Archive
//	if err = a.Archive(context.Background(), files); err != nil {
//		panic(err)
//	}
//}
//
//func archiveWithArchiver(source, target string) {
//
//	walkedFiles := make(map[string]string)
//	_ = filepath.Walk(source, func(pathname string, info os.FileInfo, err error) error {
//		if info.IsDir() {
//			return nil
//		}
//		walkedFiles[pathname], _ = filepath.Rel(filepath.Dir(source), pathname)
//		return nil
//	})
//
//	files, _ := archiver.FilesFromDisk(nil, walkedFiles)
//	out, _ := os.Create(target)
//	defer out.Close()
//
//	format := archiver.Zip{
//		SelectiveCompression: false,
//		Compression:          zip.Store,
//		ContinueOnError:      false,
//		TextEncoding:         "UTF8",
//	}
//	format.Archive(context.Background(), out, files)
//
//}

//func TestZip(t *testing.T) {
//	boot.CreateJar("~/workspaces/spring-cds-demo/build/libs/unzip/", "~/workspaces/spring-cds-demo/build/libs/spring-cds-demo-1.0.0-SNAPSHOT-rezip.jar")
//}

func zipSource2(source, target string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		if strings.HasSuffix(header.Name, ".jar") {
			header.Method = zip.Store
		} else {
			header.Method = zip.Deflate
		}

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
}

func zipSource(source, target string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Store

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
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
