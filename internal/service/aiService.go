package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type AIService struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewAIService() (*AIService, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, errors.New("OPENAI_API_KEY is required")
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &AIService{
		apiKey: key,
		model:  model,
		http: &http.Client{
			Timeout: 25 * time.Second,
		},
	}, nil
}

type NutritionEstimate struct {
	Kcal     int `json:"kcal"`
	ProteinG int `json:"protein_g"`
	FatG     int `json:"fat_g"`
	CarbsG   int `json:"carbs_g"`
	// можно расширять: confidence, notes
}

type chatReq struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	MaxTokens      int           `json:"max_tokens"`
	Temperature    float64       `json:"temperature"`
	ResponseFormat any           `json:"response_format"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (a *AIService) EstimateNutrition(ctx context.Context, mealText string) (NutritionEstimate, any, error) {
	schema := map[string]any{
		"name":   "nutrition_estimate",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"kcal":      map[string]any{"type": "integer", "minimum": 0, "maximum": 10000},
				"protein_g": map[string]any{"type": "integer", "minimum": 0, "maximum": 1000},
				"fat_g":     map[string]any{"type": "integer", "minimum": 0, "maximum": 1000},
				"carbs_g":   map[string]any{"type": "integer", "minimum": 0, "maximum": 2000},
			},
			"required":             []string{"kcal", "protein_g", "fat_g", "carbs_g"},
			"additionalProperties": false,
		},
	}

	sys := `Ты нутрициолог-калькулятор. Оцени КБЖУ для одного приема пищи по описанию пользователя.
Правила:
- Если не хватает граммовок, делай реалистичные допущения.
- Возвращай только JSON по схеме. Без текста.`
	user := fmt.Sprintf("Еда: %s", mealText)

	reqBody := chatReq{
		Model: a.model,
		Messages: []chatMessage{
			{Role: "system", Content: sys},
			{Role: "user", Content: user},
		},
		MaxTokens:   120,
		Temperature: 0,
		ResponseFormat: map[string]any{
			"type":        "json_schema",
			"json_schema": schema,
		},
	}

	b, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(b))
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return NutritionEstimate{}, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return NutritionEstimate{}, nil, fmt.Errorf("openai status=%d", resp.StatusCode)
	}

	var out chatResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return NutritionEstimate{}, nil, err
	}
	if len(out.Choices) == 0 {
		return NutritionEstimate{}, nil, errors.New("openai: empty choices")
	}

	content := out.Choices[0].Message.Content

	var est NutritionEstimate
	if err := json.Unmarshal([]byte(content), &est); err != nil {
		return NutritionEstimate{}, map[string]any{"raw": content}, err
	}

	// сырой ответ сохраним
	var raw any
	_ = json.Unmarshal([]byte(content), &raw)

	return est, raw, nil
}
