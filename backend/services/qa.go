package services

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"arke/backend/models"
)

type QAService struct {
	db            *gorm.DB
	bailianClient *BailianClient
}

func NewQAService(db *gorm.DB, bailian *BailianClient) *QAService {
	return &QAService{
		db:            db,
		bailianClient: bailian,
	}
}

func (s *QAService) GenerateQA(kbID uint, count int, difficulty, instruction string, overwrite bool) (*models.QAGenerateTask, error) {
	var kb models.KnowledgeBase
	if err := s.db.First(&kb, kbID).Error; err != nil {
		return nil, err
	}

	var segments []models.DocumentSegment
	if err := s.db.Joins("JOIN documents ON documents.id = document_segments.document_id").
		Where("documents.knowledge_base_id = ? AND documents.status = ?", kbID, models.StatusParsed).
		Order("documents.created_at DESC, document_segments.segment_index ASC").
		Find(&segments).Error; err != nil {
		return nil, err
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("知识库没有可用于生成问答的解析切片")
	}

	primaryDocID := segments[0].DocumentID
	task := &models.QAGenerateTask{
		KnowledgeBaseID: kbID,
		DocumentID:      primaryDocID,
		Status:          models.QATaskRunning,
		Progress:        0,
		TaskCount:       count,
		Difficulty:      difficulty,
		Instruction:     instruction,
	}

	if err := s.db.Create(task).Error; err != nil {
		return nil, err
	}

	var existing []models.GeneratedQAItem
	var existingQA []models.QAItem
	if err := s.db.Joins("JOIN documents ON documents.id = qa_items.document_id").
		Where("documents.knowledge_base_id = ?", kbID).
		Find(&existingQA).Error; err == nil {
		for _, qa := range existingQA {
			existing = append(existing, models.GeneratedQAItem{Question: qa.Question})
		}
	}

	generated, err := s.bailianClient.GenerateQA(segments, count, difficulty, instruction, existing)
	if err != nil {
		s.db.Model(task).Updates(map[string]any{
			"status":        models.QATaskFailed,
			"progress":      0,
			"error_message": err.Error(),
		})
		return task, err
	}

	if len(generated) == 0 {
		s.db.Model(task).Updates(map[string]any{
			"status":        models.QATaskFailed,
			"progress":      0,
			"error_message": "未生成任何问答",
		})
		return task, fmt.Errorf("未生成任何问答")
	}

	resultJSON, _ := json.Marshal(generated)
	task.Status = models.QATaskCompleted
	task.Progress = 100
	task.ResultJSON = string(resultJSON)

	return task, s.db.Save(task).Error
}

func (s *QAService) GetQATask(taskID uint) (*models.QAGenerateTaskResponse, error) {
	var task models.QAGenerateTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}

	var generatedCount int64
	s.db.Model(&models.QAItem{}).Where("document_id = ?", task.DocumentID).Count(&generatedCount)

	var generatedItems []models.GeneratedQAItem
	if task.ResultJSON != "" {
		json.Unmarshal([]byte(task.ResultJSON), &generatedItems)
	}

	return &models.QAGenerateTaskResponse{
		ID:              task.ID,
		KnowledgeBaseID: task.KnowledgeBaseID,
		DocumentID:      task.DocumentID,
		Status:          task.Status,
		Progress:        task.Progress,
		Message:         task.Message,
		TargetCount:     task.TaskCount,
		GeneratedCount:  len(generatedItems),
		Items:           generatedItems,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
	}, nil
}

