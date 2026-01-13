package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
}

// ===== Responses API =====

type respReq struct {
	Model           string       `json:"model"`
	Input           []respMsg    `json:"input"`
	Instructions    string       `json:"instructions,omitempty"`
	Temperature     float64      `json:"temperature,omitempty"`
	MaxOutputTokens int          `json:"max_output_tokens,omitempty"`
	Text            *respTextCfg `json:"text,omitempty"`
}

type respMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type respTextCfg struct {
	Format any `json:"format,omitempty"`
}

type respError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Code    any    `json:"code,omitempty"`
}

type respOut struct {
	ID     string     `json:"id"`
	Object string     `json:"object"`
	Status string     `json:"status"`
	Error  *respError `json:"error,omitempty"`
	Output []struct {
		Type    string `json:"type"`
		Role    string `json:"role,omitempty"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content,omitempty"`
	} `json:"output"`
}

func (a *AIService) EstimateNutrition(ctx context.Context, mealText string) (NutritionEstimate, any, error) {
	// JSON Schema (строгий)
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"kcal":      map[string]any{"type": "integer", "minimum": 0, "maximum": 10000},
			"protein_g": map[string]any{"type": "integer", "minimum": 0, "maximum": 1000},
			"fat_g":     map[string]any{"type": "integer", "minimum": 0, "maximum": 1000},
			"carbs_g":   map[string]any{"type": "integer", "minimum": 0, "maximum": 2000},
		},
		"required":             []string{"kcal", "protein_g", "fat_g", "carbs_g"},
		"additionalProperties": false,
	}

	instructions := `Ты нутрициолог-калькулятор. Оцени КБЖУ для ОДНОГО приема пищи по описанию пользователя.
Правила:
- Если не хватает граммовок — делай реалистичные допущения (например: яйца 1шт ~ 50г).
- Не пиши пояснений. Верни только JSON по схеме.
- Значения целые числа.`

	reqBody := respReq{
		Model: a.model,
		Input: []respMsg{
			{Role: "user", Content: fmt.Sprintf("Еда: %s", mealText)},
		},
		Instructions:    instructions,
		Temperature:     0,
		MaxOutputTokens: 120,
		Text: &respTextCfg{
			Format: map[string]any{
				"type":   "json_schema",
				"strict": true,
				"schema": schema,
				"name":   "nutrition_estimate",
			},
		},
	}

	b, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewReader(b))
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return NutritionEstimate{}, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return NutritionEstimate{}, map[string]any{
			"status": resp.StatusCode,
			"body":   string(body),
		}, fmt.Errorf("openai status=%d body=%s", resp.StatusCode, string(body))
	}

	var out respOut
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return NutritionEstimate{}, nil, err
	}

	if out.Error != nil {
		return NutritionEstimate{}, map[string]any{"error": out.Error}, fmt.Errorf("openai error: %s", out.Error.Message)
	}

	// В Responses API результат обычно лежит в output[].content[].text (type=output_text).  [oai_citation:1‡OpenAI Platform](https://platform.openai.com/docs/api-reference/responses)
	rawText := ""
	for _, item := range out.Output {
		for _, c := range item.Content {
			if c.Type == "output_text" && c.Text != "" {
				rawText += c.Text
			}
		}
	}
	rawText = strings.TrimSpace(rawText)
	if rawText == "" {
		return NutritionEstimate{}, out, errors.New("openai: empty output_text")
	}

	var est NutritionEstimate
	if err := json.Unmarshal([]byte(rawText), &est); err != nil {
		// сохраним сырьё для db.ai_raw
		return NutritionEstimate{}, map[string]any{"raw": rawText, "response": out}, err
	}

	// ai_raw сохраняем уже распарсенное (удобно смотреть в jsonb)
	var raw any
	_ = json.Unmarshal([]byte(rawText), &raw)

	return est, raw, nil
}

type ActivityEstimate struct {
	ActivityMultiplier float64 `json:"activity_multiplier"`
}

func (a *AIService) EstimateActivityMultiplier(ctx context.Context, planText string) (float64, any, error) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"activity_multiplier": map[string]any{"type": "number", "minimum": 1.1, "maximum": 2.2},
		},
		"required":             []string{"activity_multiplier"},
		"additionalProperties": false,
	}

	instructions := `Ты тренер. По недельному плану тренировок оцени средний коэффициент активности (TDEE multiplier).
Диапазон 1.2..1.9. Учитывай частоту и интенсивность тренировок.
Верни только JSON по схеме.`

	reqBody := respReq{
		Model: a.model,
		Input: []respMsg{
			{Role: "user", Content: fmt.Sprintf("План на неделю:\n%s", planText)},
		},
		Instructions:    instructions,
		Temperature:     0,
		MaxOutputTokens: 120,
		Text: &respTextCfg{
			Format: map[string]any{
				"type":   "json_schema",
				"strict": true,
				"schema": schema,
				"name":   "activity_estimate",
			},
		},
	}

	b, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewReader(b))
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, map[string]any{
			"status": resp.StatusCode,
			"body":   string(body),
		}, fmt.Errorf("openai status=%d body=%s", resp.StatusCode, string(body))
	}

	var out respOut
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, nil, err
	}

	if out.Error != nil {
		return 0, map[string]any{"error": out.Error}, fmt.Errorf("openai error: %s", out.Error.Message)
	}

	rawText := ""
	for _, item := range out.Output {
		for _, c := range item.Content {
			if c.Type == "output_text" && c.Text != "" {
				rawText += c.Text
			}
		}
	}
	rawText = strings.TrimSpace(rawText)
	if rawText == "" {
		return 0, out, errors.New("openai: empty output_text")
	}

	var est ActivityEstimate
	if err := json.Unmarshal([]byte(rawText), &est); err != nil {
		return 0, map[string]any{"raw": rawText, "response": out}, err
	}

	var raw any
	_ = json.Unmarshal([]byte(rawText), &raw)

	return est.ActivityMultiplier, raw, nil
}
