package main

import (
	"flag"
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
		// Remove extracted canary, if exists
		_ = os.Remove(filepath.Join(baseDir, EXTRACTED_CANARY))

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
