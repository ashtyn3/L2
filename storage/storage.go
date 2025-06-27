package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino/schema"
)

const (
	conversationFilePath = "conversations/conversation.json"
	systemFilePath       = "system.md"
	statsFilePath        = "stats.json"
	rootPath             = "l2"
	dataPath             = "data"
)

var pathMap = map[int]string{
	0: systemFilePath,
	1: conversationFilePath,
	2: statsFilePath,
	3: dataPath,
}

const (
	SystemFile = iota
	ConversationFile
	StatsFile
	DataFile
)

func GetPath(file int) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s", home, rootPath, pathMap[file]), nil
}

func WriteDataFile(file string, data []byte) error {
	Path, err := GetPath(DataFile)
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(Path), 0755)
	Path = filepath.Join(Path, file)
	return os.WriteFile(Path, data, 0644)
}
func ReadDataFile(file string) ([]byte, error) {
	Path, err := GetPath(DataFile)
	if err != nil {
		return nil, err
	}
	Path = filepath.Join(Path, file)
	return os.ReadFile(Path)
}
func WriteFile(file int, data []byte) error {
	path, err := GetPath(file)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func ReadFile(file int) ([]byte, error) {
	exists, err := CheckFile(file)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("file does not exist: %s", pathMap[file])
	}
	path, err := GetPath(file)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func ReadConversation() ([]*schema.Message, error) {
	data, err := ReadFile(ConversationFile)
	if err != nil {
		return nil, err
	}
	var history []*schema.Message
	err = json.Unmarshal(data, &history)
	if err != nil {
		return nil, err
	}
	return history, nil
}

func WriteConversation(history []*schema.Message) error {
	exists, err := CheckFile(ConversationFile)
	if err != nil {
		return err
	}
	if !exists {
		os.MkdirAll(filepath.Dir(pathMap[ConversationFile]), 0755)
	}
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}
	return WriteFile(ConversationFile, data)
}

func CheckFile(file int) (bool, error) {
	path, err := GetPath(file)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	return err == nil, nil
}

type Stats struct {
	TotalTokens int `json:"total_tokens"`
}

func ReadStats() (Stats, error) {
	data, err := ReadFile(StatsFile)
	if err != nil {
		return Stats{TotalTokens: 0}, err
	}
	var stats Stats
	err = json.Unmarshal(data, &stats)
	if err != nil {
		return Stats{TotalTokens: 0}, err
	}
	return stats, nil
}

func WriteStats(stats Stats) error {
	exists, err := CheckFile(StatsFile)
	if err != nil {
		return err
	}
	if !exists {
		os.MkdirAll(filepath.Dir(pathMap[StatsFile]), 0755)
	}
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	return WriteFile(StatsFile, data)
}
