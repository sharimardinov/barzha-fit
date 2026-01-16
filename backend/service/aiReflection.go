package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type ReflectionAI struct {
	client *AIClient
}

func NewReflectionAI(client *AIClient) *ReflectionAI {
	return &ReflectionAI{client: client}
}

func (a *ReflectionAI) WeeklyReflection(ctx context.Context, done, total, avgKcal, avgProtein, proteinTarget int, mainIssue string) (string, error) {
	instructions := `Сделай один короткий абзац на русском. Тон сухой, без мотивации и эмодзи.
Отрази: тренировки done/total, средние калории, средний белок и главную проблему.`

	userText := fmt.Sprintf("Данные недели: тренировки %d/%d, средние калории %d, средний белок %d, цель белка %d, главный косяк: %s.",
		done, total, avgKcal, avgProtein, proteinTarget, mainIssue)

	reqBody := respReq{
		Model: a.client.model,
		Input: []respMsg{
			{Role: "user", Content: userText},
		},
		Instructions:    instructions,
		Temperature:     0.2,
		MaxOutputTokens: 140,
	}

	out, body, status, err := a.client.postResponses(ctx, reqBody)
	if err != nil {
		return "", err
	}

	if status < 200 || status >= 300 {
		return "", fmt.Errorf("openai status=%d body=%s", status, string(body))
	}

	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}

	if out.Error != nil {
		return "", fmt.Errorf("openai error: %s", out.Error.Message)
	}

	rawText := extractOutputText(out)
	if rawText == "" {
		return "", errors.New("openai: empty output_text")
	}

	return rawText, nil
}
