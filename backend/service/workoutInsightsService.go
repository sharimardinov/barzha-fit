package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"barzhafit/backend/domain"
)

type WorkoutInsightsStorage interface {
	StrengthExerciseNames(ctx context.Context, chatID int64, limit int) ([]string, error)
	StrengthEntriesByExerciseSince(ctx context.Context, chatID int64, exerciseName string, since time.Time, limit int) ([]domain.StrengthExerciseEntry, error)
}

type WorkoutInsightPoint struct {
	CompletedAt time.Time `json:"completedAt"`
	Weight      float64   `json:"weight"`
	Reps        int       `json:"reps"`
	Volume      float64   `json:"volume"`
	E1RM        float64   `json:"e1rm"`
}

type WorkoutInsightMetrics struct {
	Sets         int     `json:"sets"`
	Sessions     int     `json:"sessions"`
	TotalReps    int     `json:"totalReps"`
	TotalVolume  float64 `json:"totalVolume"`
	AvgWeight    float64 `json:"avgWeight"`
	AvgReps      float64 `json:"avgReps"`
	AvgVolume    float64 `json:"avgVolume"`
	MaxWeight    float64 `json:"maxWeight"`
	CurrentE1RM  float64 `json:"currentE1RM"`
	PreviousE1RM float64 `json:"previousE1RM"`
	BestE1RM     float64 `json:"bestE1RM"`
}

type WorkoutInsightTrends struct {
	E1RMDeltaPct   float64 `json:"e1rmDeltaPct"`
	VolumeDeltaPct float64 `json:"volumeDeltaPct"`
	Direction      string  `json:"direction"`
}

type WorkoutInsightAIAdvice struct {
	Summary     string `json:"summary"`
	LoadAdvice  string `json:"loadAdvice"`
	Recovery    string `json:"recovery"`
	NextSession string `json:"nextSession"`
	Confidence  int    `json:"confidence"`
}

type WorkoutExerciseInsights struct {
	Exercises        []string                `json:"exercises"`
	SelectedExercise string                  `json:"selectedExercise"`
	PeriodDays       int                     `json:"periodDays"`
	Points           []WorkoutInsightPoint   `json:"points"`
	Metrics          WorkoutInsightMetrics   `json:"metrics"`
	Trends           WorkoutInsightTrends    `json:"trends"`
	IsPlateau        bool                    `json:"isPlateau"`
	IsDowntrend      bool                    `json:"isDowntrend"`
	Recommendation   string                  `json:"recommendation"`
	Actions          []string                `json:"actions"`
	AIAvailable      bool                    `json:"aiAvailable"`
	AIAdvice         *WorkoutInsightAIAdvice `json:"aiAdvice,omitempty"`
	AIError          string                  `json:"aiError,omitempty"`
}

type WorkoutInsightsService struct {
	storage WorkoutInsightsStorage
	ai      *AIClient
	now     func() time.Time
}

func NewWorkoutInsightsService(storage WorkoutInsightsStorage, ai *AIClient) *WorkoutInsightsService {
	return &WorkoutInsightsService{
		storage: storage,
		ai:      ai,
		now:     time.Now,
	}
}

