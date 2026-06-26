package services

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"arke/backend/config"
	"arke/backend/models"
)

type SettingsService struct {
	db  *gorm.DB
	cfg config.Config
}

func NewSettingsService(db *gorm.DB, cfg config.Config) *SettingsService {
	return &SettingsService{db: db, cfg: cfg}
}

func (s *SettingsService) Defaults() map[string]string {
	return map[string]string{
		"maxFileSizeMB":          "200",
		"allowedTypes":           "pdf,ppt,pptx,xls,xlsx,png,jpg,jpeg,doc,docx,md,txt",
		"defaultQACount":         "10",
		"qaGenerateBatchSize":    "10",
		"modelEndpoint":          s.cfg.DashScopeBaseURL,
		"modelName":              s.cfg.DashScopeModel,
		"storageMode":            "local",
		"localUploadDir":         "/app/uploads",
		"rustfsEndpoint":         s.cfg.S3Endpoint,
		"rustfsAccessKey":        s.cfg.S3AccessKey,
		"rustfsSecretKey":        s.cfg.S3SecretKey,
		"rustfsBucket":           s.cfg.S3Bucket,
		"rustfsRegion":           s.cfg.S3Region,
		"rustfsUseSSL":           strconv.FormatBool(s.cfg.S3UseSSL),
		"parseEngine":            "auto",
		"parsePDFNativeFallback": "true",
		"mineruBaseURL":          s.cfg.MinerUBaseURL,
		"mineruTimeoutSeconds":   strconv.Itoa(int(s.cfg.MinerUTimeout.Seconds())),
		"mineruParseMethod":      "auto",
		"mineruEffort":           "medium",
		"mineruLanguage":         "ch",
		"mineruImageAnalysis":    "true",
		"mineruTableEnable":      "true",
		"mineruFormulaEnable":    "true",
	}
}

func (s *SettingsService) GetSettings() (map[string]string, error) {
	settings := s.Defaults()
	var rows []models.SystemSetting
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		settings[row.SettingKey] = row.SettingValue
	}
	return settings, nil
}

func (s *SettingsService) UpdateSettings(values map[string]string) (map[string]string, error) {
	if _, err := s.validateSettings(values, true); err != nil {
		return nil, err
	}
	return s.GetSettings()
}

func (s *SettingsService) validateSettings(values map[string]string, save bool) (map[string]string, error) {
	defaults := s.Defaults()
	for key, value := range values {
		if _, exists := defaults[key]; !exists {
			continue
		}
		value = strings.TrimSpace(value)
		if key == "storageMode" && value != "local" && value != "rustfs" {
			return nil, fmt.Errorf("文档上传位置只支持 local 或 rustfs")
		}
		if key == "rustfsUseSSL" || key == "parsePDFNativeFallback" || key == "mineruImageAnalysis" || key == "mineruTableEnable" || key == "mineruFormulaEnable" {
			if value != "true" && value != "false" {
				return nil, fmt.Errorf("%s 只能为 true 或 false", key)
			}
		}
		if key == "parseEngine" && value != "auto" && value != "mineru" && value != "native" {
			return nil, fmt.Errorf("文档解析引擎只支持 auto、mineru 或 native")
		}
		if key == "mineruParseMethod" && value != "auto" && value != "txt" && value != "ocr" {
			return nil, fmt.Errorf("MinerU 解析方法只支持 auto、txt 或 ocr")
		}
		if key == "mineruEffort" && value != "low" && value != "medium" && value != "high" {
			return nil, fmt.Errorf("MinerU 解析精度只支持 low、medium 或 high")
		}
		if key == "mineruTimeoutSeconds" {
			seconds, err := strconv.Atoi(value)
			if err != nil || seconds < 1 || seconds > 600 {
				return nil, fmt.Errorf("MinerU 超时时间需在 1-600 秒之间")
			}
		}
		if save {
			row := models.SystemSetting{SettingKey: key, SettingValue: value}
			if err := s.db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "setting_key"}},
				DoUpdates: clause.AssignmentColumns([]string{"setting_value"}),
			}).Create(&row).Error; err != nil {
				return nil, err
			}
		}
	}
	if save {
		return s.GetSettings()
	}
	return values, nil
}

func (s *SettingsService) ParseOptionsFromSettings(values map[string]string) ParseOptions {
	defaults := s.Defaults()
	merged := make(map[string]string, len(defaults))
	for key, value := range defaults {
		merged[key] = value
	}
	for key, value := range values {
		merged[key] = value
	}
	timeoutSeconds, _ := strconv.Atoi(merged["mineruTimeoutSeconds"])
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300
	}
	pdfFallback, _ := strconv.ParseBool(merged["parsePDFNativeFallback"])
	imageAnalysis, _ := strconv.ParseBool(merged["mineruImageAnalysis"])
	tableEnable, _ := strconv.ParseBool(merged["mineruTableEnable"])
	formulaEnable, _ := strconv.ParseBool(merged["mineruFormulaEnable"])
	return ParseOptions{
		Engine:            merged["parseEngine"],
		PDFNativeFallback: pdfFallback,
		MinerUBaseURL:     merged["mineruBaseURL"],
		MinerUTimeout:     timeoutSeconds,
		MinerUParseMethod: merged["mineruParseMethod"],
		MinerUEffort:      merged["mineruEffort"],
		MinerULanguage:    merged["mineruLanguage"],
		ImageAnalysis:     imageAnalysis,
		TableEnable:       tableEnable,
		FormulaEnable:     formulaEnable,
	}
}

func (s *SettingsService) TestParseSettings(values map[string]string) error {
	settings, err := s.GetSettings()
	if err != nil {
		return err
	}
	for key, value := range values {
		if isParseSettingKey(key) {
			settings[key] = value
		}
	}
	if _, err := s.UpdateSettingsValidationOnly(settings); err != nil {
		return err
	}
	options := s.ParseOptionsFromSettings(settings)
	if options.Engine == "native" {
		return nil
	}
	client := NewMinerUClientWithOptions(options)
	if client == nil || !client.Enabled() {
		return fmt.Errorf("当前解析引擎需要 MinerU，请填写 MinerU 服务地址")
	}
	return client.TestConnection()
}

func isParseSettingKey(key string) bool {
	switch key {
	case "parseEngine", "parsePDFNativeFallback", "mineruBaseURL", "mineruTimeoutSeconds", "mineruParseMethod", "mineruEffort", "mineruLanguage", "mineruImageAnalysis", "mineruTableEnable", "mineruFormulaEnable":
		return true
	default:
		return false
	}
}

func (s *SettingsService) UpdateSettingsValidationOnly(values map[string]string) (map[string]string, error) {
	return s.validateSettings(values, false)
}

func (s *SettingsService) StorageOptionsFromSettings(values map[string]string) StorageOptions {
	defaults := s.Defaults()
	merged := make(map[string]string, len(defaults))
	for key, value := range defaults {
		merged[key] = value
	}
	for key, value := range values {
		merged[key] = value
	}
	useSSL, _ := strconv.ParseBool(merged["rustfsUseSSL"])
	return StorageOptions{
		Endpoint:  merged["rustfsEndpoint"],
		AccessKey: merged["rustfsAccessKey"],
		SecretKey: merged["rustfsSecretKey"],
		Bucket:    merged["rustfsBucket"],
		Region:    merged["rustfsRegion"],
		UseSSL:    useSSL,
	}
}
