package services

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"gorm.io/gorm"

	"arke/backend/config"
	"arke/backend/models"
)

type DocumentService struct {
	db            *gorm.DB
	cfg           config.Config
	storage       StorageInterface
	mineruClient  *MinerUClient
	bailianClient *BailianClient
}

func NewDocumentService(db *gorm.DB, cfg config.Config, storage StorageInterface, mineru *MinerUClient, bailian *BailianClient) *DocumentService {
	return &DocumentService{
		db:            db,
		cfg:           cfg,
		storage:       storage,
		mineruClient:  mineru,
		bailianClient: bailian,
	}
}

func (s *DocumentService) UploadDocument(file *multipart.FileHeader) (*models.Document, error) {
	fileType := strings.ToLower(filepath.Ext(file.Filename))[1:]
	if fileType == "" {
		return nil, fmt.Errorf("无法识别文件类型")
	}

	docID := uuid.New().String()
	objectKey := fmt.Sprintf("documents/%s/%s", docID, file.Filename)

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败：%w", err)
	}
	defer src.Close()

	if err := s.storage.PutObject(objectKey, src, file.Size, file.Header.Get("Content-Type")); err != nil {
		return nil, fmt.Errorf("上传到存储失败：%w", err)
	}

	document := &models.Document{
		Name:         docID,
		OriginalName: file.Filename,
		FileType:     fileType,
		FileSize:     file.Size,
		FilePath:     objectKey,
		Status:       models.StatusUploaded,
	}

	if err := s.db.Create(document).Error; err != nil {
		return nil, fmt.Errorf("保存文档记录失败：%w", err)
	}

	return document, nil
}

func (s *DocumentService) ParseDocument(doc *models.Document) error {
	if err := s.db.Model(doc).Update("status", models.StatusParsing).Error; err != nil {
		return err
	}

	tmpPath, cleanup, err := s.storage.DownloadToTemp(doc.FilePath, doc.FileType)
	if err != nil {
		return fmt.Errorf("下载文件失败：%w", err)
	}
	defer cleanup()

	var segments []models.DocumentSegment
	parseErr := s.parseFile(tmpPath, doc.OriginalName, doc.FileType, &segments)

	if parseErr != nil {
		s.db.Model(doc).Updates(map[string]any{
			"status":      models.StatusFailed,
			"parse_error": parseErr.Error(),
		})
		return parseErr
	}

	if len(segments) == 0 {
		err := fmt.Errorf("解析后未生成任何内容")
		s.db.Model(doc).Updates(map[string]any{
			"status":      models.StatusFailed,
			"parse_error": err.Error(),
		})
		return err
	}

	for i := range segments {
		segments[i].DocumentID = doc.ID
		segments[i].SegmentIndex = i + 1
	}

	if err := s.db.Create(&segments).Error; err != nil {
		return fmt.Errorf("保存解析内容失败：%w", err)
	}

	return s.db.Model(doc).Updates(map[string]any{
		"status":      models.StatusParsed,
		"parse_error": "",
	}).Error
}

func (s *DocumentService) parseFile(filePath, originalName, fileType string, segments *[]models.DocumentSegment) error {
	canUseMinerU := s.mineruClient != nil && s.mineruClient.Enabled() && mineruSupportedType(fileType)

	switch fileType {
	case "pdf":
		if canUseMinerU {
			return s.parsePDFWithMinerU(filePath, originalName, segments)
		}
		return s.parsePDFWithNative(filePath, segments)
	case "doc", "docx", "ppt", "pptx":
		if canUseMinerU {
			return s.parseOfficeWithMinerU(filePath, originalName, segments)
		}
		return fmt.Errorf("当前环境未配置 MinerU，无法解析 %s 文件", fileType)
	case "xlsx", "xls":
		return s.parseExcel(filePath, segments)
	case "md", "txt":
		return s.parseText(filePath, segments)
	default:
		if canUseMinerU {
			parsed, err := s.mineruClient.ParseFile(filePath, originalName)
			if err != nil {
				return fmt.Errorf("MinerU 解析失败：%w", err)
			}
			*segments = parsed
			return nil
		}
		return s.parseText(filePath, segments)
	}
}

func (s *DocumentService) parseOfficeWithMinerU(filePath, originalName string, segments *[]models.DocumentSegment) error {
	parsed, err := s.mineruClient.ParseFile(filePath, originalName)
	if err != nil {
		return fmt.Errorf("MinerU 解析失败：%w", err)
	}
	*segments = parsed
	return nil
}

func (s *DocumentService) parsePDFWithMinerU(filePath, originalName string, segments *[]models.DocumentSegment) error {
	if !mineruSupportedType("pdf") {
		return s.parsePDFWithNative(filePath, segments)
	}
	parsed, err := s.mineruClient.ParseFile(filePath, originalName)
	if err != nil {
		return fmt.Errorf("MinerU 解析失败，回退到原生解析：%w", err)
	}
	*segments = parsed
	return nil
}

func (s *DocumentService) parsePDFWithNative(filePath string, segments *[]models.DocumentSegment) error {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开 PDF 失败：%w", err)
	}
	defer f.Close()

	var buf strings.Builder
	total := r.NumPage()
	for i := 1; i <= total; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n")
	}

	content := strings.TrimSpace(buf.String())
	if content == "" {
		return fmt.Errorf("PDF 文件中未提取到文本内容（可能是扫描件，建议使用 MinerU 解析）")
	}

	*segments = []models.DocumentSegment{{
		SegmentType:  "document",
		SegmentIndex: 1,
		Title:        "全文",
		Content:      content,
	}}
	return nil
}

