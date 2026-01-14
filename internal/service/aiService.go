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

	"barzhafit/internal/domain"
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

func (a *AIService) GenerateTrainingPlan(ctx context.Context, profile any) (string, any, error) {
	instructions := `Ты профессиональный силовой тренер, работающий с опытными атлетами.

Считай, что пользователь может ошибаться в своих предположениях.
Критически оценивай входные данные и отбрасывай неэффективные или опасные идеи.
Если пользователь предлагает слишком много вариантов — агрессивно фильтруй.

Говори по-русски.
Не используй вступления, приветствия и канцелярит.
Пиши прямо, коротко и по делу.

Задача:
— выбрать только рабочие упражнения
— убрать дубли и мусор
— составить недельную программу тренировок
— адаптировать её под опыт, вес, цели и травмы
— не переоценивать восстановление, даже если есть фармакология

Если есть проблемы со спиной:
— ограничь осевую нагрузку
— избегай ненужных рисков
— кратко объясни логику решений

Приоритет — результат и здоровье, а не разнообразие.

Всегда отвечай на русском языке.`

	body, _ := json.Marshal(profile)
	prompt := fmt.Sprintf(`На основе профиля атлета составь недельную программу тренировок.

Требования:
— количество тренировочных дней соответствует профилю
— исключить неэффективные и дублирующие упражнения
— учитывать травмы
— сочетать силу и гипертрофию
— не включать разминку и растяжку, если это не критично

Профиль атлета:
%s`, string(body))

	reqBody := respReq{
		Model: a.model,
		Input: []respMsg{
			{Role: "user", Content: prompt},
		},
		Instructions:    instructions,
		Temperature:     0.2,
		MaxOutputTokens: 1200,
	}

	b, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewReader(b))
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", map[string]any{
			"status": resp.StatusCode,
			"body":   string(body),
		}, fmt.Errorf("openai status=%d body=%s", resp.StatusCode, string(body))
	}

	var out respOut
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", nil, err
	}

	if out.Error != nil {
		return "", map[string]any{"error": out.Error}, fmt.Errorf("openai error: %s", out.Error.Message)
	}

	rawText := extractOutputText(out)
	if rawText == "" {
		return "", out, errors.New("openai: empty output_text")
	}

	return rawText, out, nil
}

func extractOutputText(out respOut) string {
	rawText := ""
	for _, item := range out.Output {
		for _, c := range item.Content {
			if c.Type == "output_text" && c.Text != "" {
				rawText += c.Text
			}
		}
	}
	return strings.TrimSpace(rawText)
}

func (a *AIService) EstimateActivityMultiplierWithProfile(ctx context.Context, planText string, p domain.Profile) (float64, any, error) {
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
		Model: a.model,
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

func (a *AIService) WeeklyReflection(ctx context.Context, done, total, avgKcal, avgProtein, proteinTarget int, mainIssue string) (string, error) {
	instructions := `Сделай один короткий абзац на русском. Тон сухой, без мотивации и эмодзи.
Отрази: тренировки done/total, средние калории, средний белок и главную проблему.`

	userText := fmt.Sprintf("Данные недели: тренировки %d/%d, средние калории %d, средний белок %d, цель белка %d, главный косяк: %s.",
		done, total, avgKcal, avgProtein, proteinTarget, mainIssue)

	reqBody := respReq{
		Model: a.model,
		Input: []respMsg{
			{Role: "user", Content: userText},
		},
		Instructions:    instructions,
		Temperature:     0.2,
		MaxOutputTokens: 140,
	}

	b, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewReader(b))
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.http.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai status=%d body=%s", resp.StatusCode, string(body))
	}

	var out respOut
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}

	if out.Error != nil {
		return "", fmt.Errorf("openai error: %s", out.Error.Message)
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
		return "", errors.New("openai: empty output_text")
	}

	return rawText, nil
}
