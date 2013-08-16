/*
Copyright 2013 Manu Goyal

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License.  You may obtain a copy of the
License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied.  See the License for the
specific language governing permissions and limitations under the License.
*/

// Miscellaneous utility functions

package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type filePair struct {
	path string
	fi   os.FileInfo
}

type filterFunc func(filePair) bool

func doWalkDir(path string, fileChan chan filePair, filter filterFunc) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	fp := filePair{path, info}
	if !filter(fp) {
		return nil
	}
	fileChan <- fp
	if info.IsDir() {
		dir, err := os.Open(path)
		if err != nil {
			return err
		}
		// Recurse into the subdirectories
		subfiles, err := dir.Readdirnames(0)
		if err != nil {
			return err
		}
		dir.Close()
		for _, subfile := range subfiles {
			if err := doWalkDir(filepath.Join(path, subfile), fileChan, filter); err != nil {
				return err
			}
		}
	}
	return nil
}

// Traverses the files at a location, recursing into subdirectories,
// and adding every file and directory that passes a given filter as
// an absolute path onto a channel. If a directory fails the filter,
// it skips the entire directory. The only reason I want to use this
// rather than filepath.Walk is that filepath.Walk sorts the
// directories, which is unnecessary for this and thus causes a
// slowdown
func walkDir(path string, fileChan chan filePair, filter filterFunc) error {
	if err := doWalkDir(filepath.Clean(path), fileChan, filter); err != nil {
		return err
	}
	close(fileChan)
	return nil
}

// tars all the files in a directory, recursing into subdirectories as
// well. It skips dotfiles and symlinks. dir is the absolute path of
// the directory needing to be compressed
func tarDir(dirPath string, tw *tar.Writer) error {
	fileChan := make(chan filePair)
	go walkDir(dirPath, fileChan, func(fp filePair) bool {
		if filepath.Base(fp.path)[0] == '.' || uint32(fp.fi.Mode()&os.ModeSymlink) > 0 {
			return false
		}
		return true
	})
	// The tar needs to write headers that include the name of the
	// directory we are compressiong, so it can decompress into
	// that directory again
	baseDirPath := filepath.Dir(dirPath)
	for fp := range fileChan {
		if fp.fi.IsDir() {
			continue
		}
		th, err := tar.FileInfoHeader(fp.fi, "")
		if err != nil {
			return err
		}
		// Sets name to the name rooted by the baseDirPath
		if th.Name, err = filepath.Rel(baseDirPath, fp.path); err != nil {
			return err
		}
		if err := tw.WriteHeader(th); err != nil {
			return fmt.Errorf("Error while writing file %s: %s", th.Name, err)
		}

		f, err := os.Open(fp.path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("Error while writing file %s: %s", th.Name, err)
		}
		f.Close()
	}
	return nil
}
