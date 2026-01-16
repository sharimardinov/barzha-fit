package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type NutritionEstimate struct {
	Kcal     int `json:"kcal"`
	ProteinG int `json:"protein_g"`
	FatG     int `json:"fat_g"`
	CarbsG   int `json:"carbs_g"`
}

type NutritionAI struct {
	client *AIClient
}

func NewNutritionAI(client *AIClient) *NutritionAI {
	return &NutritionAI{client: client}
}

func (a *NutritionAI) EstimateNutrition(ctx context.Context, mealText string) (NutritionEstimate, any, error) {
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
		Model: a.client.model,
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

	out, body, status, err := a.client.postResponses(ctx, reqBody)
	if err != nil {
		return NutritionEstimate{}, nil, err
	}

	if status < 200 || status >= 300 {
		return NutritionEstimate{}, map[string]any{
			"status": status,
			"body":   string(body),
		}, fmt.Errorf("openai status=%d body=%s", status, string(body))
	}

	if err := json.Unmarshal(body, &out); err != nil {
		return NutritionEstimate{}, nil, err
	}

	if out.Error != nil {
		return NutritionEstimate{}, map[string]any{"error": out.Error}, fmt.Errorf("openai error: %s", out.Error.Message)
	}

	rawText := extractOutputText(out)
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
