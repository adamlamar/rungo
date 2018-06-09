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

	// If the file exists, re-open to read the sha256 from the file.
	// Otherwise download and write to disk.
	file, err := os.OpenFile(fileToSave, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return "", errors.Wrapf(err, "open %q failed", fileToSave)
	}
	defer file.Close()

	if err != nil {
		return "", errors.Wrap(err, "failed to open sha256 file")
	}

	_, err = io.ReadAtLeast(file, shaSum, len(shaSum))
	if err == nil {
		return string(shaSum), nil
	}
	log.Debugf("Failed to read sha256 file: %v", err)

	log.Infof("Downloading sha256 file %s", url)
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
	err := os.MkdirAll(dir, os.ModeDir|0755)
	if err != nil {
		return errors.Wrapf(err, "mkdir %q failed", dir)
	}

	_, err = os.Stat(fileToSave)
	if !os.IsNotExist(err) {
		log.Debugf("File %q already exists, skipping download", fileToSave)
		return nil // file exists
	}

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

	// Open in-progress file to hold partial contents
	inProgressFile, err := ioutil.TempFile(dir, "golang-download")
	if err != nil {
		return errors.Wrap(err, "failed to open in-progress download file")
	}

	// Write to disk and calculate sha256
	hasher := sha256.New()
	writer := io.MultiWriter(inProgressFile, hasher)

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return errors.Wrap(err, "copy to disk failed")
	}
	err = inProgressFile.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close in-progress download file")
	}

	// Compare expected and actual sha256
	actualSha256 := hex.EncodeToString(hasher.Sum(nil))
	if actualSha256 != expectedSha256 {
		return fmt.Errorf("failed to verify archive from %s: expected sha256 %s but calculated %s", url, expectedSha256, actualSha256)
	}

	// if download is complete, move the in-progress file to complete
	err = os.Rename(inProgressFile.Name(), fileToSave)
	if err != nil {
		return errors.Wrap(err, "failed to move in-progress file")
	}
	log.Debugf("Successfully downloaded %s with sha256 %s", url, actualSha256)
	return nil
}
