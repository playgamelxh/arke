package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"arke/backend/models"
)

type ParseOptions struct {
	Engine            string
	PDFNativeFallback bool
	MinerUBaseURL     string
	MinerUTimeout     int
	MinerUParseMethod string
	MinerUEffort      string
	MinerULanguage    string
	ImageAnalysis     bool
	TableEnable       bool
	FormulaEnable     bool
}

type MinerUClient struct {
	baseURL string
	client  *http.Client
	options ParseOptions
}

func NewMinerUClient(baseURL string, timeout time.Duration) *MinerUClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil
	}
	if timeout <= 0 {
		timeout = 300 * time.Second
	}
	if timeout > 300*time.Second {
		timeout = 300 * time.Second
	}
	tlsTimeout := 120 * time.Second
	if tlsTimeout > timeout {
		tlsTimeout = timeout
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   tlsTimeout,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
	return &MinerUClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout, Transport: transport},
		options: ParseOptions{
			MinerUParseMethod: "auto",
			MinerUEffort:      "medium",
			MinerULanguage:    "ch",
			ImageAnalysis:     true,
			TableEnable:       true,
			FormulaEnable:     true,
		},
	}
}

func NewMinerUClientWithOptions(options ParseOptions) *MinerUClient {
	return NewMinerUClientWithDetailOptions(options.MinerUBaseURL, time.Duration(options.MinerUTimeout)*time.Second, options)
}

func NewMinerUClientWithDetailOptions(baseURL string, timeout time.Duration, options ParseOptions) *MinerUClient {
	client := NewMinerUClient(baseURL, timeout)
	if client == nil {
		return nil
	}
	client.options = normalizeParseOptions(options)
	return client
}

func normalizeParseOptions(options ParseOptions) ParseOptions {
	if options.MinerUParseMethod == "" {
		options.MinerUParseMethod = "auto"
	}
	if options.MinerUEffort == "" {
		options.MinerUEffort = "medium"
	}
	if options.MinerULanguage == "" {
		options.MinerULanguage = "ch"
	}
	return options
}

func (m *MinerUClient) Enabled() bool {
	return m != nil && m.baseURL != ""
}

