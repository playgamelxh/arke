package services

import (
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"gorm.io/gorm"

	"arke/backend/config"
	"arke/backend/models"
)

type KBDocumentService struct {
	db           *gorm.DB
	cfg          config.Config
	storage      StorageInterface
	settings     *SettingsService
	mineruClient *MinerUClient
	milvus       *MilvusBridge
	embedding    *EmbeddingClient
}

func NewKBDocumentService(db *gorm.DB, cfg config.Config, storage StorageInterface, settings *SettingsService, mineru *MinerUClient, milvus *MilvusBridge, embedding *EmbeddingClient) *KBDocumentService {
	return &KBDocumentService{
		db:           db,
		cfg:          cfg,
		storage:      storage,
		settings:     settings,
		mineruClient: mineru,
		milvus:       milvus,
		embedding:    embedding,
	}
}

// UploadDocument 上传文档并关联到知识库
func (s *KBDocumentService) UploadDocument(kbID uint, file *multipart.FileHeader, chunkStrategy string, chunkSize, chunkOverlap int) (*models.Document, error) {
	var kb models.KnowledgeBase
	if err := s.db.First(&kb, kbID).Error; err != nil {
		return nil, fmt.Errorf("知识库不存在")
	}

	// 默认使用知识库配置
	if chunkStrategy == "" {
		chunkStrategy = string(kb.ChunkStrategy)
	}
	if chunkSize <= 0 {
		chunkSize = kb.ChunkSize
	}
	if chunkOverlap <= 0 {
		chunkOverlap = kb.ChunkOverlap
	}

	fileType := strings.ToLower(getFileExt(file.Filename))
	if fileType == "" {
		return nil, fmt.Errorf("无法识别文件类型")
	}

	docID := uuid.New().String()
	objectKey := fmt.Sprintf("knowledge_bases/%d/documents/%s/%s", kbID, docID, file.Filename)

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败：%w", err)
	}
	defer src.Close()

	if err := s.storage.PutObject(objectKey, src, file.Size, file.Header.Get("Content-Type")); err != nil {
		return nil, fmt.Errorf("上传到文档存储失败：%w", err)
	}

	document := &models.Document{
		KnowledgeBaseID: &kbID,
		Name:            docID,
		OriginalName:    file.Filename,
		FileType:        fileType,
		FileSize:        file.Size,
		FilePath:        objectKey,
		Status:          models.StatusUploaded,
		ChunkStrategy:   chunkStrategy,
		ChunkSize:       chunkSize,
		ChunkOverlap:    chunkOverlap,
	}

	if err := s.db.Create(document).Error; err != nil {
		return nil, fmt.Errorf("保存文档记录失败：%w", err)
	}

	// 更新知识库文档计数
	s.db.Model(&kb).UpdateColumn("doc_count", gorm.Expr("doc_count + 1"))

	return document, nil
}

// ListByKB 获取知识库下的文档
func (s *KBDocumentService) ListByKB(kbID uint, page, pageSize int) ([]models.DocumentResponse, int64, error) {
	var docs []models.Document
	var total int64

	query := s.db.Model(&models.Document{}).Where("knowledge_base_id = ?", kbID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := s.db.Where("knowledge_base_id = ?", kbID).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&docs).Error; err != nil {
		return nil, 0, err
	}

	responses := make([]models.DocumentResponse, 0, len(docs))
	for _, doc := range docs {
		var segmentCount int64
		s.db.Model(&models.DocumentSegment{}).Where("document_id = ?", doc.ID).Count(&segmentCount)
		responses = append(responses, s.toResponse(doc, segmentCount))
	}

	return responses, total, nil
}

