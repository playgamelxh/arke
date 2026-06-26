package services

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"arke/backend/models"
)

type KnowledgeAskService struct {
	kbService *KnowledgeBaseService
	bailian   *BailianClient
}

func NewKnowledgeAskService(kbService *KnowledgeBaseService, bailian *BailianClient) *KnowledgeAskService {
	return &KnowledgeAskService{kbService: kbService, bailian: bailian}
}

func (s *KnowledgeAskService) Ask(req models.KnowledgeAskRequest) (models.KnowledgeAskResponse, error) {
	question := strings.TrimSpace(req.Question)
	if question == "" {
		return models.KnowledgeAskResponse{}, fmt.Errorf("问题不能为空")
	}

	recallCount := req.RecallCount
	if recallCount <= 0 {
		recallCount = 10
	}
	if recallCount > 50 {
		recallCount = 50
	}

	useCount := req.UseCount
	if useCount <= 0 {
		useCount = 5
	}
	if useCount > recallCount {
		useCount = recallCount
	}

	rerankMode := strings.TrimSpace(req.RerankMode)
	if rerankMode == "" {
		rerankMode = "similarity"
	}

	hits, err := s.kbService.Search(req.KnowledgeBaseID, question, recallCount)
	if err != nil {
		return models.KnowledgeAskResponse{}, err
	}
	if len(hits) == 0 {
		return models.KnowledgeAskResponse{}, fmt.Errorf("未召回到知识库内容")
	}

	sources := buildKnowledgeAskSources(hits)
	rerankSources(sources, rerankMode, question)
	if useCount > len(sources) {
		useCount = len(sources)
	}
	usedSources := sources[:useCount]

	answer, err := s.bailian.GenerateKnowledgeAnswer(usedSources, question)
	if err != nil {
		return models.KnowledgeAskResponse{}, err
	}

	return models.KnowledgeAskResponse{
		Answer:        answer.Answer,
		Confidence:    answer.Confidence,
		Sources:       usedSources,
		RecallCount:   len(sources),
		UseCount:      len(usedSources),
		RerankMode:    rerankMode,
		SourceExcerpt: answer.SourceExcerpt,
	}, nil
}

func buildKnowledgeAskSources(hits []SearchHit) []models.KnowledgeAskSource {
	sources := make([]models.KnowledgeAskSource, 0, len(hits))
	for _, hit := range hits {
		docID, _ := strconv.ParseUint(hit.DocID, 10, 64)
		segmentIDValue, _ := strconv.ParseUint(hit.SegmentID, 10, 64)
		var segmentID *uint
		if segmentIDValue > 0 {
			id := uint(segmentIDValue)
			segmentID = &id
		}
		score := 1 - hit.Distance
		if score < 0 {
			score = 0
		}
		sources = append(sources, models.KnowledgeAskSource{
			DocumentID:       uint(docID),
			SourceSegmentID:  segmentID,
			Content:          hit.Content,
			Score:            score,
			OriginalDistance: hit.Distance,
		})
	}
	return sources
}

func rerankSources(sources []models.KnowledgeAskSource, mode string, question string) {
	switch mode {
	case "length":
		sort.SliceStable(sources, func(i, j int) bool {
			return len([]rune(sources[i].Content)) > len([]rune(sources[j].Content))
		})
	case "keyword":
		keywords := tokenizeQuestion(question)
		sort.SliceStable(sources, func(i, j int) bool {
			left := keywordMatchScore(sources[i].Content, keywords)
			right := keywordMatchScore(sources[j].Content, keywords)
			if left == right {
				return sources[i].Score > sources[j].Score
			}
			return left > right
		})
	default:
		sort.SliceStable(sources, func(i, j int) bool {
			return sources[i].Score > sources[j].Score
		})
	}
}

func tokenizeQuestion(question string) []string {
	parts := strings.FieldsFunc(question, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == '，' || r == ',' || r == '。' || r == '？' || r == '?' || r == '；' || r == ';' || r == '：' || r == ':'
	})
	keywords := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len([]rune(part)) >= 2 {
			keywords = append(keywords, part)
		}
	}
	return keywords
}

func keywordMatchScore(content string, keywords []string) int {
	if len(keywords) == 0 {
		return 0
	}
	score := 0
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			score++
		}
	}
	return score
}
