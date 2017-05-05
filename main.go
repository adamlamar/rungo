package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"

	log "github.com/Sirupsen/logrus"
)

const (
	DEFAULT_DOWNLOAD_URL = "https://storage.googleapis.com/golang/go%s.%s-%s.tar.gz"
	DEFAULT_GOOS         = runtime.GOOS
	DEFAULT_GOARCH       = runtime.GOARCH

	EXTRACTED_CANARY = "go-install"
)

var goosFlag = flag.String("goos", DEFAULT_GOOS, "Go OS")
var goarchFlag = flag.String("goarch", DEFAULT_GOARCH, "Go Architecture")
var verbose = flag.Bool("verbose", false, "Verbose output")

func main() {
	flag.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// Flags
	goos := *goosFlag
	goarch := *goarchFlag

	version := ""
	if len(flag.Args()) < 1 {
		log.Fatal("Must provide go version: e.g., go-install 1.2.3")
	} else {
		version = flag.Args()[0]
	}

	// baseDir of all file operations for this go version
	baseDir := filepath.Join("go", version)

	// Prepend provided path prefix to baseDir
	if len(flag.Args()) == 2 {
		baseDir = filepath.Join(flag.Args()[1], baseDir)
	}

	// URL to download golangArchive
	fileUrl := fmt.Sprintf(DEFAULT_DOWNLOAD_URL, version, goos, goarch)

	// Location on the filesystem to store the golang archive
	golangArchive := filepath.Join(baseDir, path.Base(fileUrl))

	err := downloadFile(fileUrl, golangArchive)
	if err != nil {
		log.Fatalf("Failed to download: %v", err)
	}

	canaryFile := filepath.Join(baseDir, EXTRACTED_CANARY) // File that signals extraction has already occurred
	if fileExists(canaryFile) {
		log.Debugf("Skipping extraction due to presence of canary at %q", canaryFile)
	} else {
		err = extractFile(golangArchive, baseDir)
		if err != nil {
			log.Fatalf("Failed to extract: %v", err)
		}
		ioutil.WriteFile(canaryFile, []byte(""), 0755)
		log.Infof("Successfully extracted %q", golangArchive)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func downloadFile(url, fileToSave string) error {
	err := os.MkdirAll(filepath.Dir(fileToSave), os.ModeDir|0700)
	if err != nil {
		log.Debugf("Failed to MkdirAll: %v", err)
		return err
	}

	file, err := os.OpenFile(fileToSave, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0755)
	if os.IsExist(err) {
		log.Debugf("File %q already exists, skipping download", fileToSave)
		return nil
	} else if err != nil {
		return err
	}
	defer file.Close()

	// Download file
	log.Infof("Downloading file %s", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Debugf("Failed to download golang archive: %v", err)
		return err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		log.Debugf("Failed to download golang archive: non-2XX response: %q", resp.Status)
		return fmt.Errorf("Received %q http status", resp.Status)
	}
	defer resp.Body.Close()

	// Write file to disk
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Debugf("Failed copy golang archive to disk: %v", err)
		return err
	}

	return nil
}

func extractFile(golangArchive, baseDir string) error {
	// Remove extracted canary, if exists
	_ = os.Remove(filepath.Join(baseDir, EXTRACTED_CANARY))

	log.Debugf("Extracting %q", golangArchive)
	err := os.MkdirAll(baseDir, os.ModeDir|0700)
	if err != nil {
		log.Debugf("Failed to MkdirAll: %v", err)
		return err
	}

	file, err := os.Open(golangArchive)
	if err != nil {
		log.Debugf("Failed to Open: %v", err)
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		log.Debugf("Failed to gzip.NewReader: %v", err)
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	// Extract all files, based off http://blog.ralch.com/tutorial/golang-working-with-tar-and-gzip/
	fileCount := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Debugf("Failed to tarReader.Next(): %v", err)
			return err
		}

		path := filepath.Join(baseDir, header.Name)
		fileInfo := header.FileInfo()
		if fileInfo.IsDir() {
			err = os.MkdirAll(path, fileInfo.Mode())
			if err != nil {
				log.Debugf("Failed to MkdirAll for %q: %v", path, err)
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileInfo.Mode())
		if err != nil {
			log.Debugf("Failed to OpenFile for %q: %v", path, err)
			return err
		}

		_, err = io.Copy(file, tarReader)
		if err != nil {
			log.Debugf("Failed to io.Copy for %q: %v", path, err)
			file.Close()
			return err
		}
		file.Close()
		fileCount++
	}
	log.Debugf("Wrote %d files to %q", fileCount, baseDir)

	return nil
}