func (s *WorkoutInsightsService) StrengthExerciseInsights(ctx context.Context, chatID int64, exerciseName string, periodDays int, includeAI bool) (WorkoutExerciseInsights, error) {
	period := clampPeriodDays(periodDays)
	resp := WorkoutExerciseInsights{
		PeriodDays:  period,
		AIAvailable: s.ai != nil,
	}

	exercises, err := s.storage.StrengthExerciseNames(ctx, chatID, 100)
	if err != nil {
		return WorkoutExerciseInsights{}, err
	}
	resp.Exercises = exercises
	if len(exercises) == 0 {
		resp.Recommendation = "Пока нет завершенных силовых подходов. Сделай первую тренировку, и появится аналитика."
		resp.Actions = []string{
			"Заверши хотя бы 3 силовых подхода в одном упражнении.",
			"Используй одинаковую технику, чтобы динамика была сравнима.",
		}
		return resp, nil
	}

	selected := strings.TrimSpace(exerciseName)
	if selected == "" || !containsString(exercises, selected) {
		selected = exercises[0]
	}
	resp.SelectedExercise = selected

	since := s.now().AddDate(0, 0, -period)
	entries, err := s.storage.StrengthEntriesByExerciseSince(ctx, chatID, selected, since, 600)
	if err != nil {
		return WorkoutExerciseInsights{}, err
	}

	points := make([]WorkoutInsightPoint, 0, len(entries))
	metrics := WorkoutInsightMetrics{}
	sessionDays := make(map[string]struct{})
	totalWeight := 0.0
	for _, e := range entries {
		volume := e.Weight * float64(maxInt(e.Reps, 0))
		e1rm := estimateE1RM(e.Weight, e.Reps)
		points = append(points, WorkoutInsightPoint{
			CompletedAt: e.CompletedAt,
			Weight:      round1(e.Weight),
			Reps:        e.Reps,
			Volume:      round1(volume),
			E1RM:        round1(e1rm),
		})

		metrics.Sets++
		metrics.TotalReps += maxInt(e.Reps, 0)
		metrics.TotalVolume += volume
		totalWeight += maxFloat(e.Weight, 0)
		metrics.MaxWeight = maxFloat(metrics.MaxWeight, e.Weight)
		metrics.BestE1RM = maxFloat(metrics.BestE1RM, e1rm)
		if e1rm > 0 {
			metrics.CurrentE1RM = e1rm
		}
		dayKey := e.CompletedAt.Format("2006-01-02")
		sessionDays[dayKey] = struct{}{}
	}

	resp.Points = points
	metrics.Sessions = len(sessionDays)
	if metrics.Sets > 0 {
		metrics.AvgWeight = totalWeight / float64(metrics.Sets)
		metrics.AvgReps = float64(metrics.TotalReps) / float64(metrics.Sets)
		metrics.AvgVolume = metrics.TotalVolume / float64(metrics.Sets)
	}
	metrics.PreviousE1RM = previousE1RM(points)
	metrics.TotalVolume = round1(metrics.TotalVolume)
	metrics.AvgWeight = round1(metrics.AvgWeight)
	metrics.AvgReps = round1(metrics.AvgReps)
	metrics.AvgVolume = round1(metrics.AvgVolume)
	metrics.MaxWeight = round1(metrics.MaxWeight)
	metrics.CurrentE1RM = round1(metrics.CurrentE1RM)
	metrics.PreviousE1RM = round1(metrics.PreviousE1RM)
	metrics.BestE1RM = round1(metrics.BestE1RM)
	resp.Metrics = metrics

	e1rmDelta := estimateE1RMTrendPct(points)
	volumeDelta := estimateVolumeTrendPct(points)
	resp.Trends = WorkoutInsightTrends{
		E1RMDeltaPct:   round1(e1rmDelta),
		VolumeDeltaPct: round1(volumeDelta),
		Direction:      trendDirection(e1rmDelta),
	}

	resp.IsPlateau = metrics.Sets >= 8 && math.Abs(e1rmDelta) < 2 && math.Abs(volumeDelta) < 8
	resp.IsDowntrend = metrics.Sets >= 6 && e1rmDelta <= -4
	resp.Recommendation, resp.Actions = buildRuleRecommendation(resp)

	if includeAI && s.ai != nil && len(points) > 0 {
		aiAdvice, err := s.generateAIAdvice(ctx, resp)
		if err != nil {
			resp.AIError = err.Error()
		} else {
			resp.AIAdvice = &aiAdvice
		}
	}

	return resp, nil
}

func (s *WorkoutInsightsService) generateAIAdvice(ctx context.Context, insights WorkoutExerciseInsights) (WorkoutInsightAIAdvice, error) {
	lastPoints := insights.Points
	if len(lastPoints) > 8 {
		lastPoints = lastPoints[len(lastPoints)-8:]
	}
	compact := make([]map[string]any, 0, len(lastPoints))
	for _, p := range lastPoints {
		compact = append(compact, map[string]any{
			"date":   p.CompletedAt.Format("2006-01-02"),
			"weight": p.Weight,
			"reps":   p.Reps,
			"e1rm":   p.E1RM,
			"volume": p.Volume,
		})
	}

	payload := map[string]any{
		"exercise":      insights.SelectedExercise,
		"period_days":   insights.PeriodDays,
		"metrics":       insights.Metrics,
		"trends":        insights.Trends,
		"is_plateau":    insights.IsPlateau,
		"is_downtrend":  insights.IsDowntrend,
		"last_sessions": compact,
	}
	payloadJSON, _ := json.Marshal(payload)

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"summary": map[string]any{
				"type":      "string",
				"minLength": 8,
				"maxLength": 220,
			},
			"loadAdvice": map[string]any{
				"type":      "string",
				"minLength": 8,
				"maxLength": 220,
			},
			"recovery": map[string]any{
				"type":      "string",
				"minLength": 8,
				"maxLength": 220,
			},
			"nextSession": map[string]any{
				"type":      "string",
				"minLength": 8,
				"maxLength": 220,
			},
			"confidence": map[string]any{
				"type":    "integer",
				"minimum": 1,
				"maximum": 100,
			},
		},
		"required":             []string{"summary", "loadAdvice", "recovery", "nextSession", "confidence"},
		"additionalProperties": false,
	}

	reqBody := respReq{
		Model: s.ai.model,
		Input: []respMsg{
			{Role: "user", Content: string(payloadJSON)},
		},
		Instructions: `Ты силовой тренер. Дай краткий практичный разбор динамики упражнения.
Правила:
- Пиши только на русском.
- Без воды и морализаторства.
- Не предлагай фармакологию и рискованные практики.
- Рекомендации должны быть конкретны для следующей тренировки.
- Верни только JSON по схеме.`,
		Temperature:     0.2,
		MaxOutputTokens: 260,
		Text: &respTextCfg{
			Format: map[string]any{
				"type":   "json_schema",
				"strict": true,
				"schema": schema,
				"name":   "workout_insights",
			},
		},
	}

	out, body, status, err := s.ai.postResponses(ctx, reqBody)
	if err != nil {
		return WorkoutInsightAIAdvice{}, err
	}
	if status < 200 || status >= 300 {
		return WorkoutInsightAIAdvice{}, fmt.Errorf("openai status=%d", status)
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return WorkoutInsightAIAdvice{}, err
	}
	if out.Error != nil {
		return WorkoutInsightAIAdvice{}, errors.New(out.Error.Message)
	}
	rawText := extractOutputText(out)
	if rawText == "" {
		return WorkoutInsightAIAdvice{}, errors.New("openai empty output")
	}
	var advice WorkoutInsightAIAdvice
	if err := json.Unmarshal([]byte(rawText), &advice); err != nil {
		return WorkoutInsightAIAdvice{}, err
	}
	return advice, nil
}

