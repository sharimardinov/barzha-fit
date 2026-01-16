package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"barzhafit/backend/domain"
)

type ActivityEstimate struct {
	ActivityMultiplier float64 `json:"activity_multiplier"`
}

type ActivityAI struct {
	client *AIClient
}

func NewActivityAI(client *AIClient) *ActivityAI {
	return &ActivityAI{client: client}
}

func (a *ActivityAI) EstimateActivityMultiplier(ctx context.Context, planText string) (float64, any, error) {
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
		Model: a.client.model,
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

	out, body, status, err := a.client.postResponses(ctx, reqBody)
	if err != nil {
		return 0, nil, err
	}

	if status < 200 || status >= 300 {
		return 0, map[string]any{
			"status": status,
			"body":   string(body),
		}, fmt.Errorf("openai status=%d body=%s", status, string(body))
	}

	if err := json.Unmarshal(body, &out); err != nil {
		return 0, nil, err
	}

	if out.Error != nil {
		return 0, map[string]any{"error": out.Error}, fmt.Errorf("openai error: %s", out.Error.Message)
	}

	rawText := extractOutputText(out)
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

func (a *ActivityAI) EstimateActivityMultiplierWithProfile(ctx context.Context, planText string, p domain.Profile) (float64, any, error) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"activity_multiplier": map[string]any{"type": "number", "minimum": 1.1, "maximum": 2.2},
		},
		"required":             []string{"activity_multiplier"},
		"additionalProperties": false,
	}

	instructions := `Ты тренер. По плану тренировок и данным профиля оцени консервативный коэффициент активности (TDEE multiplier).
Правила:
- Будь строгим и консервативным, не завышай.
- Диапазон 1.2..1.8.
- Учитывай пол/возраст/вес/рост/%жира только как фон, главный фактор — частота и интенсивность плана.
Верни только JSON по схеме.`

	userText := fmt.Sprintf(
		"План тренировок:\n%s\n\nПрофиль: пол=%s, возраст=%d, рост=%d см, вес=%.1f кг, жир=%.1f%%.",
		planText, p.Sex, p.Age, p.HeightCM, p.WeightKG, p.BodyFatPct,
	)

	reqBody := respReq{
		Model: a.client.model,
		Input: []respMsg{
			{Role: "user", Content: userText},
		},
		Instructions:    instructions,
		Temperature:     0,
		MaxOutputTokens: 120,
		Text: &respTextCfg{
			Format: map[string]any{
				"type":   "json_schema",
				"strict": true,
				"schema": schema,
				"name":   "activity_multiplier",
			},
		},
	}

	out, body, status, err := a.client.postResponses(ctx, reqBody)
	if err != nil {
		return 0, nil, err
	}

	if status < 200 || status >= 300 {
		return 0, map[string]any{
			"status": status,
			"body":   string(body),
		}, fmt.Errorf("openai status=%d body=%s", status, string(body))
	}

	if err := json.Unmarshal(body, &out); err != nil {
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
