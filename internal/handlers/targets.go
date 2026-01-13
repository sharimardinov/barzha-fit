package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"barzhafit/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Targets struct {
	api     *tgbotapi.BotAPI
	targets *service.TargetsService
}

func NewTargets(api *tgbotapi.BotAPI, targets *service.TargetsService) *Targets {
	return &Targets{api: api, targets: targets}
}

func (h *Targets) Handle(m *tgbotapi.Message) {
	ctx := context.Background()
	chatID := m.Chat.ID
	cmd := strings.ToLower(m.Command())

	args := strings.Fields(strings.TrimSpace(m.CommandArguments()))
	if len(args) == 0 {
		if cmd == "targetsrefresh" {
			args = []string{"refresh"}
		} else if cmd == "targetsset" {
			h.api.Send(tgbotapi.NewMessage(chatID, "Формат: /targetsset kcal|protein|fat|carbs 2600"))
			return
		} else {
			h.show(ctx, chatID)
			return
		}
	}

	if cmd == "targetsset" {
		if len(args) >= 2 {
			args = append([]string{"set"}, args...)
		} else {
			h.api.Send(tgbotapi.NewMessage(chatID, "Формат: /targetsset kcal|protein|fat|carbs 2600"))
			return
		}
	}

	switch strings.ToLower(args[0]) {
	case "refresh":
		t, err := h.targets.Refresh(ctx, chatID)
		if err != nil || t.ChatID == 0 {
			h.api.Send(tgbotapi.NewMessage(chatID, "Не могу пересчитать. Сначала /profileset ..."))
			return
		}
		h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Цели обновлены (calc): %dkcal, Б %d, Ж %d, У %d", t.Kcal, t.ProteinG, t.FatG, t.CarbsG)))
		return

	case "set":
		// /targetsset kcal 2600
		if len(args) < 3 {
			h.api.Send(tgbotapi.NewMessage(chatID, "Формат: /targetsset kcal|protein|fat|carbs 2600"))
			return
		}
		field := strings.ToLower(args[1])
		val, err := strconv.Atoi(args[2])
		if err != nil || val <= 0 {
			h.api.Send(tgbotapi.NewMessage(chatID, "Число кривое"))
			return
		}
		if err := h.targets.SetManual(ctx, chatID, field, val); err != nil {
			h.api.Send(tgbotapi.NewMessage(chatID, "Ошибка сохранения"))
			return
		}
		h.api.Send(tgbotapi.NewMessage(chatID, "Ок. Поставил вручную."))
		return

	default:
		h.api.Send(tgbotapi.NewMessage(chatID, "Команды: /targets, /targetsrefresh, /targetsset kcal 2600"))
		return
	}
}

func (h *Targets) show(ctx context.Context, chatID int64) {
	t, ok, err := h.targets.Get(ctx, chatID)
	if err != nil || !ok {
		h.api.Send(tgbotapi.NewMessage(chatID, "Целей нет. Сначала /profileset ..., затем /targetsrefresh"))
		return
	}
	h.api.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"Цели (%s):\nКалории %d\nБелок %d г\nЖир %d г\nУгли %d г",
		t.Source, t.Kcal, t.ProteinG, t.FatG, t.CarbsG,
	)))
}