func buildRuleRecommendation(insights WorkoutExerciseInsights) (string, []string) {
	m := insights.Metrics
	if m.Sets == 0 {
		return "За выбранный период данных нет.", []string{
			"Расширь период до 180 дней или выполни подходы в этом упражнении.",
		}
	}
	if insights.IsDowntrend {
		return "Есть спад силовых показателей. Нужна разгрузка и восстановление.", []string{
			"Снизь рабочий вес на 7-10% на 1 неделю.",
			"Сохрани технику и темп, не работай до отказа каждый подход.",
			"Проверь сон и восстановление между тренировками.",
		}
	}
	if insights.IsPlateau {
		return "Похоже на плато: нагрузка почти не меняется и сила стоит на месте.", []string{
			"Добавь малую прогрессию: +2.5 кг или +1 повтор в одном из рабочих подходов.",
			"Сделай 1 тяжелый и 1 объемный день для этого упражнения.",
			"Если плато держится 3+ недели, сделай разгрузочную неделю.",
		}
	}
	if insights.Trends.E1RMDeltaPct >= 3 {
		return "Динамика положительная, прогресс идет в хорошем темпе.", []string{
			"Сохраняй текущую схему и добавляй нагрузку постепенно.",
			"Повышай вес только при чистой технике и стабильных повторах.",
		}
	}
	return "Динамика нейтральная: явного прогресса или спада нет.", []string{
		"Проверь регулярность: 2-3 качественных экспозиции на упражнение в неделю.",
		"Добавь микро-прогрессию по весу или повторам на ближайшие 2 тренировки.",
	}
}

func previousE1RM(points []WorkoutInsightPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	cut := len(points) / 2
	if cut <= 0 {
		cut = 1
	}
	prev := 0.0
	for i := 0; i < cut; i++ {
		prev = maxFloat(prev, points[i].E1RM)
	}
	return prev
}

func estimateE1RMTrendPct(points []WorkoutInsightPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	cut := len(points) / 2
	if cut <= 0 || cut >= len(points) {
		return pct(points[len(points)-1].E1RM, points[0].E1RM)
	}
	prevMax := 0.0
	for i := 0; i < cut; i++ {
		prevMax = maxFloat(prevMax, points[i].E1RM)
	}
	currMax := 0.0
	for i := cut; i < len(points); i++ {
		currMax = maxFloat(currMax, points[i].E1RM)
	}
	return pct(currMax, prevMax)
}

func estimateVolumeTrendPct(points []WorkoutInsightPoint) float64 {
	if len(points) < 4 {
		return pct(points[len(points)-1].Volume, points[0].Volume)
	}
	win := len(points) / 4
	if win < 2 {
		win = 2
	}
	if win*2 > len(points) {
		win = len(points) / 2
	}
	if win <= 0 {
		return 0
	}
	prevAvg := 0.0
	for _, p := range points[len(points)-2*win : len(points)-win] {
		prevAvg += p.Volume
	}
	prevAvg /= float64(win)
	currAvg := 0.0
	for _, p := range points[len(points)-win:] {
		currAvg += p.Volume
	}
	currAvg /= float64(win)
	return pct(currAvg, prevAvg)
}

func trendDirection(v float64) string {
	switch {
	case v > 2:
		return "up"
	case v < -2:
		return "down"
	default:
		return "flat"
	}
}

func estimateE1RM(weight float64, reps int) float64 {
	if weight <= 0 || reps <= 0 {
		return 0
	}
	return weight * (1 + float64(reps)/30.0)
}

func pct(curr, prev float64) float64 {
	if prev <= 0 {
		if curr <= 0 {
			return 0
		}
		return 100
	}
	return ((curr - prev) / prev) * 100
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func clampPeriodDays(v int) int {
	if v <= 0 {
		return 90
	}
	if v < 14 {
		return 14
	}
	if v > 365 {
		return 365
	}
	return v
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
