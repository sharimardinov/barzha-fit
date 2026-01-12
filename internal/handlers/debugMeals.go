package handlers

import (
	"barzhafit/internal/storage/db"
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DebugMeals struct {
	api *tgbotapi.BotAPI
	db  *pgxpool.Pool
}

func NewDebugMeals(api *tgbotapi.BotAPI, db *pgxpool.Pool) *DebugMeals {
	return &DebugMeals{api: api, db: db}
}

func (h *DebugMeals) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID

	rows, err := h.db.Query(ctx, `
		select id, eaten_at, text, kcal, protein_g, fat_g, carbs_g,
		       eaten_at AT TIME ZONE 'UTC' as eaten_at_utc
		from meals
		where chat_id=$1
		order by id desc
		limit 10
	`, chatID)
	if err != nil {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка: %v", err)))
		return
	}
	defer rows.Close()

	var items []db.Meal
	var timestamps []string
	for rows.Next() {
		var m db.Meal
		var eatenAtUTC time.Time
		if err := rows.Scan(&m.ID, &m.EatenAt, &m.Text, &m.Kcal, &m.ProteinG, &m.FatG, &m.CarbsG, &eatenAtUTC); err != nil {
			continue
		}
		items = append(items, m)
		timestamps = append(timestamps, fmt.Sprintf("Local: %s\nUTC: %s",
			m.EatenAt.Format("2006-01-02 15:04:05 MST"),
			eatenAtUTC.Format("2006-01-02 15:04:05 MST")))
	}

	if len(items) == 0 {
		_, _ = h.api.Send(tgbotapi.NewMessage(chatID, "Вообще нет записей в базе"))
		return
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("🔍 Найдено записей: %d\n\n", len(items)))
	b.WriteString(fmt.Sprintf("Сейчас: %s\n\n", time.Now().Format("2006-01-02 15:04:05 MST")))

	for i, it := range items {
		b.WriteString(fmt.Sprintf("#%d [ID=%d]\n", i+1, it.ID))
		b.WriteString(fmt.Sprintf("%s\n", timestamps[i]))
		b.WriteString(fmt.Sprintf("%dkcal (Б%d Ж%d У%d)\n", it.Kcal, it.ProteinG, it.FatG, it.CarbsG))
		b.WriteString(fmt.Sprintf("Текст: %s\n\n", it.Text))
	}

	_, _ = h.api.Send(tgbotapi.NewMessage(chatID, b.String()))
}