// List 列出所有文档（全局）
func (s *KBDocumentService) List(page, pageSize int, kbID uint) ([]models.DocumentResponse, int64, error) {
	if kbID > 0 {
		return s.ListByKB(kbID, page, pageSize)
	}

	var docs []models.Document
	var total int64

	query := s.db.Model(&models.Document{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&docs).Error; err != nil {
		return nil, 0, err
	}

	responses := make([]models.DocumentResponse, 0, len(docs))
	for _, doc := range docs {
		var segmentCount int64
		s.db.Model(&models.DocumentSegment{}).Where("document_id = ?", doc.ID).Count(&segmentCount)
		responses = append(responses, s.toResponse(doc, segmentCount))
	}

	return responses, total, nil
}

func (s *KBDocumentService) toResponse(doc models.Document, segmentCount int64) models.DocumentResponse {
	return models.DocumentResponse{
		ID:              doc.ID,
		Name:            doc.Name,
		OriginalName:    doc.OriginalName,
		FileType:        doc.FileType,
		FileSize:        doc.FileSize,
		Status:          doc.Status,
		ParseError:      doc.ParseError,
		SegmentCount:    segmentCount,
		KnowledgeBaseID: doc.KnowledgeBaseID,
		ChunkStrategy:   doc.ChunkStrategy,
		ChunkSize:       doc.ChunkSize,
		ChunkOverlap:    doc.ChunkOverlap,
		CreatedAt:       doc.CreatedAt,
		UpdatedAt:       doc.UpdatedAt,
	}
}

// GetDocument 获取文档
func (s *KBDocumentService) GetDocument(id uint) (*models.Document, error) {
	var doc models.Document
	if err := s.db.First(&doc, id).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// UpdateDocument 编辑文档元信息
func (s *KBDocumentService) UpdateDocument(id uint, req models.DocumentUpdateRequest) (*models.Document, error) {
	var doc models.Document
	if err := s.db.First(&doc, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if req.OriginalName != "" {
		updates["original_name"] = req.OriginalName
	}
	if req.ChunkStrategy != "" {
		strategy := models.ChunkStrategy(req.ChunkStrategy)
		if !strategy.Valid() {
			return nil, fmt.Errorf("不支持的切片策略：%s", req.ChunkStrategy)
		}
		updates["chunk_strategy"] = req.ChunkStrategy
	}
	chunkSize := doc.ChunkSize
	if req.ChunkSize != nil {
		if *req.ChunkSize < 100 || *req.ChunkSize > 5000 {
			return nil, fmt.Errorf("切片大小需在 100-5000 之间")
		}
		chunkSize = *req.ChunkSize
		updates["chunk_size"] = *req.ChunkSize
	}
	if req.ChunkOverlap != nil {
		if *req.ChunkOverlap < 0 {
			return nil, fmt.Errorf("切片重叠不能小于 0")
		}
		if *req.ChunkOverlap >= chunkSize {
			return nil, fmt.Errorf("切片重叠必须小于切片大小")
		}
		if *req.ChunkOverlap > 2000 {
			return nil, fmt.Errorf("切片重叠不能超过 2000")
		}
		updates["chunk_overlap"] = *req.ChunkOverlap
	}

	if len(updates) > 0 {
		if err := s.db.Model(&models.Document{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.GetDocument(id)
}

// UpdateDocumentStatus 更新文档状态
func (s *KBDocumentService) UpdateDocumentStatus(id uint, status models.DocumentStatus, parseError string) error {
	updates := map[string]any{"status": status}
	if parseError != "" {
		updates["parse_error"] = parseError
	}
	return s.db.Model(&models.Document{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteDocument 删除文档（包括向量数据）
func (s *KBDocumentService) DeleteDocument(id uint) error {
	var doc models.Document
	if err := s.db.First(&doc, id).Error; err != nil {
		return err
	}

	// 如果关联了知识库，从 Milvus 中删除向量
	if doc.KnowledgeBaseID != nil {
		var kb models.KnowledgeBase
		if err := s.db.First(&kb, *doc.KnowledgeBaseID).Error; err == nil {
			docIDStr := fmt.Sprintf("%d", id)
			s.milvus.DeleteByFilter(kb.MilvusCollection, fmt.Sprintf("doc_id == \"%s\"", docIDStr))
		}
		s.db.Model(&models.KnowledgeBase{}).Where("id = ?", *doc.KnowledgeBaseID).UpdateColumn("doc_count", gorm.Expr("doc_count - 1"))
	}

	tx := s.db.Begin()
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
		s.storage.RemoveObject(doc.FilePath)
	}

	return tx.Commit().Error
}

// ParseDocument 解析文档（使用知识库的切片策略）
func (s *KBDocumentService) ParseDocument(doc *models.Document) error {
	if doc.KnowledgeBaseID == nil {
		return fmt.Errorf("文档未关联知识库，无法解析")
	}

	var kb models.KnowledgeBase
	if err := s.db.First(&kb, *doc.KnowledgeBaseID).Error; err != nil {
		return fmt.Errorf("知识库不存在")
	}

	if err := s.db.Model(doc).Updates(map[string]any{
		"status":      models.StatusParsing,
		"parse_error": "",
	}).Error; err != nil {
		return err
	}

	tmpPath, cleanup, err := s.storage.DownloadToTemp(doc.FilePath, doc.FileType)
	if err != nil {
		return fmt.Errorf("下载文件失败：%w", err)
	}
	defer cleanup()

	// 解析文件为完整文本
	rawSegments, err := s.parseRaw(tmpPath, doc.OriginalName, doc.FileType)
	if err != nil {
		s.db.Model(doc).Updates(map[string]any{
			"status":      models.StatusFailed,
			"parse_error": err.Error(),
		})
		return err
	}

	if len(rawSegments) == 0 {
		err := fmt.Errorf("解析后未生成任何内容")
		s.db.Model(doc).Updates(map[string]any{
			"status":      models.StatusFailed,
			"parse_error": err.Error(),
		})
		return err
	}

	// 根据切片策略对原始内容进行切片
	strategy := models.ChunkStrategy(doc.ChunkStrategy)
	if strategy == "" {
		strategy = models.ChunkStrategy(kb.ChunkStrategy)
	}
	if !strategy.Valid() {
		strategy = models.ChunkStrategyParagraph
	}
	chunkSize := doc.ChunkSize
	if chunkSize <= 0 {
		chunkSize = kb.ChunkSize
	}
	chunkOverlap := doc.ChunkOverlap
	if chunkOverlap < 0 {
		chunkOverlap = kb.ChunkOverlap
	}
	chunker := NewChunker(strategy, chunkSize, chunkOverlap)

	// 合并所有段的内容，按切片策略重新切分
	var fullContent strings.Builder
	for _, seg := range rawSegments {
		fullContent.WriteString(seg.Content)
		fullContent.WriteString("\n")
	}

	chunks := chunker.Split(fullContent.String())
	if len(chunks) == 0 {
		err := fmt.Errorf("切片后未生成任何内容")
		s.db.Model(doc).Updates(map[string]any{
			"status":      models.StatusFailed,
			"parse_error": err.Error(),
		})
		return err
	}

	// 清理旧 segments
	s.db.Where("document_id = ?", doc.ID).Delete(&models.DocumentSegment{})

	// 保存新 segments
	segments := make([]models.DocumentSegment, 0, len(chunks))
	for _, ch := range chunks {
		segments = append(segments, models.DocumentSegment{
			DocumentID:   doc.ID,
			SegmentType:  "chunk",
			SegmentIndex: ch.Index + 1,
			Title:        ch.Title,
			Content:      ch.Content,
		})
	}

	if err := s.db.Create(&segments).Error; err != nil {
		return fmt.Errorf("保存切片失败：%w", err)
	}

	return s.db.Model(doc).Updates(map[string]any{
		"status":      models.StatusParsed,
		"parse_error": "",
	}).Error
}

// ParseAndIndexDocument 解析文档并自动写入向量库
func (s *KBDocumentService) ParseAndIndexDocument(doc *models.Document) error {
	if err := s.ParseDocument(doc); err != nil {
		return err
	}
	return s.IndexDocument(doc.ID)
}

// parseRaw 解析文件为原始内容
func (s *KBDocumentService) parseRaw(filePath, originalName, fileType string) ([]models.DocumentSegment, error) {
	options := s.parseOptions()
	mineruClient := s.mineruClient
	if options.MinerUBaseURL != "" {
		mineruClient = NewMinerUClientWithOptions(options)
	}
	canUseMinerU := options.Engine != "native" && mineruClient != nil && mineruClient.Enabled() && mineruSupportedType(fileType)

	switch fileType {
	case "pdf":
		if canUseMinerU {
			parsed, err := mineruClient.ParseFile(filePath, originalName)
			if err == nil {
				return parsed, nil
			}
			if options.Engine == "mineru" || !options.PDFNativeFallback {
				return nil, fmt.Errorf("MinerU 解析失败：%w", err)
			}
		}
		return s.parsePDFNative(filePath)
	case "doc", "docx", "ppt", "pptx":
		if canUseMinerU {
			parsed, err := mineruClient.ParseFile(filePath, originalName)
			if err == nil {
				return parsed, nil
			}
			return nil, fmt.Errorf("MinerU 解析失败：%w", err)
		}
		return nil, fmt.Errorf("当前文档解析设置未启用 MinerU，无法解析 %s 文件", fileType)
	case "xlsx", "xls":
		if options.Engine == "mineru" && canUseMinerU {
			parsed, err := mineruClient.ParseFile(filePath, originalName)
			if err == nil {
				return parsed, nil
			}
			return nil, fmt.Errorf("MinerU 解析失败：%w", err)
		}
		return s.parseExcel(filePath)
	case "md", "txt":
		return s.parseText(filePath)
	default:
		if canUseMinerU {
			parsed, err := mineruClient.ParseFile(filePath, originalName)
			if err == nil {
				return parsed, nil
			}
			if options.Engine == "mineru" {
				return nil, fmt.Errorf("MinerU 解析失败：%w", err)
			}
		}
		return s.parseText(filePath)
	}
}

func (s *KBDocumentService) parseOptions() ParseOptions {
	if s.settings == nil {
		return ParseOptions{Engine: "auto", PDFNativeFallback: true}
	}
	settings, err := s.settings.GetSettings()
	if err != nil {
		return ParseOptions{Engine: "auto", PDFNativeFallback: true}
	}
	return s.settings.ParseOptionsFromSettings(settings)
}

func (s *KBDocumentService) parsePDFNative(filePath string) ([]models.DocumentSegment, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开 PDF 失败：%w", err)
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
		return nil, fmt.Errorf("PDF 文件中未提取到文本内容（可能是扫描件，建议使用 MinerU 解析）")
	}
	return []models.DocumentSegment{{
		SegmentType:  "document",
		SegmentIndex: 1,
		Title:        "全文",
		Content:      content,
	}}, nil
}

func (s *KBDocumentService) parseExcel(filePath string) ([]models.DocumentSegment, error) {
	content, err := extractExcelContent(filePath)
	if err != nil {
		return nil, err
	}
	if content == "" {
		return nil, fmt.Errorf("Excel 文件为空")
	}
	return []models.DocumentSegment{{
		SegmentType: "document",
		Title:       "全文",
		Content:     content,
	}}, nil
}

func (s *KBDocumentService) parseText(filePath string) ([]models.DocumentSegment, error) {
	content, err := readFileText(filePath)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("文件内容为空")
	}
	return []models.DocumentSegment{{
		SegmentType: "document",
		Title:       "全文",
		Content:     content,
	}}, nil
}

// IndexDocument 将文档的切片向量化并写入 Milvus
func (s *KBDocumentService) IndexDocument(docID uint) error {
	var doc models.Document
	if err := s.db.First(&doc, docID).Error; err != nil {
		return err
	}
	if doc.KnowledgeBaseID == nil {
		return fmt.Errorf("文档未关联知识库")
	}
	if doc.Status != models.StatusParsed {
		return fmt.Errorf("文档状态不是已解析：%s", doc.Status)
	}

	var kb models.KnowledgeBase
	if err := s.db.First(&kb, *doc.KnowledgeBaseID).Error; err != nil {
		return err
	}

	// 获取所有切片
	var segments []models.DocumentSegment
	if err := s.db.Where("document_id = ?", docID).Order("segment_index ASC").Find(&segments).Error; err != nil {
		return err
	}

	if len(segments) == 0 {
		return fmt.Errorf("文档没有切片")
	}

	docIDStr := fmt.Sprintf("%d", docID)
	if err := s.milvus.LoadCollection(kb.MilvusCollection); err != nil {
		return fmt.Errorf("加载 collection 失败：%w", err)
	}

	if err := s.milvus.DeleteByFilter(kb.MilvusCollection, fmt.Sprintf("doc_id == \"%s\"", docIDStr)); err != nil && !isMilvusCollectionEmptyError(err) {
		return fmt.Errorf("删除旧向量失败：%w", err)
	}

	if err := s.db.Model(&models.DocumentSegment{}).
		Where("document_id = ?", docID).
		Updates(map[string]any{"vector_id": "", "indexed_at": nil}).Error; err != nil {
		return fmt.Errorf("清理旧索引状态失败：%w", err)
	}

	// 对每个切片生成 embedding
	contents := make([]string, len(segments))
	for i, seg := range segments {
		contents[i] = seg.Title + "\n" + seg.Content
	}

	vectors, err := s.embedding.EmbedBatchWith(contents, kb.EmbeddingModel, kb.EmbeddingDim)
	if err != nil {
		return fmt.Errorf("生成 embedding 失败：%w", err)
	}

	// 构造 Milvus 插入数据
	items := make([]InsertItem, len(segments))
	now := time.Now()
	for i, seg := range segments {
		items[i] = InsertItem{
			ID:        fmt.Sprintf("seg_%d_%d", docID, seg.ID),
			KBID:      fmt.Sprintf("%d", *doc.KnowledgeBaseID),
			DocID:     docIDStr,
			SegmentID: fmt.Sprintf("%d", seg.ID),
			Content:   seg.Content,
			Vector:    vectors[i],
		}
	}

	if err := s.milvus.Insert(kb.MilvusCollection, items); err != nil {
		return fmt.Errorf("插入向量失败：%w", err)
	}

	// 更新 segment 的 vector_id 和 indexed_at
	for i, seg := range segments {
		s.db.Model(&models.DocumentSegment{}).Where("id = ?", seg.ID).Updates(map[string]any{
			"vector_id":  items[i].ID,
			"indexed_at": &now,
		})
	}

	// 更新知识库向量计数
	var totalVectors int64
	s.db.Model(&models.DocumentSegment{}).
		Joins("JOIN documents ON documents.id = document_segments.document_id").
		Where("documents.knowledge_base_id = ? AND document_segments.vector_id IS NOT NULL AND document_segments.vector_id != ''", *doc.KnowledgeBaseID).
		Count(&totalVectors)

	s.db.Model(&models.KnowledgeBase{}).
		Where("id = ?", *doc.KnowledgeBaseID).
		UpdateColumn("vector_count", totalVectors)

	return nil
}

func isMilvusCollectionEmptyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "collection not loaded") || strings.Contains(msg, "collection not found") || strings.Contains(msg, "index not found") || strings.Contains(msg, "not found")
}

// UpdateSegment 编辑切片并同步向量
func (s *KBDocumentService) UpdateSegment(segmentID uint, title, content string) error {
	var segment models.DocumentSegment
	if err := s.db.First(&segment, segmentID).Error; err != nil {
		return err
	}

	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("切片内容不能为空")
	}
	if strings.TrimSpace(title) == "" {
		title = segment.Title
	}

	var doc models.Document
	if err := s.db.First(&doc, segment.DocumentID).Error; err != nil {
		return err
	}
	if doc.KnowledgeBaseID == nil {
		return s.updateSegmentNoKB(segmentID, title, content)
	}

	updates := map[string]any{
		"title":   title,
		"content": content,
	}

	if segment.VectorID != "" {
		var kb models.KnowledgeBase
		if err := s.db.First(&kb, *doc.KnowledgeBaseID).Error; err != nil {
			return err
		}

		fullText := title + "\n" + content
		vector, err := s.embedding.EmbedWith(fullText, kb.EmbeddingModel, kb.EmbeddingDim)
		if err != nil {
			return fmt.Errorf("生成 embedding 失败：%w", err)
		}

		if err := s.milvus.LoadCollection(kb.MilvusCollection); err != nil {
			return err
		}

		if err := s.milvus.DeleteByFilter(kb.MilvusCollection, fmt.Sprintf("id == \"%s\"", segment.VectorID)); err != nil {
			return fmt.Errorf("删除旧向量失败：%w", err)
		}

		if err := s.milvus.Insert(kb.MilvusCollection, []InsertItem{{
			ID:        segment.VectorID,
			KBID:      fmt.Sprintf("%d", *doc.KnowledgeBaseID),
			DocID:     fmt.Sprintf("%d", segment.DocumentID),
			SegmentID: fmt.Sprintf("%d", segmentID),
			Content:   content,
			Vector:    vector,
		}}); err != nil {
			return fmt.Errorf("更新 Milvus 向量失败：%w", err)
		}

		now := time.Now()
		updates["indexed_at"] = &now
	}

	return s.db.Model(&models.DocumentSegment{}).Where("id = ?", segmentID).Updates(updates).Error
}

func (s *KBDocumentService) updateSegmentNoKB(segmentID uint, title, content string) error {
	return s.db.Model(&models.DocumentSegment{}).Where("id = ?", segmentID).Updates(map[string]any{
		"title":   title,
		"content": content,
	}).Error
}

// DeleteSegment 删除切片
func (s *KBDocumentService) DeleteSegment(segmentID uint) error {
	var segment models.DocumentSegment
	if err := s.db.First(&segment, segmentID).Error; err != nil {
		return err
	}

	if segment.VectorID != "" {
		var doc models.Document
		if err := s.db.First(&doc, segment.DocumentID).Error; err == nil && doc.KnowledgeBaseID != nil {
			var kb models.KnowledgeBase
			if err := s.db.First(&kb, *doc.KnowledgeBaseID).Error; err == nil {
				s.milvus.DeleteByFilter(kb.MilvusCollection, fmt.Sprintf("id == \"%s\"", segment.VectorID))
			}
		}
	}

	return s.db.Delete(&models.DocumentSegment{}, segmentID).Error
}

// GetSegments 获取文档的所有切片
func (s *KBDocumentService) GetSegments(docID uint) ([]models.DocumentSegment, error) {
	var segments []models.DocumentSegment
	if err := s.db.Where("document_id = ?", docID).Order("segment_index ASC").Find(&segments).Error; err != nil {
		return nil, err
	}
	return segments, nil
}

func getFileExt(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return ""
	}
	return filename[idx+1:]
}