func (s *QAService) GetQAItems(docID uint, page, pageSize int) ([]models.QAResponse, int64, error) {
	var items []models.QAItem
	var total int64

	query := s.db.Model(&models.QAItem{})
	if docID > 0 {
		query = query.Where("document_id = ?", docID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	responses := make([]models.QAResponse, 0, len(items))
	for _, item := range items {
		var tags []string
		if item.Tags != "" {
			json.Unmarshal([]byte(item.Tags), &tags)
		}

		var docName string
		s.db.Model(&models.Document{}).Select("original_name").Where("id = ?", item.DocumentID).Scan(&docName)

		responses = append(responses, models.QAResponse{
			ID:              item.ID,
			DocumentID:      item.DocumentID,
			DocumentName:    docName,
			SourceSegmentID: item.SourceSegmentID,
			Question:        item.Question,
			Answer:          item.Answer,
			Tags:            tags,
			Enabled:         item.Enabled,
			Confidence:      item.Confidence,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}

	return responses, total, nil
}

func (s *QAService) GetQAItem(id uint) (*models.QAResponse, error) {
	var item models.QAItem
	if err := s.db.First(&item, id).Error; err != nil {
		return nil, err
	}

	var tags []string
	if item.Tags != "" {
		json.Unmarshal([]byte(item.Tags), &tags)
	}

	var docName string
	s.db.Model(&models.Document{}).Select("original_name").Where("id = ?", item.DocumentID).Scan(&docName)

	return &models.QAResponse{
		ID:              item.ID,
		DocumentID:      item.DocumentID,
		DocumentName:    docName,
		SourceSegmentID: item.SourceSegmentID,
		Question:        item.Question,
		Answer:          item.Answer,
		Tags:            tags,
		Enabled:         item.Enabled,
		Confidence:      item.Confidence,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}, nil
}

func (s *QAService) UpsertQA(item models.QAItem) (*models.QAResponse, error) {
	tagsJSON, _ := json.Marshal(item.Tags)
	item.Tags = string(tagsJSON)

	if item.ID > 0 {
		if err := s.db.Model(&models.QAItem{}).Where("id = ?", item.ID).Updates(item).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.Create(&item).Error; err != nil {
			return nil, err
		}
	}

	return s.GetQAItem(item.ID)
}

func (s *QAService) UpdateQAStatus(id uint, enabled bool) error {
	return s.db.Model(&models.QAItem{}).Where("id = ?", id).Update("enabled", enabled).Error
}

func (s *QAService) DeleteQA(id uint) error {
	return s.db.Delete(&models.QAItem{}, id).Error
}

func (s *QAService) BatchDeleteQA(ids []uint) error {
	return s.db.Delete(&models.QAItem{}, ids).Error
}

func (s *QAService) SaveGeneratedQA(kbID uint, items []models.GeneratedQAItem, overwrite bool) error {
	var docIDs []uint
	if err := s.db.Model(&models.Document{}).Where("knowledge_base_id = ?", kbID).Pluck("id", &docIDs).Error; err != nil {
		return err
	}
	if len(docIDs) == 0 {
		return fmt.Errorf("知识库没有可保存问答的文档")
	}
	if overwrite {
		if err := s.db.Delete(&models.QAItem{}, "document_id IN ?", docIDs).Error; err != nil {
			return err
		}
	}

	docIDSet := map[uint]bool{}
	for _, id := range docIDs {
		docIDSet[id] = true
	}

	qaItems := make([]models.QAItem, 0, len(items))
	for _, item := range items {
		docID := item.DocumentID
		if docID == 0 && item.SourceSegmentID != nil {
			var segment models.DocumentSegment
			if err := s.db.First(&segment, *item.SourceSegmentID).Error; err == nil {
				docID = segment.DocumentID
			}
		}
		if docID == 0 {
			continue
		}
		if !docIDSet[docID] {
			continue
		}
		tagsJSON, _ := json.Marshal(item.Keywords)
		qaItems = append(qaItems, models.QAItem{
			DocumentID:      docID,
			SourceSegmentID: item.SourceSegmentID,
			Question:        item.Question,
			Answer:          item.Answer,
			Tags:            string(tagsJSON),
			Enabled:         true,
			Confidence:      item.Confidence,
		})
	}

	if len(qaItems) == 0 {
		return fmt.Errorf("没有可保存的问答，来源文档不属于当前知识库")
	}

	return s.db.Create(&qaItems).Error
}

func (s *QAService) GenerateAnswer(docID uint, question string) (models.GenerateAnswerResponse, error) {
	var segments []models.DocumentSegment
	if err := s.db.Where("document_id = ?", docID).Order("segment_index ASC").Find(&segments).Error; err != nil {
		return models.GenerateAnswerResponse{}, err
	}

	if len(segments) == 0 {
		return models.GenerateAnswerResponse{}, fmt.Errorf("文档没有解析内容")
	}

	return s.bailianClient.GenerateAnswer(segments, question)
}
