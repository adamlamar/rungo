package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

func runGo(binary, baseDir string, args []string) error {
	goBinary := filepath.Join(baseDir, "go", "bin", binary)
	cmd := exec.Command(goBinary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Debugf("Executing %q with arguments %v", goBinary, args)
	return cmd.Run()
}

// Get the requested version, either from env variable or go-version file
func findVersion() string {
	envVersion := findEnvVersion()
	if envVersion != "" {
		return envVersion
	}

	fileVersion := findVersionFile()
	if fileVersion != "" {
		return fileVersion
	}

	log.Fatal("Failed to determine desired go version")
	return "system"
}

func findEnvVersion() string {
	return os.Getenv("GO_VERSION")
}

func findVersionFile() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Debugf("Couldn't determine current working directory: %v", err)
		return ""
	}

	currentDir := dir
	for {
		versionFileName := filepath.Join(currentDir, ".go-version")
		versionFile, err := os.Open(versionFileName)
		if err == nil {
			scanner := bufio.NewScanner(versionFile)
			version := ""
			if scanner.Scan() { // Read a single line
				version = scanner.Text()
			}
			_ = versionFile.Close()
			if version != "" {
				log.Debugf("Using version specification from %v", versionFileName)
				return version
			}
		}

		currentDir = filepath.Dir(currentDir)
		if currentDir == filepath.Dir(currentDir) {
			log.Debugf("Couldn't find any `.go-version` file in tree %v", dir)
			return ""
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func setGoRoot(baseDir string) {
	os.Setenv("GOROOT", filepath.Join(baseDir, "go"))
}

func downloadFile(url, fileToSave string) error {
	dir := filepath.Dir(fileToSave)
	err := os.MkdirAll(dir, os.ModeDir|0755)
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
