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
	"path/filepath"
	"strings"
	"time"

	"barzhafit/internal/domain"
)

type AIService struct {
	apiKey string
	model  string
	http   *http.Client
	logPath string
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
	logPath := os.Getenv("AI_LOG_PATH")
	if logPath == "" {
		logPath = "logs/ai_training.log"
	}
	return &AIService{
		apiKey: key,
		model:  model,
		logPath: logPath,
		http: &http.Client{
			Timeout: 60 * time.Second,
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
	TopP            float64      `json:"top_p,omitempty"`
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

Всегда отвечай на русском языке.

— Говори кратко, по делу, без вступлений, объяснений и описаний.
— Только логика, безопасность и результат.

Твоя задача:
— составить недельную программу тренировок
— адаптировать её под опыт, вес, цели, травмы
— игнорировать бесполезные желания пользователя, если они неэффективны
— не переоценивать восстановление, даже при использовании фармакологии

Правила:
— сначала определить количество тренировок в неделю
— затем выбрать подходящий шаблон: Фулл боди, Верх-низ, Сплит, Push-Pull
— затем определить 1–2 главных движения, 2–3 вспомогательных, 1–2 изоляции/кор

При травмах:
— при проблемах со спиной: избегать осевой нагрузки, исключить становую с пола, присед со штангой, good morning, наклоны со штангой
— использовать: жим ногами, болгарские, блочные тяги, гиперэкстензии без веса

Ограничения:
— каждый тренировочный день должен содержать 5–7 пунктов
— день отдыха — не более 1–2 активностей (мобилити, ходьба)
— если день содержит только 1–2 активности, type должен быть только "rest"
— каждый день должен содержать строго: day, name, focus, type, items
— никаких переносов строк или пустых значений внутри items
— comment всегда пустая строка ""
— никакого текста вне JSON.`

	body, _ := json.Marshal(profile)
	prompt := fmt.Sprintf(`На основе профиля атлета составь недельную программу тренировок.

Требования:
1. Определи training_days_per_week — это ограничение количества тренировок
2. Затем выбери split:
   — Фулл боди → 2–3 дня
   — Верх-низ → 4 дня
   — Сплит → 5–6 дней (если нет травм)
   — Push-Pull допускается как вариант при 4–5 днях
3. После выбора сплита — составь week_plan

Указания:
— ротировать паттерны: вертикальные/горизонтальные жимы и тяги, упражнения на ноги
— не использовать одинаковые паттерны в одном дне
— одно и то же базовое упражнение — максимум 1 раз в неделю (2 — если цель сила)
— структура тренировочного дня: 1–2 главных упражнения, 2–3 вспомогательных, 1–2 изоляции/кор
— rest-дни: 1–2 активности (например: "Ходьба 30–45 мин", "Мобилити 10 мин", "Отдых")
— если items 1–2, type должен быть "rest"; тренировки с 5–7 пунктами и type "train"
— избегай опасных упражнений при травмах (особенно поясницы и плеч)
— никакой разминки, описаний или текста — только структура
— comment всегда ""
— использовать блок normalized из профиля как источник ключевых параметров
— разрешены только упражнения и кардио из списка ниже (строгое совпадение названий)
— использовать ровно такие строки как в списке, включая сокращения и скобки

Разрешенные упражнения:
Грудь:
- Жим штанги лежа (max.)
- Жим штанги лежа на накл.ск.
- Жим лежа на накл.ск. в тренажере
- Жим гантелей лежа
- Жим гантелей лежа на накл.ск.
- Жим сидя
- Разведение гантелей лежа
- Бабочка
- Сведение рук в кроссовере

Спина:
- Тяга вертикального блока
- Тяга верт. блока обратный хват
- Тяга горизонтального блока/прям. ручка
- Гребная тяга в тренажере (черн.)
- Гребная тяга в тренажере (зел.)
- Хаммер верхнего блока
- Пулловер
- Пулл эраунд
- Тяга штанги в наклоне
- Тяга гантелей в наклоне
- Шраги со штангой
- Шраги с гантелями
- Шраги в тренажере
- Подтягивания в гравитроне
- Гиперэкстензия
- Гиперэкстензия обратная
- Тяга Т-образного грифа
- Тяга Т-образного грифа в тренажере
- Тяга нижнего Хаммера

Ноги (передняя часть):
- Приседания со штангой
- Приседания со штангой в смите
- Разгибание голени
- Выпады с гантелями
- Жим платформы ногами
- Жим платформы паралельный
- Гакк приседания
- Приседания с гантелями

Ноги (задняя часть):
- Сгибание голени
- Сгибание голени сидя
- Приседания со штангой в смите
- Жим ногами высокая пост. стопы
- Сгибание голени в кроссовере
- Мертвая тяга

Таз:
- Приседание плие
- Румынская тяга штанги
- Румынская тяга на 1 ноге
- Ягодичный мостик
- Болгарские приседания
- Разгибание бедра в кроссовере
- Приведение бедра
- Отведение бедра
- Гиперэкстензия на ягодицы

Дельты (передний пучок):
- Жим гантелей сидя
- Жим арнольда
- Армейский жим стоя
- Жим штанги сидя в смите
- Сгибание плеча в кроссовере
- Сгибание плеча с гантелями
- Обратный жим от плеч в тренажере

Дельты (средний пучок):
- Махи с гантелями стоя
- Махи с гантелями сидя
- Протяжка со штангой
- Протяжка с гантелями
- Махи в кроссовере (манжеты)

Дельты (задний пучок):
- Отведение плеча в кроссовере
- Отведение плеча в трен. бабочка
- Отведение плеча на скамье с гантел.

Бицепс:
- Сгибание предплечья Larry Scott
- Сгибание предплечья со штангой
- Сгибание предплечья с гантелями
- Сгибание предплечья в кроссовере
- Боковая тяга тросса на бицепс

Трицепс:
- Фрунцузский жим с гантелями
- Фрунцузский жим со штангой
- Жим гантели из-за головы
- Разгибание предплечья в кроссовере
- Разгибание пред. из-за голов. в крос.
- Жим лежа штанги узким хватом
- Поочередное разгибание пред. в крос.

Кардио (строго из списка):
- Бег
- Эллипс
- Ходьба
- Стэппер

Формат ответа:
JSON следующей структуры:
{
  "week_plan": [
    {
      "day": 1,
      "name": "День 1",
      "focus": "Грудь и спина",
      "type": "train",
      "items": [
        "Жим штанги лежа — 4x6-8",
        "Тяга верхнего блока — 4x6-8"
      ]
    }
  ],
  "comment": ""
}

Профиль атлета:
%s`, string(body))

	a.appendAILog("REQUEST", prompt, "")

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"week_plan": map[string]any{
				"type":     "array",
				"minItems": 7,
				"maxItems": 7,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"day":   map[string]any{"type": "integer", "minimum": 1, "maximum": 7},
						"name":  map[string]any{"type": "string"},
						"focus": map[string]any{"type": "string"},
						"type":  map[string]any{"type": "string", "enum": []string{"train", "rest"}},
						"items": map[string]any{
							"type":     "array",
							"minItems": 1,
							"maxItems": 7,
							"items":    map[string]any{"type": "string"},
						},
					},
					"required":             []string{"day", "name", "focus", "type", "items"},
					"additionalProperties": false,
				},
			},
			"comment": map[string]any{"type": "string"},
		},
		"required":             []string{"week_plan", "comment"},
		"additionalProperties": false,
	}

	reqBody := respReq{
		Model: a.model,
		Input: []respMsg{
			{Role: "user", Content: prompt},
		},
		Instructions:    instructions,
		Temperature:     0.8,
		TopP:            0.9,
		MaxOutputTokens: 1200,
		Text: &respTextCfg{
			Format: map[string]any{
				"type":   "json_schema",
				"strict": true,
				"schema": schema,
				"name":   "training_plan",
			},
		},
	}

	b, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewReader(b))
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.http.Do(httpReq)
	if err != nil {
		a.appendAILog("RESPONSE_ERROR", "", fmt.Sprintf("request_failed: %v", err))
		return "", nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyText := string(body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		a.appendAILog("RESPONSE_ERROR", bodyText, fmt.Sprintf("status=%d", resp.StatusCode))
		return "", map[string]any{
			"status": resp.StatusCode,
			"body":   bodyText,
		}, fmt.Errorf("openai status=%d body=%s", resp.StatusCode, bodyText)
	}

	var out respOut
	if err := json.Unmarshal(body, &out); err != nil {
		a.appendAILog("RESPONSE_ERROR", bodyText, fmt.Sprintf("decode_failed: %v", err))
		return "", nil, err
	}

	if out.Error != nil {
		a.appendAILog("RESPONSE_ERROR", bodyText, fmt.Sprintf("openai_error: %s", out.Error.Message))
		return "", map[string]any{"error": out.Error}, fmt.Errorf("openai error: %s", out.Error.Message)
	}

	rawText := extractOutputText(out)
	if rawText == "" {
		a.appendAILog("RESPONSE_ERROR", bodyText, "empty output_text")
		return "", out, errors.New("openai: empty output_text")
	}

	a.appendAILog("RESPONSE", rawText, "")
	return cleanJSON(rawText), out, nil
}

func (a *AIService) appendAILog(kind, payload, note string) {
	if a.logPath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(a.logPath), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(a.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	ts := time.Now().Format(time.RFC3339)
	if note != "" {
		_, _ = fmt.Fprintf(f, "[%s] %s: %s\n", ts, kind, note)
	} else {
		_, _ = fmt.Fprintf(f, "[%s] %s:\n", ts, kind)
	}
	if payload != "" {
		_, _ = fmt.Fprintln(f, payload)
	}
	_, _ = fmt.Fprintln(f, "-----")
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

func cleanJSON(raw string) string {
	raw = sanitizeJSON(raw)
	return strings.TrimSpace(raw)
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
