package services

import (
	"strings"

	"arke/backend/models"
)

type Chunker struct {
	strategy models.ChunkStrategy
	size     int
	overlap  int
}

func NewChunker(strategy models.ChunkStrategy, size, overlap int) *Chunker {
	if size <= 0 {
		size = 500
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 4
	}
	return &Chunker{
		strategy: strategy,
		size:     size,
		overlap:  overlap,
	}
}

type ChunkItem struct {
	Index  int
	Title  string
	Content string
}

func (c *Chunker) Split(content string) []ChunkItem {
	content = strings.TrimSpace(content)
	if content == "" {
		return []ChunkItem{}
	}

	switch c.strategy {
	case models.ChunkStrategyNone:
		return []ChunkItem{{Index: 0, Title: "全文", Content: content}}
	case models.ChunkStrategyFixed:
		return c.splitFixed(content)
	case models.ChunkStrategySentence:
		return c.splitSentence(content)
	default:
		return c.splitParagraph(content)
	}
}

func (c *Chunker) splitParagraph(content string) []ChunkItem {
	// 优先按段落分割；过长的段落按固定大小再切分
	paragraphs := splitByParagraphs(content)
	var chunks []ChunkItem
	idx := 0
	var buffer strings.Builder
	bufferTitle := ""

	flush := func() {
		text := strings.TrimSpace(buffer.String())
		if text != "" {
			chunks = append(chunks, ChunkItem{Index: idx, Title: bufferTitle, Content: text})
			idx++
		}
		buffer.Reset()
		bufferTitle = ""
	}

	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		title := extractTitle(p)

		// 如果当前段落长度超过 size，按固定大小切分
		if len([]rune(p)) > c.size {
			flush()
			subChunks := c.splitFixed(p)
			for _, sc := range subChunks {
				chunks = append(chunks, ChunkItem{Index: idx, Title: title, Content: sc.Content})
				idx++
			}
			continue
		}

		// 累积：超过 size 时 flush
		if len([]rune(buffer.String()))+len([]rune(p)) > c.size && buffer.Len() > 0 {
			flush()
		}
		if buffer.Len() == 0 {
			bufferTitle = title
		}
		buffer.WriteString(p)
		buffer.WriteString("\n")
	}
	flush()
	return chunks
}

func (c *Chunker) splitFixed(content string) []ChunkItem {
	runes := []rune(content)
	if len(runes) == 0 {
		return []ChunkItem{}
	}
	if c.size <= 0 {
		return []ChunkItem{{Index: 0, Title: "全文", Content: content}}
	}

	var chunks []ChunkItem
	step := c.size - c.overlap
	if step <= 0 {
		step = c.size
	}
	idx := 0
	for start := 0; start < len(runes); start += step {
		end := start + c.size
		if end > len(runes) {
			end = len(runes)
		}
		text := strings.TrimSpace(string(runes[start:end]))
		if text != "" {
			chunks = append(chunks, ChunkItem{Index: idx, Title: "片段", Content: text})
			idx++
		}
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func (c *Chunker) splitSentence(content string) []ChunkItem {
	sentences := splitBySentences(content)
	var chunks []ChunkItem
	var buffer strings.Builder
	idx := 0
	flush := func() {
		text := strings.TrimSpace(buffer.String())
		if text != "" {
			chunks = append(chunks, ChunkItem{Index: idx, Title: "片段", Content: text})
			idx++
		}
		buffer.Reset()
	}

	for _, s := range sentences {
		if len([]rune(buffer.String()))+len([]rune(s)) > c.size && buffer.Len() > 0 {
			flush()
		}
		buffer.WriteString(s)
	}
	flush()
	return chunks
}

func splitByParagraphs(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(content, "\n")
}

func splitBySentences(content string) []string {
	delimiters := []string{"。", "！", "？", "；", ". ", "! ", "? ", "; "}
	result := []string{content}
	for _, d := range delimiters {
		var next []string
		for _, r := range result {
			parts := strings.Split(r, d)
			for i, p := range parts {
				if i < len(parts)-1 {
					next = append(next, p+d)
				} else {
					next = append(next, p)
				}
			}
		}
		result = next
	}
	var sentences []string
	for _, s := range result {
		s = strings.TrimSpace(s)
		if s != "" {
			sentences = append(sentences, s)
		}
	}
	return sentences
}

func extractTitle(paragraph string) string {
	// 提取首行作为标题
	paragraph = strings.TrimSpace(paragraph)
	if idx := strings.Index(paragraph, "\n"); idx > 0 {
		paragraph = paragraph[:idx]
	}
	if idx := strings.Index(paragraph, "。"); idx > 0 && idx < 50 {
		paragraph = paragraph[:idx]
	}
	runes := []rune(paragraph)
	if len(runes) > 30 {
		paragraph = string(runes[:30]) + "..."
	}
	if paragraph == "" {
		return "片段"
	}
	return paragraph
}
