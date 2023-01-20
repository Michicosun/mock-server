package util

import (
	"errors"
	"os"
	"path/filepath"
)

type fileStorageDriver struct {
	prefix string
}

const (
	file_storage_dir_name = ".storage"
)

func createIfNotExists(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(path, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func createFileStorage() (*fileStorageDriver, error) {
	file_storage_root, err := FileStorageRoot()
	if err != nil {
		return nil, err
	}

	return &fileStorageDriver{
		prefix: file_storage_root,
	}, nil
}

func FileStorageRoot() (string, error) {
	root, err := GetProjectRoot()
	if err != nil {
		return "", err
	}

	storage_root := filepath.Join(root, file_storage_dir_name)
	if err := createIfNotExists(storage_root); err != nil {
		return "", err
	}

	file_storage_root, err := filepath.Abs(storage_root)
	if err != nil {
		return "", err
	}
	return file_storage_root, nil
}

func NewFileStorageDriver(prefix string) (*fileStorageDriver, error) {
	driver, err := createFileStorage()
	if err != nil {
		return nil, err
	}

	root := filepath.Join(driver.prefix, prefix)
	if err := createIfNotExists(root); err != nil {
		return nil, err
	}

	driver.prefix = root
	return driver, nil
}

func (fs *fileStorageDriver) Read(prefix string, filename string) (string, error) {
	full_path := filepath.Join(fs.prefix, prefix, filename)

	file, err := os.ReadFile(full_path)
	if err != nil {
		return "", err
	}

	return string(file), nil
}

func (fs *fileStorageDriver) Write(prefix string, filename string, data []byte) error {
	folder := filepath.Join(fs.prefix, prefix)
	err := createIfNotExists(folder)
	if err != nil {
		return err
	}

	full_path := filepath.Join(folder, filename)

	file, err := os.Create(full_path)
	if err != nil {
		return err
	}

	cnt_read := 0
	for cnt_read < len(data) {
		n, err := file.Write(data)
		if err != nil {
			return err
		}
		cnt_read += n
	}

	return nil
}