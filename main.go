package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"

	log "github.com/Sirupsen/logrus"
	"github.com/mitchellh/go-homedir"
)

const (
	DEFAULT_GOLANG        = "1.10.3"
	DEFAULT_GOOS          = runtime.GOOS
	DEFAULT_GOARCH        = runtime.GOARCH
	DEFAULT_DOWNLOAD_BASE = "https://storage.googleapis.com/golang/"

	EXTRACTED_CANARY = "go-extracted"
	SHA_EXTENSION    = ".sha256"
	RUNGO_VERSION    = "0.0.7"
)

func main() {
	verbose := os.Getenv("RUNGO_VERBOSE")
	if verbose != "" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.SetFormatter(&log.TextFormatter{DisableColors: true})
	log.Debugf("Starting rungo version %s", RUNGO_VERSION)

	// Find the version requested
	version := findVersion()

	// Find the user's home directory
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Fatalf("Failed to determine home directory: %v", err)
	}

	// baseDir of all file operations for this go version
	baseDir := filepath.Join(homeDir, DEFAULT_HOME_INSTALL_LOCATION, version)

	// Form URL to download golangArchive
	downloadBase := os.Getenv("RUNGO_DOWNLOAD_BASE")
	if downloadBase == "" {
		downloadBase = DEFAULT_DOWNLOAD_BASE
	}
	fileUrl := downloadBase + fmt.Sprintf(DEFAULT_ARCHIVE_NAME, version, DEFAULT_GOOS, DEFAULT_GOARCH)

	// Location on the filesystem to store the golang archive
	golangArchive := filepath.Join(baseDir, path.Base(fileUrl))

	sha256sum, err := fetchSha256(fileUrl+SHA_EXTENSION, golangArchive+SHA_EXTENSION)
	if err != nil {
		log.Fatalf("Failed to fetch sha256: %v", err)
	}

	err = downloadFile(fileUrl, sha256sum, golangArchive)
	if err != nil {
		log.Fatalf("Failed to download: %v", err)
	}

	// Extract golang archive
	canaryFile := filepath.Join(baseDir, EXTRACTED_CANARY) // File that signals extraction has already occurred
	if fileExists(canaryFile) {
		log.Debugf("Skipping extraction due to presence of canary at %q", canaryFile)
	} else {
		// Remove extracted canary, if exists
		_ = os.Remove(filepath.Join(baseDir, EXTRACTED_CANARY))

		err = extractFile(golangArchive, baseDir)
		if err != nil {
			log.Fatalf("Failed to extract: %v", err)
		}
		ioutil.WriteFile(canaryFile, []byte(""), 0755)
		log.Debugf("Successfully extracted %q", golangArchive)
	}

	// Run go command
	setGoRoot(baseDir)
	binary := filepath.Base(os.Args[0])
	if binary == "rungo" {
		binary = "go"
	} else if binary == "rungo.exe" {
		binary = "go.exe"
	}

	err = runGo(binary, baseDir, os.Args[1:])
	if err != nil {
		log.Fatalf("command failed: %v", err)
	}
}
