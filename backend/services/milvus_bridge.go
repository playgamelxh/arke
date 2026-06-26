package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MilvusBridge struct {
	baseURL string
	client  *http.Client
}

func NewMilvusBridge(baseURL string) *MilvusBridge {
	if baseURL == "" {
		baseURL = "http://milvus-bridge:8088"
	}
	return &MilvusBridge{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type CreateCollectionRequest struct {
	CollectionName string         `json:"collection_name"`
	Dimension      int            `json:"dimension"`
	IndexType      string         `json:"index_type"`
	IndexParams    map[string]any `json:"index_params"`
	MetricType     string         `json:"metric_type"`
}

type InsertItem struct {
	ID         string    `json:"id"`
	KBID       string    `json:"kb_id"`
	DocID      string    `json:"doc_id"`
	SegmentID  string    `json:"segment_id"`
	Content    string    `json:"content"`
	Vector     []float32 `json:"vector"`
}

type InsertRequest struct {
	Items []InsertItem `json:"items"`
}

type SearchRequest struct {
	Vectors      [][]float32 `json:"vectors"`
	TopK         int         `json:"top_k"`
	Filter       string      `json:"filter,omitempty"`
	OutputFields []string    `json:"output_fields"`
}

type SearchHit struct {
	ID       string  `json:"id"`
	Distance float64 `json:"distance"`
	DocID    string  `json:"doc_id"`
	SegmentID string `json:"segment_id"`
	Content  string  `json:"content"`
	KBID     string  `json:"kb_id"`
}

type SearchResponse struct {
	Results [][]SearchHit `json:"results"`
}

type CountResponse struct {
	CollectionName string `json:"collection_name"`
	NumEntities    int64  `json:"num_entities"`
}

func (m *MilvusBridge) doRequest(method, path string, body any, result any) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, m.baseURL+path, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("milvus-bridge 请求失败：%w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("milvus-bridge 返回 %d：%s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("解析响应失败：%w", err)
		}
	}
	return nil
}

func (m *MilvusBridge) Health() error {
	return m.doRequest("GET", "/health", nil, nil)
}

func (m *MilvusBridge) CreateCollection(req CreateCollectionRequest) error {
	return m.doRequest("POST", "/collections", req, nil)
}

func (m *MilvusBridge) DropCollection(name string) error {
	return m.doRequest("DELETE", "/collections/"+name, nil, nil)
}

func (m *MilvusBridge) CollectionExists(name string) (bool, error) {
	var result struct {
		Exists bool `json:"exists"`
	}
	if err := m.doRequest("GET", "/collections/"+name+"/exists", nil, &result); err != nil {
		return false, err
	}
	return result.Exists, nil
}

func (m *MilvusBridge) LoadCollection(name string) error {
	return m.doRequest("POST", "/collections/"+name+"/load", nil, nil)
}

func (m *MilvusBridge) ReleaseCollection(name string) error {
	return m.doRequest("POST", "/collections/"+name+"/release", nil, nil)
}

func (m *MilvusBridge) Insert(name string, items []InsertItem) error {
	return m.doRequest("POST", "/collections/"+name+"/insert", InsertRequest{Items: items}, nil)
}

func (m *MilvusBridge) DeleteByFilter(name, filter string) error {
	return m.doRequest("POST", "/collections/"+name+"/delete", map[string]string{"filter": filter}, nil)
}

func (m *MilvusBridge) Search(name string, req SearchRequest) (*SearchResponse, error) {
	var result SearchResponse
	if err := m.doRequest("POST", "/collections/"+name+"/search", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (m *MilvusBridge) Count(name string) (int64, error) {
	var result CountResponse
	if err := m.doRequest("GET", "/collections/"+name+"/count", nil, &result); err != nil {
		return 0, err
	}
	return result.NumEntities, nil
}
