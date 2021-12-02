package config

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type localFile struct {
	name     string
	fullPath string
}

func writeFiles(root string, files []*file) ([]*localFile, error) {
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("root %q must be an absolute path but is not", root)
	}
	if err := os.MkdirAll(root, 0o777); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}

	localFiles := make([]*localFile, 0, len(files))
	for _, of := range files {
		lf, err := writeFile(root, of)
		if err != nil {
			return nil, err
		}
		localFiles = append(localFiles, lf)
	}

	return localFiles, nil
}

func writeFile(root string, n *file) (*localFile, error) {
	if filepath.IsAbs(n.Path) {
		return nil, fmt.Errorf("file.path error: %q file must be relative path: %q", n.Name, n.Path)
	}
	path := filepath.Clean(n.Path)
	if strings.HasPrefix(path, "../") {
		return nil, fmt.Errorf("%q refers to file outside root", n.Path)
	}
	lf := &localFile{
		name:     n.Name,
		fullPath: filepath.Join(root, path),
	}
	if err := os.MkdirAll(filepath.Dir(lf.fullPath), 0777); err != nil {
		return nil, fmt.Errorf("unable to create dir: %w", err)
	}
	content := []byte(n.Content)
	file, err := os.Open(lf.fullPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("cannot open old file: %v", err)
		}
		if err := ioutil.WriteFile(lf.fullPath, content, 0777); err != nil {
			return nil, fmt.Errorf("cannot create new file: %v", err)
		}
		return lf, nil
	}
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	file.Close()
	if err != nil {
		return nil, fmt.Errorf("cannot read old file contents: %v", err)
	}
	var oldSum [sha256.Size]byte
	hash.Sum(oldSum[:0])
	newSum := sha256.Sum256(content)
	if newSum == oldSum {
		return lf, nil
	}
	if err := ioutil.WriteFile(lf.fullPath, content, 0o777); err != nil {
		return nil, fmt.Errorf("cannot create new file: %v", err)
	}
	return lf, nil
}
