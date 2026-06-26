package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Port             string
	DatabaseDSN      string
	MigrationURL     string
	MigrationPath    string
	AllowedOrigins   []string
	DashScopeAPIKey  string
	DashScopeBaseURL string
	DashScopeModel   string
	DashScopeTimeout time.Duration
	EmbeddingModel   string
	EmbeddingDim     int
	MinerUBaseURL    string
	MinerUTimeout    time.Duration
	S3Endpoint       string
	S3AccessKey      string
	S3SecretKey      string
	S3Bucket         string
	S3Region         string
	S3UseSSL         bool
	MilvusAddress    string
}

type yamlConfig struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		DSN           string `yaml:"dsn"`
		MigrationURL  string `yaml:"migration_url"`
		MigrationPath string `yaml:"migration_path"`
	} `yaml:"database"`
	CORS struct {
		AllowedOrigins []string `yaml:"allowed_origins"`
	} `yaml:"cors"`
	DashScope struct {
		APIKey         string `yaml:"api_key"`
		BaseURL        string `yaml:"base_url"`
		Model          string `yaml:"model"`
		TimeoutSeconds int    `yaml:"timeout_seconds"`
		EmbeddingModel string `yaml:"embedding_model"`
		EmbeddingDim   int    `yaml:"embedding_dim"`
	} `yaml:"dashscope"`
	MinerU struct {
		BaseURL        string `yaml:"base_url"`
		TimeoutSeconds int    `yaml:"timeout_seconds"`
	} `yaml:"mineru"`
	S3 struct {
		Endpoint  string `yaml:"endpoint"`
		AccessKey string `yaml:"access_key"`
		SecretKey string `yaml:"secret_key"`
		Bucket    string `yaml:"bucket"`
		Region    string `yaml:"region"`
		UseSSL    bool   `yaml:"use_ssl"`
	} `yaml:"s3"`
	Milvus struct {
		Address string `yaml:"address"`
	} `yaml:"milvus"`
}

func Load() Config {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	configPath := findConfigFile(env)
	log.Printf("Loading config from: %s", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var yamlCfg yamlConfig
	if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	return convertToConfig(yamlCfg)
}

func findConfigFile(env string) string {
	candidates := []string{
		fmt.Sprintf("config/config.%s.yaml", env),
		fmt.Sprintf("../config/config.%s.yaml", env),
		fmt.Sprintf("../../config/config.%s.yaml", env),
		"config/config.yaml",
		"../config/config.yaml",
		"../../config/config.yaml",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	log.Fatalf("Config file not found for env: %s", env)
	return ""
}

func convertToConfig(yamlCfg yamlConfig) Config {
	embModel := yamlCfg.DashScope.EmbeddingModel
	if embModel == "" {
		embModel = "text-embedding-v3"
	}
	embDim := yamlCfg.DashScope.EmbeddingDim
	if embDim == 0 {
		embDim = 1024
	}
	cfg := Config{
		Port:             yamlCfg.Server.Port,
		DatabaseDSN:      yamlCfg.Database.DSN,
		MigrationURL:     yamlCfg.Database.MigrationURL,
		MigrationPath:    yamlCfg.Database.MigrationPath,
		AllowedOrigins:   yamlCfg.CORS.AllowedOrigins,
		DashScopeAPIKey:  yamlCfg.DashScope.APIKey,
		DashScopeBaseURL: yamlCfg.DashScope.BaseURL,
		DashScopeModel:   yamlCfg.DashScope.Model,
		DashScopeTimeout: parseTimeout(yamlCfg.DashScope.TimeoutSeconds),
		EmbeddingModel:   embModel,
		EmbeddingDim:     embDim,
		MinerUBaseURL:    yamlCfg.MinerU.BaseURL,
		MinerUTimeout:    parseTimeout(yamlCfg.MinerU.TimeoutSeconds),
		S3Endpoint:       yamlCfg.S3.Endpoint,
		S3AccessKey:      yamlCfg.S3.AccessKey,
		S3SecretKey:      yamlCfg.S3.SecretKey,
		S3Bucket:         yamlCfg.S3.Bucket,
		S3Region:         yamlCfg.S3.Region,
		S3UseSSL:         yamlCfg.S3.UseSSL,
		MilvusAddress:    yamlCfg.Milvus.Address,
	}
	applyEnvOverrides(&cfg)
	return cfg
}

func applyEnvOverrides(cfg *Config) {
	if value := os.Getenv("S3_ENDPOINT"); value != "" {
		cfg.S3Endpoint = value
	}
	if value := os.Getenv("S3_ACCESS_KEY"); value != "" {
		cfg.S3AccessKey = value
	}
	if value := os.Getenv("S3_SECRET_KEY"); value != "" {
		cfg.S3SecretKey = value
	}
	if value := os.Getenv("S3_BUCKET"); value != "" {
		cfg.S3Bucket = value
	}
	if value := os.Getenv("S3_REGION"); value != "" {
		cfg.S3Region = value
	}
	if value := os.Getenv("S3_USE_SSL"); value != "" {
		cfg.S3UseSSL = value == "true"
	}
}

func parseTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		return 300 * time.Second
	}
	if seconds > 300 {
		seconds = 300
	}
	return time.Duration(seconds) * time.Second
}

func GetConfigDir() string {
	candidates := []string{"config", "../config", "../../config"}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			return absPath
		}
	}
	return "./config"
}
