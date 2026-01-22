package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// SearchParams は検索パラメータ（自由形式キーワード対応）
type SearchParams struct {
	Keywords []string `json:"keywords"` // 自由形式キーワード配列
	FloorID  *string  `json:"floor_id"`
}

// GeminiService は Gemini API ラッパー
type GeminiService struct {
	client *genai.Client
	model  string
}

// NewGeminiService は GeminiService を初期化
func NewGeminiService(ctx context.Context) (*GeminiService, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	modelName := os.Getenv("GEMINI_MODEL")
	if modelName == "" {
		modelName = "gemini-2.5-flash-lite"
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiService{
		client: client,
		model:  modelName,
	}, nil
}

// ParseSearchQuery は自然言語クエリからキーワードを抽出
// 自由形式の属性（free_attributes）検索用
func (s *GeminiService) ParseSearchQuery(
	ctx context.Context,
	query string,
) (SearchParams, error) {
	if query == "" {
		return SearchParams{}, fmt.Errorf("query cannot be empty")
	}

	model := s.client.GenerativeModel(s.model)
	temp := float32(0.3)
	model.Temperature = &temp // 低い温度で安定した出力

	prompt := `You are a seat search assistant for an office booking system.
The system manages seat attributes as free-form keywords (e.g., "静か", "充電器あり", "一人向け", "明るい").

Extract keywords from the user's natural language query that match seat attributes.
Return a JSON object with:
- keywords: array of Japanese keywords to search for (e.g., ["静か", "一人向け"])
- floor_id: floor identifier if mentioned (e.g., "2階", "5F"), or null

Return ONLY valid JSON, nothing else. Do not include markdown or code blocks.

Examples:
- User: "静かな場所がいい" → {"keywords": ["静か"], "floor_id": null}
- User: "電源付きで一人向けの座席" → {"keywords": ["充電器あり", "一人向け"], "floor_id": null}
- User: "2階の明るい座席" → {"keywords": ["明るい"], "floor_id": "2階"}
- User: "集中できる場所" → {"keywords": ["静か"], "floor_id": null}
- User: "グループで作業できる広い座席" → {"keywords": ["グループ"], "floor_id": null}

Guidelines:
1. Extract 1-3 most relevant keywords
2. Keep keywords concise (1-3 words each)
3. Use Japanese when user uses Japanese
4. For "静か" related queries, include "静か"
5. Return empty array [] if no clear attributes are mentioned

User query: ` + query

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return SearchParams{}, fmt.Errorf("failed to generate content with Gemini: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return SearchParams{}, fmt.Errorf("no candidates returned from Gemini")
	}

	if len(resp.Candidates[0].Content.Parts) == 0 {
		return SearchParams{}, fmt.Errorf("no content parts returned from Gemini")
	}

	text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok {
		return SearchParams{}, fmt.Errorf("unexpected response format from Gemini")
	}

	// デバッグログ
	log.Printf("[AI Search] Query: %s, Gemini Response: %s", query, string(text))

	// JSON解析
	var params SearchParams
	if err := json.Unmarshal([]byte(text), &params); err != nil {
		log.Printf("[AI Search] JSON Parse Error: %v, Response: %s", err, string(text))
		return SearchParams{}, fmt.Errorf("failed to parse Gemini response as JSON: %w", err)
	}

	// キーワード正規化（前後の空白を除去）
	for i, kw := range params.Keywords {
		params.Keywords[i] = strings.TrimSpace(kw)
	}

	log.Printf("[AI Search] Extracted Keywords: %v, FloorID: %v", params.Keywords, params.FloorID)

	return params, nil
}

// Close はクライアントを閉じる
func (s *GeminiService) Close() error {
	return s.client.Close()
}
