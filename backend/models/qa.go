package models

import "time"

type QAItem struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	DocumentID      uint      `json:"documentId"`
	Document        Document  `json:"-"`
	SourceSegmentID *uint     `json:"sourceSegmentId"`
	Question        string    `json:"question"`
	Answer          string    `json:"answer" gorm:"type:longtext"`
	Tags            string    `json:"-" gorm:"type:json"`
	Enabled         bool      `json:"enabled"`
	Confidence      float64   `json:"confidence"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type QAGenerateTask struct {
	ID              uint                 `json:"id" gorm:"primaryKey"`
	KnowledgeBaseID uint                 `json:"knowledgeBaseId" gorm:"index"`
	DocumentID      uint                 `json:"documentId"`
	Status          QAGenerateTaskStatus `json:"status"`
	Progress        int                  `json:"progress"`
	Message         string               `json:"message"`
	TaskCount       int                  `json:"-" gorm:"column:task_count"`
	Difficulty      string               `json:"-"`
	Instruction     string               `json:"-" gorm:"type:text"`
	ResultJSON      string               `json:"-" gorm:"type:longtext"`
	ErrorMessage    string               `json:"-" gorm:"type:text"`
	CreatedAt       time.Time            `json:"createdAt"`
	UpdatedAt       time.Time            `json:"updatedAt"`
}
