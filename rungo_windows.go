package main

import (
	"archive/zip"
	"io"
	"os"
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

func extractFile(golangArchive, baseDir string) error {
	log.Debugf("Extracting %q", golangArchive)
	err := os.MkdirAll(baseDir, os.ModeDir|0755)
	if err != nil {
		return errors.Wrapf(err, "mkdir %q failed", baseDir)
	}

	zipReader, err := zip.OpenReader(golangArchive)
	if err != nil {
		return errors.Wrapf(err, "open reader for file %q failed", golangArchive)
	}
	defer zipReader.Close()

	fileCount := 0
	for _, fileInZip := range zipReader.File {
		fileContents, err := fileInZip.Open()
		if err != nil {
			return errors.Wrapf(err, "zip open of %q failed", fileInZip.Name)
		}

		path := filepath.Join(baseDir, fileInZip.Name)
		fileInfo := fileInZip.FileHeader.FileInfo()
		if fileInfo.IsDir() {
			err = os.MkdirAll(path, fileInfo.Mode())
			if err != nil {
				return errors.Wrapf(err, "mkdir %q failed", path)
			}
			continue
		} else {
			// Make directory containing the current file, if needed. Some tarballs don't include the top-level directory entry
			err = os.MkdirAll(filepath.Dir(path), os.ModeDir|0755)
			if err != nil {
				return errors.Wrapf(err, "mkdir %q failed", path)
			}
		}

		fileOnDisk, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileInfo.Mode())
		if err != nil {
			return errors.Wrapf(err, "open file %q failed", path)
		}

		_, err = io.Copy(fileOnDisk, fileContents)
		if err != nil {
			fileOnDisk.Close()
			return errors.Wrapf(err, "copy for %q failed", path)
		}
		fileOnDisk.Close()
		fileCount++
	}
	log.Debugf("Wrote %d files to %q", fileCount, baseDir)

	return nil
}
