package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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

// Returns the sha256 for this url. Downloads if necessary
func fetchSha256(url, fileToSave string) (string, error) {
	dir := filepath.Dir(fileToSave)
	err := os.MkdirAll(dir, os.ModeDir|0755)
	if err != nil {
		return "", errors.Wrapf(err, "mkdir %q failed", dir)
	}

	// Return the hex-encoded sha256, which is double the size of the unencoded version
	shaSum := make([]byte, sha256.Size*2)
	file, err := os.OpenFile(fileToSave, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0755)

	// If the file exists, re-open to read the sha256 from the file.
	// Otherwise download and write to disk.
	if os.IsExist(err) {
		log.Debugf("Found sha256 file at %q", fileToSave)
		shaFile, err := os.Open(fileToSave)
		if err != nil {
			return "", errors.Wrap(err, "failed to open sha256 file")
		}
		defer shaFile.Close()

		_, err = io.ReadAtLeast(shaFile, shaSum, len(shaSum))
		if err != nil {
			return "", errors.Wrap(err, "could not read sha256 from file")
		}
		return string(shaSum), nil
	} else if err != nil {
		return "", errors.Wrapf(err, "open %q failed", fileToSave)
	}
	defer file.Close()

	log.Debugf("Downloading sha256 file %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "download of sha failed")
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return "", fmt.Errorf("failed due to non-2XX response: %q", resp.Status)
	}
	defer resp.Body.Close()

	_, err = io.ReadAtLeast(resp.Body, shaSum, len(shaSum))
	if err != nil {
		return "", errors.Wrap(err, "could not read bytes from response into buffer")
	}

	// Write the sha to the file we opened earlier
	_, err = file.Write(shaSum)
	if err != nil {
		// Technically we could continue since we have the sha256. This fails early instead.
		return "", errors.Wrap(err, "failed to write sha256")
	}
	return string(shaSum), nil
}

func downloadFile(url, expectedSha256, fileToSave string) error {
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

	// Write to disk and calculate sha256
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return errors.Wrap(err, "copy to disk failed")
	}

	// Compare expected and actual sha256
	actualSha256 := hex.EncodeToString(hasher.Sum(nil))
	if actualSha256 != expectedSha256 {
		return fmt.Errorf("failed to verify archive from %s: expected sha256 %s but calculated %s", url, expectedSha256, actualSha256)
	}

	// if download is complete, write the canary file for success
	ioutil.WriteFile(canaryFile, []byte(""), 0755)
	log.Debugf("Successfully downloaded %s with sha256 %s", url, actualSha256)
	return nil
}
