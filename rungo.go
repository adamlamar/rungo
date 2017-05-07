package rungo

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

func RunGo(baseDir string, args []string) error {
	goBinary := filepath.Join(baseDir, "go", "bin", "go")
	cmd := exec.Command(goBinary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Debugf("Executing %q with arguments %v", goBinary, args)
	return cmd.Run()
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func SetGoRoot(baseDir string) {
	os.Setenv("GOROOT", baseDir)
}

func DownloadFile(url, fileToSave string) error {
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
