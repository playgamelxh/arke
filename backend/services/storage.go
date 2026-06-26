package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"arke/backend/config"
)

type StorageOptions struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
}

type StorageClient struct {
	client *minio.Client
	bucket string
}

func NewStorageClient(cfg config.Config) (*StorageClient, error) {
	return NewStorageClientWithOptions(StorageOptions{
		Endpoint:  cfg.S3Endpoint,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
		Bucket:    cfg.S3Bucket,
		Region:    cfg.S3Region,
		UseSSL:    cfg.S3UseSSL,
	})
}

func NewStorageClientWithOptions(options StorageOptions) (*StorageClient, error) {
	endpoint := strings.TrimSpace(options.Endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimRight(endpoint, "/")
	if endpoint == "" {
		return nil, fmt.Errorf("S3_ENDPOINT 未配置")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(options.AccessKey, options.SecretKey, ""),
		Secure: options.UseSSL,
		Region: options.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化对象存储客户端失败：%w", err)
	}

	sc := &StorageClient{client: client, bucket: options.Bucket}
	if err := sc.ensureBucket(); err != nil {
		return nil, err
	}
	return sc, nil
}

func (s *StorageClient) ensureBucket() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var lastErr error
	for i := 0; i < 5; i++ {
		exists, err := s.client.BucketExists(ctx, s.bucket)
		if err != nil {
			lastErr = err
			time.Sleep(1 * time.Second)
			continue
		}
		if exists {
			return nil
		}
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			exists, existsErr := s.client.BucketExists(ctx, s.bucket)
			if existsErr == nil && exists {
				return nil
			}
			lastErr = err
			time.Sleep(1 * time.Second)
			continue
		}
		return nil
	}
	return fmt.Errorf("连接对象存储或创建存储桶失败：%w", lastErr)
}

func (s *StorageClient) PutObject(objectKey string, reader io.Reader, size int64, contentType string) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	_, err := s.client.PutObject(ctx, s.bucket, objectKey, reader, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *StorageClient) DownloadToTemp(objectKey, fileType string) (string, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	object, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return "", func() {}, err
	}
	defer object.Close()

	suffix := ""
	if fileType != "" {
		suffix = "." + fileType
	}
	tmp, err := os.CreateTemp("", "arke-doc-*"+suffix)
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.Remove(tmp.Name()) }

	if _, err := io.Copy(tmp, object); err != nil {
		tmp.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return tmp.Name(), cleanup, nil
}

func (s *StorageClient) RemoveObject(objectKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

// LocalStorageClient 本地文件存储（S3 不可用时的后备方案）
type LocalStorageClient struct {
	baseDir string
}

func NewLocalStorageClient(baseDir string) (*LocalStorageClient, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("创建本地存储目录失败：%w", err)
	}
	return &LocalStorageClient{baseDir: baseDir}, nil
}

func (s *LocalStorageClient) PutObject(objectKey string, reader io.Reader, size int64, contentType string) error {
	filePath := filepath.Join(s.baseDir, objectKey)
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败：%w", err)
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败：%w", err)
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	return err
}

func (s *LocalStorageClient) DownloadToTemp(objectKey, fileType string) (string, func(), error) {
	filePath := filepath.Join(s.baseDir, objectKey)
	src, err := os.Open(filePath)
	if err != nil {
		return "", func() {}, fmt.Errorf("打开文件失败：%w", err)
	}
	defer src.Close()

	suffix := ""
	if fileType != "" {
		suffix = "." + fileType
	}
	tmp, err := os.CreateTemp("", "arke-doc-*"+suffix)
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.Remove(tmp.Name()) }

	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return tmp.Name(), cleanup, nil
}

func (s *LocalStorageClient) RemoveObject(objectKey string) error {
	filePath := filepath.Join(s.baseDir, objectKey)
	return os.Remove(filePath)
}

// StorageInterface 定义存储接口
type StorageInterface interface {
	PutObject(objectKey string, reader io.Reader, size int64, contentType string) error
	DownloadToTemp(objectKey, fileType string) (string, func(), error)
	RemoveObject(objectKey string) error
}
