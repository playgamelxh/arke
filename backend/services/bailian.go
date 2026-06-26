package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"arke/backend/config"
	"arke/backend/models"
)

type BailianClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	Temperature    float64       `json:"temperature"`
	ResponseFormat any           `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

func NewBailianClient(cfg config.Config) *BailianClient {
	timeout := cfg.DashScopeTimeout
	if timeout <= 0 {
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
	return &BailianClient{
		apiKey:  cfg.DashScopeAPIKey,
		baseURL: cfg.DashScopeBaseURL,
		model:   cfg.DashScopeModel,
		client:  &http.Client{Timeout: timeout, Transport: transport},
	}
}

func (b *BailianClient) GenerateQA(segments []models.DocumentSegment, count int, difficulty, instruction string, existing []models.GeneratedQAItem) ([]models.GeneratedQAItem, error) {
	if b.apiKey == "" {
		return nil, errors.New("未配置 DASHSCOPE_API_KEY，无法调用阿里百炼生成问答")
	}
	if len(segments) == 0 {
		return []models.GeneratedQAItem{}, nil
	}
	prompt := buildLLMPrompt(segments, count, difficulty, instruction, existing)
	payload := chatRequest{Model: b.model, Temperature: 0.2, ResponseFormat: map[string]any{"type": "json_object"}, Messages: []chatMessage{{Role: "system", Content: "你是企业知识库问答生成专家。你只能基于用户提供的知识库切片生成问答，不能编造知识库中不存在的信息。输出必须是合法 JSON。"}, {Role: "user", Content: prompt}}}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, b.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, formatBailianRequestError(err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, formatBailianHTTPError(resp.StatusCode, respBody)
	}
	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("解析阿里百炼响应失败：%w", err)
	}
	if chatResp.Error != nil {
		return nil, fmt.Errorf("阿里百炼错误：%s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return nil, errors.New("阿里百炼未返回生成结果")
	}
	items, err := parseGeneratedQA(chatResp.Choices[0].Message.Content, segments, count)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (b *BailianClient) GenerateAnswer(segments []models.DocumentSegment, question string) (models.GenerateAnswerResponse, error) {
	if b.apiKey == "" {
		return models.GenerateAnswerResponse{}, fmt.Errorf("未配置 DASHSCOPE_API_KEY，无法调用阿里百炼生成答案")
	}
	content := documentContentFromSegments(segments)
	if content == "" {
		return models.GenerateAnswerResponse{}, fmt.Errorf("文档无可用解析内容")
	}

	prompt := buildAnswerPrompt(content, question)
	payload := chatRequest{
		Model:          b.model,
		Temperature:    0.2,
		ResponseFormat: map[string]any{"type": "json_object"},
		Messages: []chatMessage{
			{Role: "system", Content: "你是企业知识库问答助手。只能基于用户提供的整篇文档内容回答问题，不能编造文档中不存在的信息。输出必须是合法 JSON。"},
			{Role: "user", Content: prompt},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, b.baseURL, bytes.NewReader(body))
	if err != nil {
		return models.GenerateAnswerResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+b.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return models.GenerateAnswerResponse{}, formatBailianRequestError(err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.GenerateAnswerResponse{}, formatBailianHTTPError(resp.StatusCode, respBody)
	}
	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return models.GenerateAnswerResponse{}, fmt.Errorf("解析阿里百炼响应失败：%w", err)
	}
	if chatResp.Error != nil {
		return models.GenerateAnswerResponse{}, fmt.Errorf("阿里百炼错误：%s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return models.GenerateAnswerResponse{}, fmt.Errorf("阿里百炼未返回生成结果")
	}
	return parseGeneratedAnswer(chatResp.Choices[0].Message.Content)
}

func (b *BailianClient) GenerateKnowledgeAnswer(sources []models.KnowledgeAskSource, question string) (models.GenerateAnswerResponse, error) {
	if b.apiKey == "" {
		return models.GenerateAnswerResponse{}, fmt.Errorf("未配置 DASHSCOPE_API_KEY，无法调用阿里百炼生成答案")
	}
	content := knowledgeAskContentFromSources(sources)
	if content == "" {
		return models.GenerateAnswerResponse{}, fmt.Errorf("召回内容为空，无法生成答案")
	}

	prompt := buildKnowledgeAnswerPrompt(content, question)
	payload := chatRequest{
		Model:          b.model,
		Temperature:    0.2,
		ResponseFormat: map[string]any{"type": "json_object"},
		Messages: []chatMessage{
			{Role: "system", Content: "你是企业知识库问答助手。只能基于已召回的知识库内容回答问题，不能编造召回内容中不存在的信息。输出必须是合法 JSON。"},
			{Role: "user", Content: prompt},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, b.baseURL, bytes.NewReader(body))
	if err != nil {
		return models.GenerateAnswerResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+b.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return models.GenerateAnswerResponse{}, formatBailianRequestError(err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.GenerateAnswerResponse{}, formatBailianHTTPError(resp.StatusCode, respBody)
	}
	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return models.GenerateAnswerResponse{}, fmt.Errorf("解析阿里百炼响应失败：%w", err)
	}
	if chatResp.Error != nil {
		return models.GenerateAnswerResponse{}, fmt.Errorf("阿里百炼错误：%s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return models.GenerateAnswerResponse{}, fmt.Errorf("阿里百炼未返回生成结果")
	}
	return parseGeneratedAnswer(chatResp.Choices[0].Message.Content)
}

func formatBailianRequestError(err error) error {
	msg := err.Error()
	if dnsErr, ok := err.(*url.Error); ok && dnsErr.Err != nil && strings.Contains(dnsErr.Err.Error(), "no such host") {
		return errors.New("无法解析 dashscope.aliyuncs.com，请检查 Docker DNS 或本机网络")
	}
	if strings.Contains(msg, "no such host") || strings.Contains(msg, "lookup dashscope.aliyuncs.com") {
		return errors.New("无法解析 dashscope.aliyuncs.com，请检查 Docker DNS 或本机网络连接")
	}
	if strings.Contains(msg, "context deadline exceeded") || strings.Contains(msg, "Client.Timeout") {
		return errors.New("调用阿里百炼超时，请减少生成数量后重试")
	}
	if strings.Contains(msg, "TLS handshake timeout") {
		return errors.New("连接阿里百炼 TLS 握手超时，请检查网络连接或稍后重试")
	}
	return fmt.Errorf("调用阿里百炼失败：%w", err)
}

func formatBailianHTTPError(status int, body []byte) error {
	raw := strings.TrimSpace(string(body))
	apiMsg := extractDashScopeErrorMessage(raw)
	switch {
	case status == http.StatusForbidden && strings.Contains(strings.ToLower(apiMsg+raw), "ip access denied"):
		return errors.New("阿里百炼 API Key 启用了 IP 白名单限制，当前服务器出口 IP 未在白名单中。请登录百炼控制台 → API Key 管理，关闭 IP 限制或将本机公网 IP 加入白名单")
	case status == http.StatusUnauthorized:
		return errors.New("阿里百炼 API Key 无效或已过期，请检查 DASHSCOPE_API_KEY 配置")
	case apiMsg != "":
		return fmt.Errorf("阿里百炼返回异常（%d）：%s", status, apiMsg)
	case raw != "":
		return fmt.Errorf("阿里百炼返回异常状态 %d：%s", status, truncateText(raw, 300))
	default:
		return fmt.Errorf("阿里百炼返回异常状态 %d", status)
	}
}

func extractDashScopeErrorMessage(raw string) string {
	var payload struct {
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Error.Message)
}

func buildLLMPrompt(segments []models.DocumentSegment, count int, difficulty, instruction string, existing []models.GeneratedQAItem) string {
	content := knowledgeBaseContentFromSegments(segments)
	instruction = strings.TrimSpace(instruction)
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("请基于以下知识库切片生成 %d 条问答，难度为 %s。\n", count, difficulty))
	if instruction != "" {
		builder.WriteString("用户生成要求（必须优先满足，可包含指定问题、关注主题、答案侧重点等）：\n")
		builder.WriteString(instruction)
		builder.WriteString("\n\n")
	}
	if len(existing) > 0 {
		builder.WriteString("以下问题已生成，请勿重复，需生成不同角度的新问答：\n")
		for i, item := range existing {
			if i >= 20 {
				break
			}
			builder.WriteString(fmt.Sprintf("- %s\n", item.Question))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("返回 JSON 对象，格式为：{\"items\":[{\"question\":\"问题\",\"answer\":\"答案\",\"keywords\":[\"关键词1\",\"关键词2\"],\"sourceSegmentId\":123,\"sourceExcerpt\":\"来源原文摘录\",\"confidence\":0.9}]}。\n")
	if instruction != "" {
		builder.WriteString("要求：若用户指定了具体问题，需按指定问题生成对应答案；其余问答也需符合用户要求；问题清晰，答案准确，必须基于知识库切片；sourceSegmentId 必须填写所引用切片的编号；keywords 为 2-5 个从答案提取的核心关键词；sourceExcerpt 需引用切片原文；不要输出 Markdown。\n\n")
	} else {
		builder.WriteString("要求：问题清晰，答案准确，必须基于知识库切片；sourceSegmentId 必须填写所引用切片的编号；keywords 为 2-5 个从答案提取的核心关键词；sourceExcerpt 需引用切片原文；不要输出 Markdown。\n\n")
	}
	builder.WriteString("知识库切片：\n")
	builder.WriteString(content)
	return builder.String()
}

func buildAnswerPrompt(content, question string) string {
	var builder strings.Builder
	builder.WriteString("请基于以下整篇文档内容，回答用户问题。\n")
	builder.WriteString("返回 JSON 对象，格式为：{\"answer\":\"答案正文\",\"sourceExcerpt\":\"来源原文摘录\",\"confidence\":0.9}。\n")
	builder.WriteString("要求：答案准确完整，仅使用文档中的信息；sourceExcerpt 必须引用原文；不要输出 Markdown。\n\n")
	builder.WriteString(fmt.Sprintf("用户问题：%s\n\n", question))
	builder.WriteString("文档全文：\n")
	builder.WriteString(content)
	return builder.String()
}

func buildKnowledgeAnswerPrompt(content, question string) string {
	var builder strings.Builder
	builder.WriteString("请基于以下已经召回并重排后的知识库内容，回答用户问题。\n")
	builder.WriteString("返回 JSON 对象，格式为：{\"answer\":\"答案正文\",\"sourceExcerpt\":\"来源原文摘录\",\"confidence\":0.9}。\n")
	builder.WriteString("要求：答案准确完整，仅使用召回内容中的信息；如果召回内容不足以回答，要明确说明依据不足；sourceExcerpt 必须引用最关键的来源原文；不要输出 Markdown。\n\n")
	builder.WriteString(fmt.Sprintf("用户问题：%s\n\n", question))
	builder.WriteString("召回内容：\n")
	builder.WriteString(content)
	return builder.String()
}

func knowledgeAskContentFromSources(sources []models.KnowledgeAskSource) string {
	parts := make([]string, 0, len(sources))
	for index, source := range sources {
		if text := strings.TrimSpace(cleanText(source.Content)); text != "" {
			segmentID := uint(0)
			if source.SourceSegmentID != nil {
				segmentID = *source.SourceSegmentID
			}
			parts = append(parts, fmt.Sprintf("[来源%d][文档ID:%d][切片ID:%d][相似度:%.4f]\n%s", index+1, source.DocumentID, segmentID, source.Score, text))
		}
	}
	return strings.Join(parts, "\n\n")
}

func documentContentFromSegments(segments []models.DocumentSegment) string {
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		if text := strings.TrimSpace(cleanText(segment.Content)); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

func knowledgeBaseContentFromSegments(segments []models.DocumentSegment) string {
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		if text := strings.TrimSpace(cleanText(segment.Content)); text != "" {
			parts = append(parts, fmt.Sprintf("[切片编号:%d][文档ID:%d][标题:%s]\n%s", segment.ID, segment.DocumentID, segment.Title, text))
		}
	}
	return strings.Join(parts, "\n\n")
}

func parseGeneratedQA(content string, segments []models.DocumentSegment, count int) ([]models.GeneratedQAItem, error) {
	content = stripJSONFence(content)
	var payload struct {
		Items []models.GeneratedQAItem `json:"items"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		var direct []models.GeneratedQAItem
		if err2 := json.Unmarshal([]byte(content), &direct); err2 != nil {
			return nil, fmt.Errorf("阿里百炼返回内容不是有效问答 JSON：%w", err)
		}
		payload.Items = direct
	}
	segmentMap := map[uint]models.DocumentSegment{}
	for _, segment := range segments {
		segmentMap[segment.ID] = segment
	}
	items := make([]models.GeneratedQAItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		item.Question = strings.TrimSpace(item.Question)
		item.Answer = strings.TrimSpace(item.Answer)
		item.SourceExcerpt = strings.TrimSpace(item.SourceExcerpt)
		item.Keywords = normalizeKeywords(item.Keywords)
		if len(item.Keywords) == 0 && item.Answer != "" {
			item.Keywords = extractKeywordsFromAnswer(item.Answer)
		}
		if item.Question == "" || item.Answer == "" {
			continue
		}
		if item.SourceSegmentID != nil {
			if segment, exists := segmentMap[*item.SourceSegmentID]; exists {
				item.DocumentID = segment.DocumentID
				if item.SourceExcerpt == "" {
					item.SourceExcerpt = truncateText(segment.Content, 120)
				}
			}
		}
		if item.SourceSegmentID == nil || item.DocumentID == 0 {
			for _, segment := range segments {
				if item.SourceExcerpt != "" && strings.Contains(segment.Content, item.SourceExcerpt) {
					segmentID := segment.ID
					item.SourceSegmentID = &segmentID
					item.DocumentID = segment.DocumentID
					break
				}
			}
		}
		if item.SourceSegmentID == nil || item.DocumentID == 0 {
			segmentID := segments[0].ID
			item.SourceSegmentID = &segmentID
			item.DocumentID = segments[0].DocumentID
			if item.SourceExcerpt == "" {
				item.SourceExcerpt = truncateText(segments[0].Content, 120)
			}
		}
		if item.Confidence <= 0 || item.Confidence > 1 {
			item.Confidence = 0.9
		}
		items = append(items, item)
		if len(items) >= count {
			break
		}
	}
	return items, nil
}

