// Functions for tarring a directory

package main

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
)

// Writes the given file to the tar
func tarFile(rootPath, dirPath string, fi os.FileInfo, tw *tar.Writer) error {
	relFilepath := filepath.Join(dirPath, fi.Name())
	f, err := os.Open(filepath.Join(rootPath, relFilepath))
	if err != nil {
		return err
	}
	defer f.Close()
	th, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}
	// Sets name to the name rooted by the dirPath
	th.Name = relFilepath
	if err := tw.WriteHeader(th); err != nil {
		return err
	}
	if _, err := io.Copy(tw, f); err != nil {
		return err
	}
	return nil
}

// tars an entire directory, recursing into subdirectories as well.
// rootPath + dirPath is the absolute path of the directory, and is
// used to find files to open, but dirPath is the actual directory
// being compressed and is written to the tar, so it can be expanded
// anywhere.
func compressDir(rootPath, dirName string, tw *tar.Writer) error {
	dir, err := os.Open(filepath.Join(rootPath, dirName))
	if err != nil {
		return err
	}
	defer dir.Close()
	files, err := dir.Readdir(0)
	for _, fi := range files {
		switch {
		case fi.Name()[0] == '.':
		case fi.IsDir():
			if err := compressDir(rootPath, filepath.Join(dirName, fi.Name()), tw); err != nil {
				return err
			}
		default:
			if err := tarFile(rootPath, dirName, fi, tw); err != nil {
				return err
			}
		}
	}
	return nil
}
