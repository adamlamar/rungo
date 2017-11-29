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
	DEFAULT_GOOS   = runtime.GOOS
	DEFAULT_GOARCH = runtime.GOARCH

	EXTRACTED_CANARY = "go-extracted"
)

func main() {
	verbose := os.Getenv("RUNGO_VERBOSE")
	if verbose != "" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.SetFormatter(&log.TextFormatter{DisableColors: true})

	// Find the version requested
	version := findVersion()

	// Find the user's home directory
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Fatalf("Failed to determine home directory: %v", err)
	}

	// baseDir of all file operations for this go version
	baseDir := filepath.Join(homeDir, DEFAULT_HOME_INSTALL_LOCATION, version)

	// URL to download golangArchive
	fileUrl := fmt.Sprintf(DEFAULT_DOWNLOAD_URL, version, DEFAULT_GOOS, DEFAULT_GOARCH)

	// Location on the filesystem to store the golang archive
	golangArchive := filepath.Join(baseDir, path.Base(fileUrl))

	if version != "system" {
		err = downloadFile(fileUrl, golangArchive)
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
			log.Infof("Successfully extracted %q", golangArchive)
		}
	} else {
		baseDir = "/usr/local"
	}

	// Run go command
	setGoRoot(baseDir)
	binary := filepath.Base(os.Args[0])
	if binary == "rungo" {
		binary = "go"
	}

	err = runGo(binary, baseDir, os.Args[1:])
	if err != nil {
		if version == "system" {
			log.Fatalf("Failed to run system go: %v", err)
		} else {
			log.Fatalf("command failed: %v", err)
		}
	}
}
