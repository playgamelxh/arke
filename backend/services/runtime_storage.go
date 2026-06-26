package services

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type RuntimeStorage struct {
	settings *SettingsService
}

func NewRuntimeStorage(settings *SettingsService) *RuntimeStorage {
	return &RuntimeStorage{settings: settings}
}

func (s *RuntimeStorage) PutObject(objectKey string, reader io.Reader, size int64, contentType string) error {
	storage, err := s.resolve()
	if err != nil {
		return err
	}
	return storage.PutObject(objectKey, reader, size, contentType)
}

func (s *RuntimeStorage) DownloadToTemp(objectKey, fileType string) (string, func(), error) {
	primary, fallback, err := s.resolveWithFallback()
	if err != nil {
		return "", func() {}, err
	}
	path, cleanup, err := primary.DownloadToTemp(objectKey, fileType)
	if err == nil {
		return path, cleanup, nil
	}
	if fallback != nil {
		if path, cleanup, fallbackErr := fallback.DownloadToTemp(objectKey, fileType); fallbackErr == nil {
			return path, cleanup, nil
		}
	}
	return "", func() {}, err
}

func (s *RuntimeStorage) RemoveObject(objectKey string) error {
	primary, fallback, err := s.resolveWithFallback()
	if err != nil {
		return err
	}
	if err := primary.RemoveObject(objectKey); err == nil {
		return nil
	}
	if fallback != nil {
		return fallback.RemoveObject(objectKey)
	}
	return nil
}

func (s *RuntimeStorage) resolveWithFallback() (StorageInterface, StorageInterface, error) {
	settings, err := s.settings.GetSettings()
	if err != nil {
		return nil, nil, err
	}
	localDir := strings.TrimSpace(settings["localUploadDir"])
	if localDir == "" {
		localDir = "/app/uploads"
	}
	localStorage, localErr := NewLocalStorageClient(localDir)
	if settings["storageMode"] == "rustfs" {
		rustfsStorage, err := NewStorageClientWithOptions(s.settings.StorageOptionsFromSettings(settings))
		if err != nil {
			if localErr != nil {
				return nil, nil, fmt.Errorf("RustFS 存储不可用：%w；本地存储也不可用：%v", err, localErr)
			}
			return nil, localStorage, fmt.Errorf("RustFS 存储不可用：%w", err)
		}
		return rustfsStorage, localStorage, nil
	}
	if localErr != nil {
		return nil, nil, localErr
	}
	rustfsStorage, _ := NewStorageClientWithOptions(s.settings.StorageOptionsFromSettings(settings))
	return localStorage, rustfsStorage, nil
}

func (s *RuntimeStorage) resolve() (StorageInterface, error) {
	settings, err := s.settings.GetSettings()
	if err != nil {
		return nil, err
	}
	if settings["storageMode"] == "rustfs" {
		storage, err := NewStorageClientWithOptions(s.settings.StorageOptionsFromSettings(settings))
		if err != nil {
			return nil, fmt.Errorf("RustFS 存储不可用：%w", err)
		}
		return storage, nil
	}
	baseDir := strings.TrimSpace(settings["localUploadDir"])
	if baseDir == "" {
		baseDir = "/app/uploads"
	}
	return NewLocalStorageClient(baseDir)
}

func (s *RuntimeStorage) TestRustFS(values map[string]string) error {
	settings, err := s.settings.GetSettings()
	if err != nil {
		return err
	}
	for key, value := range values {
		if isRustFSSettingKey(key) {
			settings[key] = value
		}
	}
	storage, err := NewStorageClientWithOptions(s.settings.StorageOptionsFromSettings(settings))
	if err != nil {
		return err
	}
	testKey := "system/storage-test/.keep"
	content := strings.NewReader("ok")
	if err := storage.PutObject(testKey, content, int64(content.Len()), "text/plain"); err != nil {
		return err
	}
	path, cleanup, err := storage.DownloadToTemp(testKey, "txt")
	if err != nil {
		return err
	}
	defer cleanup()
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return storage.RemoveObject(testKey)
}

func isRustFSSettingKey(key string) bool {
	switch key {
	case "rustfsEndpoint", "rustfsAccessKey", "rustfsSecretKey", "rustfsBucket", "rustfsRegion", "rustfsUseSSL":
		return true
	default:
		return false
	}
}
