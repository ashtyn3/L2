package storage

import (
	"io"
	"os"
	"path/filepath"
)

func ReadSystem() (string, error) {
	if exists, err := CheckFile(SystemFile); err != nil {
		return "", err
	} else if !exists {
		err := CopySystem()
		if err != nil {
			return "", err
		}
	}

	data, err := ReadFile(SystemFile)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func CopySystem() error {
	systemPath, err := GetPath(SystemFile)
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(systemPath), 0755)
	systemFile, err := os.OpenFile(systemPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	localPath, err := os.Open("system.md")
	if err != nil {
		return err
	}
	io.Copy(systemFile, localPath)
	return nil
}