func (s *DocumentService) parseExcel(filePath string, segments *[]models.DocumentSegment) error {
	content, err := extractExcelContent(filePath)
	if err != nil {
		return err
	}
	if content == "" {
		return fmt.Errorf("Excel 文件为空")
	}
	*segments = []models.DocumentSegment{{
		SegmentType: "document",
		Title:       "全文",
		Content:     content,
	}}
	return nil
}

func (s *DocumentService) parseText(filePath string, segments *[]models.DocumentSegment) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(string(content))
	if text == "" {
		return fmt.Errorf("文件内容为空")
	}
	*segments = []models.DocumentSegment{{
		SegmentType: "document",
		Title:       "全文",
		Content:     text,
	}}
	return nil
}

func extractExcelContent(filePath string) (string, error) {
	return "", fmt.Errorf("Excel 解析暂未实现")
}

func (s *DocumentService) GetDocuments(page, pageSize int) ([]models.DocumentResponse, int64, error) {
	var docs []models.Document
	var total int64

	if err := s.db.Model(&models.Document{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&docs).Error; err != nil {
		return nil, 0, err
	}

	responses := make([]models.DocumentResponse, 0, len(docs))
	for _, doc := range docs {
		var segmentCount, qaCount int64
		s.db.Model(&models.DocumentSegment{}).Where("document_id = ?", doc.ID).Count(&segmentCount)
		s.db.Model(&models.QAItem{}).Where("document_id = ?", doc.ID).Count(&qaCount)

		responses = append(responses, models.DocumentResponse{
			ID:           doc.ID,
			Name:         doc.Name,
			OriginalName: doc.OriginalName,
			FileType:     doc.FileType,
			FileSize:     doc.FileSize,
			Status:       doc.Status,
			ParseError:   doc.ParseError,
			SegmentCount: segmentCount,
			QACount:      qaCount,
			CreatedAt:    doc.CreatedAt,
			UpdatedAt:    doc.UpdatedAt,
		})
	}

	return responses, total, nil
}

func (s *DocumentService) GetDocumentByID(id uint) (*models.Document, error) {
	var doc models.Document
	if err := s.db.First(&doc, id).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *DocumentService) DeleteDocument(id uint) error {
	tx := s.db.Begin()

	var doc models.Document
	if err := tx.First(&doc, id).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&models.DocumentSegment{}, "document_id = ?", id).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&models.QAItem{}, "document_id = ?", id).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&doc).Error; err != nil {
		tx.Rollback()
		return err
	}

	if doc.FilePath != "" {
		if err := s.storage.RemoveObject(doc.FilePath); err != nil {
			tx.Rollback()
			return fmt.Errorf("删除存储文件失败：%w", err)
		}
	}

	return tx.Commit().Error
}

func (s *DocumentService) GetDocumentSegments(docID uint) ([]models.DocumentSegment, error) {
	var segments []models.DocumentSegment
	if err := s.db.Where("document_id = ?", docID).Order("segment_index ASC").Find(&segments).Error; err != nil {
		return nil, err
	}
	return segments, nil
}

func (s *DocumentService) GetSegmentByID(id uint) (*models.DocumentSegment, error) {
	var segment models.DocumentSegment
	if err := s.db.First(&segment, id).Error; err != nil {
		return nil, err
	}
	return &segment, nil
}

func (s *DocumentService) UpdateSegment(id uint, title, content string) error {
	return s.db.Model(&models.DocumentSegment{}).Where("id = ?", id).Updates(map[string]any{
		"title":   title,
		"content": content,
	}).Error
}

func (s *DocumentService) DeleteSegment(id uint) error {
	return s.db.Delete(&models.DocumentSegment{}, id).Error
}

func (s *DocumentService) GetStats() (*models.StatsResponse, error) {
	var total, parsed, failed int64
	s.db.Model(&models.Document{}).Count(&total)
	s.db.Model(&models.Document{}).Where("status = ?", models.StatusParsed).Count(&parsed)
	s.db.Model(&models.Document{}).Where("status = ?", models.StatusFailed).Count(&failed)

	var qaCount int64
	s.db.Model(&models.QAItem{}).Count(&qaCount)

	var recentDocs []models.Document
	if err := s.db.Order("created_at DESC").Limit(5).Find(&recentDocs).Error; err != nil {
		return nil, err
	}

	recentResponses := make([]models.DocumentResponse, 0, len(recentDocs))
	for _, doc := range recentDocs {
		var segmentCount, docQACount int64
		s.db.Model(&models.DocumentSegment{}).Where("document_id = ?", doc.ID).Count(&segmentCount)
		s.db.Model(&models.QAItem{}).Where("document_id = ?", doc.ID).Count(&docQACount)

		recentResponses = append(recentResponses, models.DocumentResponse{
			ID:           doc.ID,
			Name:         doc.Name,
			OriginalName: doc.OriginalName,
			FileType:     doc.FileType,
			FileSize:     doc.FileSize,
			Status:       doc.Status,
			ParseError:   doc.ParseError,
			SegmentCount: segmentCount,
			QACount:      docQACount,
			CreatedAt:    doc.CreatedAt,
			UpdatedAt:    doc.UpdatedAt,
		})
	}

	return &models.StatsResponse{
		Documents:       total,
		Parsed:          parsed,
		Failed:          failed,
		QA:              qaCount,
		RecentDocuments: recentResponses,
	}, nil
}