func parseGeneratedAnswer(content string) (models.GenerateAnswerResponse, error) {
	content = stripJSONFence(content)
	var payload struct {
		Answer        string  `json:"answer"`
		SourceExcerpt string  `json:"sourceExcerpt"`
		Confidence    float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return models.GenerateAnswerResponse{}, fmt.Errorf("阿里百炼返回内容不是有效 JSON：%w", err)
	}
	answer := strings.TrimSpace(payload.Answer)
	if answer == "" {
		return models.GenerateAnswerResponse{}, fmt.Errorf("大模型未返回有效答案")
	}
	confidence := payload.Confidence
	if confidence <= 0 || confidence > 1 {
		confidence = 0.9
	}
	return models.GenerateAnswerResponse{
		Answer:        answer,
		SourceExcerpt: strings.TrimSpace(payload.SourceExcerpt),
		Confidence:    confidence,
	}, nil
}

func stripJSONFence(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}

func cleanText(text string) string {
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = strings.ReplaceAll(text, "\u200b", "")
	text = strings.ReplaceAll(text, "\ufeff", "")
	text = strings.ReplaceAll(text, "<[^>]+>", "")
	text = strings.ReplaceAll(text, "[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]", "")
	text = strings.ReplaceAll(text, "[\t\r]+", " ")
	text = strings.ReplaceAll(text, "\n{3,}", "\n\n")
	text = strings.ReplaceAll(text, " {2,}", " ")
	return strings.TrimSpace(text)
}

func truncateText(text string, limit int) string {
	text = cleanText(text)
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

func normalizeKeywords(keywords []string) []string {
	var result []string
	seen := map[string]bool{}
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw != "" && !seen[kw] {
			seen[kw] = true
			result = append(result, kw)
		}
	}
	if len(result) > 5 {
		result = result[:5]
	}
	return result
}

func extractKeywordsFromAnswer(answer string) []string {
	answer = cleanText(answer)
	if answer == "" {
		return nil
	}
	words := strings.Fields(answer)
	var result []string
	seen := map[string]bool{}
	for _, word := range words {
		if len(word) >= 2 && len(word) <= 10 && !seen[word] {
			seen[word] = true
			result = append(result, word)
			if len(result) >= 5 {
				break
			}
		}
	}
	return result
}
