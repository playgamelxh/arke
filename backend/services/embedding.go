package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"arke/backend/config"
)

type EmbeddingClient struct {
	apiKey     string
	baseURL    string
	model      string
	dim        int
	batchSize  int
	client     *http.Client
}

var supportedEmbeddingModels = map[string]bool{
	"text-embedding-v3": true,
	"text-embedding-v4": true,
}

// SupportedEmbeddingModels 返回当前支持的 embedding 模型列表
func SupportedEmbeddingModels() []string {
	return []string{"text-embedding-v3", "text-embedding-v4"}
}

// IsSupportedEmbeddingModel 校验模型是否在白名单内
func IsSupportedEmbeddingModel(model string) bool {
	return supportedEmbeddingModels[strings.TrimSpace(model)]
}

var supportedEmbeddingDims = map[string][]int{
	"text-embedding-v3": {1024, 768, 512, 256, 128, 64},
	"text-embedding-v4": {2048, 1536, 1024, 768, 512, 256, 128, 64},
}

// SupportedEmbeddingDims 返回指定模型支持的维度列表
func SupportedEmbeddingDims(model string) []int {
	return supportedEmbeddingDims[strings.TrimSpace(model)]
}

// IsSupportedEmbeddingDim 校验维度是否被指定模型支持
func IsSupportedEmbeddingDim(model string, dim int) bool {
	for _, d := range supportedEmbeddingDims[strings.TrimSpace(model)] {
		if d == dim {
			return true
		}
	}
	return false
}

// DefaultModel 返回客户端的默认 embedding 模型
func (e *EmbeddingClient) DefaultModel() string {
	return e.model
}

func NewEmbeddingClient(cfg config.Config) *EmbeddingClient {
	dim := cfg.EmbeddingDim
	if dim == 0 {
		dim = 1024
	}
	model := cfg.EmbeddingModel
	if model == "" {
		model = "text-embedding-v3"
	}

	timeout := cfg.DashScopeTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	return &EmbeddingClient{
		apiKey:    cfg.DashScopeAPIKey,
		baseURL:   "https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding",
		model:     model,
		dim:       dim,
		batchSize: 10,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

type embeddingRequest struct {
	Model      string           `json:"model"`
	Input      embeddingInput   `json:"input"`
	Parameters embeddingParams  `json:"parameters"`
}

type embeddingInput struct {
	Texts []string `json:"texts"`
}

type embeddingParams struct {
	Dimension int `json:"dimension,omitempty"`
}

type embeddingResponse struct {
	Output embeddingOutput `json:"output"`
}

type embeddingOutput struct {
	Embeddings []embeddingItem `json:"embeddings"`
}

type embeddingItem struct {
	Embedding []float32 `json:"embedding"`
	TextIndex int       `json:"text_index"`
}

func (e *EmbeddingClient) Embed(text string) ([]float32, error) {
	return e.EmbedWith(text, e.model, e.dim)
}

func (e *EmbeddingClient) EmbedWith(text, model string, dim int) ([]float32, error) {
	results, err := e.EmbedBatchWith([]string{text}, model, dim)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("embedding 返回为空")
	}
	return results[0], nil
}

func (e *EmbeddingClient) EmbedBatch(texts []string) ([][]float32, error) {
	return e.EmbedBatchWith(texts, e.model, e.dim)
}

func (e *EmbeddingClient) EmbedBatchWith(texts []string, model string, dim int) ([][]float32, error) {
	if e.apiKey == "" {
		return nil, fmt.Errorf("未配置 DASHSCOPE_API_KEY，无法调用 embedding")
	}
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	if strings.TrimSpace(model) == "" {
		model = e.model
	}
	if dim <= 0 {
		dim = e.dim
	}

	// 清理文本
	cleaned := make([]string, 0, len(texts))
	for _, t := range texts {
		t = strings.TrimSpace(t)
		if t == "" {
			cleaned = append(cleaned, " ")
		} else {
			cleaned = append(cleaned, t)
		}
	}

	allResults := make([][]float32, len(texts))
	for start := 0; start < len(cleaned); start += e.batchSize {
		end := start + e.batchSize
		if end > len(cleaned) {
			end = len(cleaned)
		}
		batch := cleaned[start:end]

		reqBody := embeddingRequest{
			Model: model,
			Input: embeddingInput{Texts: batch},
			Parameters: embeddingParams{
				Dimension: dim,
			},
		}
		body, _ := json.Marshal(reqBody)
		req, err := http.NewRequest(http.MethodPost, e.baseURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := e.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("调用 embedding 失败：%w", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("embedding API 返回 %d：%s", resp.StatusCode, string(respBody))
		}

		var embeddingResp embeddingResponse
		if err := json.Unmarshal(respBody, &embeddingResp); err != nil {
			return nil, fmt.Errorf("解析 embedding 响应失败：%w", err)
		}

		for _, item := range embeddingResp.Output.Embeddings {
			idx := start + item.TextIndex
			if idx < len(allResults) {
				allResults[idx] = item.Embedding
			}
		}
	}

	// 检查是否有空结果
	for i, r := range allResults {
		if len(r) == 0 {
			return nil, fmt.Errorf("第 %d 个文本 embedding 失败", i)
		}
	}

	return allResults, nil
}

func (e *EmbeddingClient) Dimension() int {
	return e.dim
}
