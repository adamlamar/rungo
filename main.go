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
	"os/exec"
	"path"
	"path/filepath"
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

const (
	DEFAULT_DOWNLOAD_URL          = "https://storage.googleapis.com/golang/go%s.%s-%s.tar.gz"
	DEFAULT_HOME_INSTALL_LOCATION = ".go"
	DEFAULT_GOOS                  = runtime.GOOS
	DEFAULT_GOARCH                = runtime.GOARCH

	EXTRACTED_CANARY = "go-extracted"
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
		log.Fatal("Must provide go version: e.g., run-go 1.8.1")
	} else {
		version = flag.Args()[0]
	}

	// Find the user's home directory
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Fatal("Failed to determine home directory: %v", err)
	}

	// baseDir of all file operations for this go version
	baseDir := filepath.Join(homeDir, DEFAULT_HOME_INSTALL_LOCATION, version)

	// URL to download golangArchive
	fileUrl := fmt.Sprintf(DEFAULT_DOWNLOAD_URL, version, goos, goarch)

	// Location on the filesystem to store the golang archive
	golangArchive := filepath.Join(baseDir, path.Base(fileUrl))

	err = downloadFile(fileUrl, golangArchive)
	if err != nil {
		log.Fatalf("Failed to download: %v", err)
	}

	// Extract golang archive
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

	// Run go command
	setGoRoot(baseDir)
	if len(flag.Args()) > 1 {
		err = runGo(baseDir, flag.Args()[1:])
	} else {
		err = runGo(baseDir, nil)
	}
	if err != nil {
		log.Fatalf("go command failed: %v", err)
	}
}

func runGo(baseDir string, args []string) error {
	goBinary := filepath.Join(baseDir, "go", "bin", "go")
	cmd := exec.Command(goBinary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Debugf("Executing %q with arguments %v", goBinary, args)
	return cmd.Run()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func setGoRoot(baseDir string) {
	os.Setenv("GOROOT", baseDir)
}

func downloadFile(url, fileToSave string) error {
	dir := filepath.Dir(fileToSave)
	err := os.MkdirAll(dir, os.ModeDir|0700)
	if err != nil {
		return errors.Wrapf(err, "mkdir %q failed", dir)
	}

	file, err := os.OpenFile(fileToSave, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0755)
	if os.IsExist(err) {
		log.Debugf("File %q already exists, skipping download", fileToSave)
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "open %q failed", fileToSave)
	}
	defer file.Close()

	// Download file
	log.Infof("Downloading file %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "download failed")
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return fmt.Errorf("failed due to non-2XX response: %q", resp.Status)
	}
	defer resp.Body.Close()

	// Write file to disk
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return errors.Wrap(err, "copy to disk failed")
	}

	return nil
}

func extractFile(golangArchive, baseDir string) error {
	// Remove extracted canary, if exists
	_ = os.Remove(filepath.Join(baseDir, EXTRACTED_CANARY))

	log.Debugf("Extracting %q", golangArchive)
	err := os.MkdirAll(baseDir, os.ModeDir|0700)
	if err != nil {
		return errors.Wrapf(err, "mkdir %q failed", baseDir)
	}

	file, err := os.Open(golangArchive)
	if err != nil {
		return errors.Wrapf(err, "file open %q failed", golangArchive)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return errors.Wrap(err, "gzip reader open failed")
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
			return errors.Wrap(err, "tar reader next failed")
		}

		path := filepath.Join(baseDir, header.Name)
		fileInfo := header.FileInfo()
		if fileInfo.IsDir() {
			err = os.MkdirAll(path, fileInfo.Mode())
			if err != nil {
				return errors.Wrapf(err, "mkdir %q failed", path)
			}
			continue
		} else {
			// Make directory containing the current file, if needed. Some tarballs don't include the top-level directory entry
			err = os.MkdirAll(filepath.Dir(path), 0755|os.ModeDir)
			if err != nil {
				return errors.Wrapf(err, "mkdir %q failed", path)
			}
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileInfo.Mode())
		if err != nil {
			return errors.Wrapf(err, "open file %q failed", path)
		}

		_, err = io.Copy(file, tarReader)
		if err != nil {
			file.Close()
			return errors.Wrapf(err, "copy for %q failed", path)
		}
		file.Close()
		fileCount++
	}
	log.Debugf("Wrote %d files to %q", fileCount, baseDir)

	return nil
}
