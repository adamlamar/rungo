// +build !windows

package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

func extractFile(golangArchive, baseDir string) error {
	log.Debugf("Extracting %q", golangArchive)
	err := os.MkdirAll(baseDir, os.ModeDir|0755)
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
			err = os.MkdirAll(filepath.Dir(path), os.ModeDir|0755)
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