func (m *MinerUClient) TestConnection() error {
	if !m.Enabled() {
		return errors.New("MinerU 服务地址不能为空")
	}
	endpoints := []string{"/docs", "/openapi.json", "/health"}
	var lastErr error
	for _, endpoint := range endpoints {
		req, err := http.NewRequest(http.MethodGet, m.baseURL+endpoint, nil)
		if err != nil {
			return err
		}
		resp, err := m.client.Do(req)
		if err != nil {
			lastErr = formatMinerURequestError(err)
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			return nil
		}
		lastErr = fmt.Errorf("MinerU %s 返回状态 %d", endpoint, resp.StatusCode)
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("MinerU 检测失败")
}

func (m *MinerUClient) ParseFile(filePath, originalName string) ([]models.DocumentSegment, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败：%w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fields := map[string]string{
		"image_analysis":                strconv.FormatBool(m.options.ImageAnalysis),
		"client_side_output_generation": "false",
		"return_middle_json":            "false",
		"return_model_output":           "false",
		"return_md":                     "true",
		"return_images":                 "false",
		"end_page_id":                   "99999",
		"effort":                        m.options.MinerUEffort,
		"parse_method":                  m.options.MinerUParseMethod,
		"start_page_id":                 "0",
		"lang_list":                     m.options.MinerULanguage,
		"return_content_list":           "false",
		"backend":                       "pipeline",
		"table_enable":                  strconv.FormatBool(m.options.TableEnable),
		"response_format_zip":           "false",
		"return_original_file":          "false",
		"formula_enable":                strconv.FormatBool(m.options.FormulaEnable),
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, err
		}
	}

	uploadName := originalName
	if uploadName == "" {
		uploadName = filepath.Base(filePath)
	}
	part, err := writer.CreateFormFile("files", uploadName)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, m.baseURL+"/file_parse", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, formatMinerURequestError(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			return nil, fmt.Errorf("MinerU 返回异常状态 %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("MinerU 返回异常状态 %d：%s", resp.StatusCode, truncateText(msg, 300))
	}

	md, err := extractMinerUMarkdown(respBody)
	if err != nil {
		return nil, err
	}
	segments := markdownToDocumentContent(md)
	if len(segments) == 0 {
		return nil, errors.New("MinerU 未返回可用文本内容")
	}
	return segments, nil
}

func extractMinerUMarkdown(body []byte) (string, error) {
	var payload struct {
		Results map[string]json.RawMessage `json:"results"`
		Message string                     `json:"message"`
		Detail  any                        `json:"detail"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("解析 MinerU 响应失败：%w", err)
	}
	if len(payload.Results) == 0 {
		if payload.Message != "" {
			return "", fmt.Errorf("MinerU 解析失败：%s", payload.Message)
		}
		return "", errors.New("MinerU 响应中无 results 字段")
	}
	for _, raw := range payload.Results {
		var item struct {
			MDContent string `json:"md_content"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		if strings.TrimSpace(item.MDContent) != "" {
			return item.MDContent, nil
		}
		var generic map[string]any
		if err := json.Unmarshal(raw, &generic); err == nil {
			if value, ok := generic["md_content"].(string); ok && strings.TrimSpace(value) != "" {
				return value, nil
			}
		}
	}
	return "", errors.New("MinerU 未返回 md_content")
}

func formatMinerURequestError(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "TLS handshake timeout") {
		return errors.New("连接 MinerU TLS 握手超时，请检查网络或 MinerU 服务状态")
	}
	if strings.Contains(msg, "context deadline exceeded") || strings.Contains(msg, "Client.Timeout") {
		return errors.New("MinerU 解析超时，请稍后重试")
	}
	if strings.Contains(msg, "connection refused") {
		return errors.New("无法连接 MinerU 服务，请检查 MINERU_BASE_URL 配置")
	}
	return fmt.Errorf("调用 MinerU 失败：%w", err)
}

func markdownToDocumentContent(md string) []models.DocumentSegment {
	md = strings.TrimSpace(md)
	if md == "" {
		return nil
	}
	content := cleanText(normalizeMarkdown(md))
	if content == "" {
		return nil
	}
	return []models.DocumentSegment{{
		SegmentType:  "document",
		SegmentIndex: 1,
		Title:        "全文",
		Content:      content,
	}}
}

func normalizeMarkdown(md string) string {
	md = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`).ReplaceAllString(md, "")
	md = regexp.MustCompile(`\[[^\]]*\]\(([^)]*)\)`).ReplaceAllString(md, "$1")
	md = regexp.MustCompile(`(?m)^#{1,6}\s*`).ReplaceAllString(md, "")
	md = regexp.MustCompile(`(?m)^>\s?`).ReplaceAllString(md, "")
	md = regexp.MustCompile(`(?m)^[-*+]\s+`).ReplaceAllString(md, "")
	md = regexp.MustCompile(`(?m)^\d+\.\s+`).ReplaceAllString(md, "")
	md = regexp.MustCompile("`{1,3}([^`]+)`{1,3}").ReplaceAllString(md, "$1")
	md = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(md, "$1")
	md = regexp.MustCompile(`__([^_]+)__`).ReplaceAllString(md, "$1")
	md = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(md, "$1")
	md = regexp.MustCompile(`_([^_]+)_`).ReplaceAllString(md, "$1")
	return md
}

func mineruSupportedType(fileType string) bool {
	switch fileType {
	case "pdf", "png", "jpg", "jpeg", "webp", "ppt", "pptx", "xls", "xlsx", "doc", "docx":
		return true
	default:
		return false
	}
}
