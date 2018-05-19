package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

const (
	DOWNLOADED_CANARY = "go-downloaded"
)

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

	return DEFAULT_GOLANG
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

// could refactor this to download bytes and be re-usable for the tar file itself
func downloadSha(url string) (string, error) {
	log.Debugf("Downloading sha file %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "download of sha failed")
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return "", fmt.Errorf("failed due to non-2XX response: %q", resp.Status)
	}
	defer resp.Body.Close()

	fileBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "could not read bytes from response into buffer")
	}

	return fmt.Sprintf("%s", fileBody), nil
}

func downloadFile(url, fileToSave string) error {
	dir := filepath.Dir(fileToSave)
	canaryFile := filepath.Join(dir, DOWNLOADED_CANARY)
	err := os.MkdirAll(dir, os.ModeDir|0755)
	if err != nil {
		return errors.Wrapf(err, "mkdir %q failed", dir)
	}

	file, err := os.OpenFile(fileToSave, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0755)
	if os.IsExist(err) {
		if fileExists(canaryFile) {
			log.Debugf("File %q already exists, skipping download", fileToSave)
			return nil
		}
		log.Infof("File %q exists, but was not fully downloaded, so will re-download", fileToSave)
		file, err = os.OpenFile(fileToSave, os.O_CREATE|os.O_RDWR, 0755)
		if err != nil {
			return errors.Wrapf(err, "open partially-downloaded %q failed", fileToSave)
		}
	} else if err != nil {
		return errors.Wrapf(err, "open %q failed", fileToSave)
	}
	defer file.Close()

	// Download the expected shasum
	expectedSum, err := downloadSha(url + ".sha256")
	if err != nil {
		return errors.Wrap(err, "download expected sha256 failed")
	}
	log.Debugf("Expected sum: %s", expectedSum)

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

	// Read file to buffer, then write to disk
	fileBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "could not read bytes from response into buffer")
	}
	_, err = file.Write(fileBody)
	if err != nil {
		return errors.Wrap(err, "copy to disk failed")
	}

	// Calculate the shasum for the downloaded file, and compare to expected
	sumBytes := sha256.Sum256(fileBody)
	sumStr := fmt.Sprintf("%x", sumBytes)
	log.Debugf("Calculated Sum: %s", sumStr)

	if expectedSum != sumStr {
		return fmt.Errorf("Downloaded SHA256 did not match.  Expected %s but calculated %s", expectedSum, sumStr)
	}

	// if download is complete, write the canary file for success
	ioutil.WriteFile(canaryFile, []byte(""), 0755)
	return nil
}
